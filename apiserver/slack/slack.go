package slack

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/nais/device/apiserver/database"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

const (
	Usage            = `enroll <token>`
	InternalErrorMsg = "Ahhh, we've messed up something here on our side :meow-disappointed: I've notified my team with the error details."
)

type slackbot struct {
	api                *slack.Client
	database           *database.APIServerDB
	tunnelEndpoint     string
	apiServerPublicKey string
	jwtValidator       jwt.Keyfunc
}

// BootstrapConfig is the information the device needs to bootstrap it's connection to the APIServer
type BootstrapConfig struct {
	DeviceIP       string `json:"deviceIP"`
	PublicKey      string `json:"publicKey"`
	TunnelEndpoint string `json:"tunnelEndpoint"`
	APIServerIP    string `json:"apiServerIP"`
}

// EnrollmentConfig is the information sent by the device during enrollment
type EnrollmentConfig struct {
	Serial      string `json:"serial"`
	PublicKey   string `json:"publicKey"`
	AccessToken string `json:"accessToken"`
}

func New(token, tunnelEndpoint string, database *database.APIServerDB, apiServerPublicKey string, validator jwt.Keyfunc) *slackbot {
	return &slackbot{
		api:                slack.New(token),
		tunnelEndpoint:     tunnelEndpoint,
		database:           database,
		apiServerPublicKey: apiServerPublicKey,
		jwtValidator:       validator,
	}
}

func (s *slackbot) handleEnroll(msg slack.Msg) string {
	parts := strings.Split(msg.Text, " ")
	if len(parts) != 2 {
		return fmt.Sprintf("invalid command format, usage:\n%v", Usage)
	}

	token := parts[1]
	enrollmentConfig, err := ParseEnrollmentToken(token)
	if err != nil {
		log.Errorf("Unable to parse enrollment token: %v", err)
		return "There is something wrong with your token :sadkek: Make sure you copied it correctly. If it still doesn't work, get help in #nais-device channel."
	}

	tokenEmail, err := s.ExtractEmailFromToken(enrollmentConfig.AccessToken)
	if err != nil {
		log.Errorf("Extracting email from access token: %v", err)
		return InternalErrorMsg
	}

	slackUserEmail, err := s.getUserEmail(msg.User)
	if err != nil {
		log.Errorf("Getting user email: %v", err)
		return "Unable to find e-mail for your slack user :confused:, I've notified the nais device team for you."
	}

	if strings.ToLower(tokenEmail) != strings.ToLower(slackUserEmail) {
		log.Errorf("email address in token did not match slack user, something is 🐟'y")
		return "Unable to authorize enrollment. Ensure you are using the same account on Slack and when authorizing the device-agent."
	}

	err = s.database.AddDevice(slackUserEmail, enrollmentConfig.PublicKey, enrollmentConfig.Serial)
	if err != nil {
		log.Errorf("Adding device to database: %v", err)
		return InternalErrorMsg
	} else {
		c, err := s.database.ReadDevice(enrollmentConfig.PublicKey)
		if err != nil {
			log.Errorf("Reading device info from database: %v", err)
			return InternalErrorMsg
		}

		bc := BootstrapConfig{
			DeviceIP:       c.IP,
			PublicKey:      s.apiServerPublicKey,
			TunnelEndpoint: s.tunnelEndpoint,
			APIServerIP:    "10.255.240.1",
		}

		b, err := json.Marshal(&bc)
		if err != nil {
			return InternalErrorMsg
		}

		token := base64.StdEncoding.EncodeToString(b)
		return fmt.Sprintf("Successfully enrolled :kekw: Copy and paste this command on your command line:\n ```echo '%s' > ~/.config/nais-device/bootstrap.token```", token)
	}
}

func (s *slackbot) ExtractEmailFromToken(tokenString string) (string, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenString, &claims, s.jwtValidator)

	if err != nil {
		return "", fmt.Errorf("parsing token: %w", err)
	}

	return claims["preferred_username"].(string), nil
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
		return "", fmt.Errorf("getting user info for %v: %w", userID, err)
	} else {
		return info.Profile.Email, nil
	}
}

func ParseEnrollmentToken(enrollmentToken string) (*EnrollmentConfig, error) {
	b, err := base64.StdEncoding.DecodeString(enrollmentToken)
	if err != nil {
		return nil, fmt.Errorf("decoding base64: %w", err)
	}

	var enrollmentConfig EnrollmentConfig
	err = json.Unmarshal(b, &enrollmentConfig)
	if err != nil {
		return nil, fmt.Errorf("decoding base64: %w", err)
	}

	return &enrollmentConfig, nil
}
