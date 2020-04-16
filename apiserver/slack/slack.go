package slack

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nais/device/apiserver/database"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

const Usage = `enroll <token>`

type slackbot struct {
	api            *slack.Client
	database       *database.APIServerDB
	tunnelEndpoint string
}

type EnrollmentConfig struct {
	DeviceIP       string `json:"deviceIP"`
	PublicKey      string `json:"publicKey"`
	TunnelEndpoint string `json:"tunnelEndpoint"`
	APIServerIP    string `json:"apiServerIP"`
}

func New(token, tunnelEndpoint string, database *database.APIServerDB) *slackbot {
	return &slackbot{
		api:            slack.New(token),
		tunnelEndpoint: tunnelEndpoint,
		database:       database,
	}
}

func (s *slackbot) handleEnroll(msg slack.Msg) string {
	parts := strings.Split(msg.Text, " ")
	if len(parts) != 3 {
		return fmt.Sprintf("invalid command format, usage:\n%v", Usage)
	}

	publicKey, serial := parts[1], parts[2]
	email, err := s.getUserEmail(msg.User)
	if err != nil {
		log.Errorf("getting user email: %v", err)
		return "unable to find email for your slack user :confused:, I've notified the nais device team for you."
	}

	err = s.database.AddDevice(email, publicKey, serial)
	if err != nil {
		log.Errorf("adding device to database: %v", err)
		return "Something went wrong during enrollment :sweat_smile:, I've notified the nais device team for you. (1)"
	} else {
		c, err := s.database.ReadDevice(publicKey)
		if err != nil {
			return "Something went wrong during enrollment :sweat_smile:, I've notified the nais device team for you. (2)"
		}

		ec := EnrollmentConfig{
			DeviceIP:       c.IP,
			PublicKey:      c.PublicKey,
			TunnelEndpoint: s.tunnelEndpoint,
			APIServerIP:    "10.255.240.1",
		}

		b, err := json.Marshal(&ec)
		if err != nil {
			return "Something went wrong during enrollment :sweat_smile:, I've notified the nais device team for you. (3)"
		}

		token := base64.StdEncoding.EncodeToString(b)
		return fmt.Sprintf("Successfully enrolled :partyparrot:, copy and paste this command on your command line: `sudo tee /usr/local/etc/nais-device/enrollment.token <<< '%s'`", token)
	}
}

func (s *slackbot) handleMsg(msg slack.Msg) string {
	parts := strings.Split(msg.Text, " ")
	if len(parts) == 0 {
		return fmt.Sprintf("unable to parse input, usage:\n%v", Usage)
	}

	switch parts[0] {
	case "enroll":
		return s.handleEnroll(msg)
	default:
		log.Debugf("Unrecognized command: %v", msg.Text)
		return fmt.Sprintf("unrecognized command, usage:\n%v", Usage)
	}
}

func (s *slackbot) Handler() {
	log.SetLevel(log.DebugLevel)
	rtm := s.api.NewRTM()

	go rtm.ManageConnection()

	for message := range rtm.IncomingEvents {
		switch ev := message.Data.(type) {

		case *slack.ConnectedEvent:
			log.Infof("Connected to %v as %v via %s", ev.Info.Team.Name, ev.Info.User.Name, ev.Info.URL)

		case *slack.RTMError:
			log.Errorf("Error: %s\n", ev.Error())

		case *slack.MessageEvent:
			msg := ev.Msg

			if msg.SubType != "" {
				break
			}

			log.Debugf("MessageEvent msg: %v", msg)
			response := s.handleMsg(msg)
			rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))

		case *slack.InvalidAuthEvent:
			log.Fatalf("slack auth failed: %v", message)
		}
	}
}

func (s *slackbot) getUserEmail(userID string) (string, error) {
	if info, err := s.api.GetUserInfo(userID); err != nil {
		return "", fmt.Errorf("getting user info: %w", err)
	} else {
		return info.Profile.Email, nil
	}
}
