package slack

import (
	"fmt"
	"strings"

	"github.com/nais/device/apiserver/database"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

const Usage = `register publicKey serialNumber`

type slackbot struct {
	api      *slack.Client
	database *database.APIServerDB
}

func New(token string, database *database.APIServerDB) *slackbot {
	return &slackbot{
		api:      slack.New(token),
		database: database,
	}
}

func (s *slackbot) handleRegister(msg slack.Msg) string {
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

	err = s.database.AddClient(email, publicKey, serial)
	if err != nil {
		log.Errorf("adding client to database: %v", err)
		return "Something went wrong during registration :sweat_smile:, I've notified the nais device team for you."
	} else {
		return "Successfully registered :partyparrot:"
	}
}

func (s *slackbot) handleMsg(msg slack.Msg) string {
	parts := strings.SplitN(msg.Text, " ", 1)
	if len(parts) == 0 {
		return fmt.Sprintf("unable to parse input, usage:\n%v", Usage)
	}

	switch parts[0] {
	case "register":
		return s.handleRegister(msg)
	default:
		return fmt.Sprintf("unrecognized command, usage:\n%v", Usage)
	}
}

func (s *slackbot) slackHandler() {
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

func (s *slackbot) Run() {
	go s.slackHandler()
}
