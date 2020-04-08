package slack

import (
	"fmt"
	"strings"

	"github.com/nais/device/apiserver/database"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

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

func (s *slackbot) registrationSlackHandler() {
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

			publicKey, serial, err := parseRegisterMessage(msg.Text)
			if err != nil {
				log.Errorf("parsing message: %v", err)
				break
			}

			email, err := s.getUserEmail(msg.User)
			if err != nil {
				log.Errorf("getting user email: %v", err)
				break
			}

			log.Infof("email: %v, publicKey: %v, serial: %v", email, publicKey, serial)

			err = s.database.AddClient(msg.Username, publicKey, serial)
			if err != nil {
				log.Errorf("adding client to database: %v", err)
				rtm.SendMessage(rtm.NewOutgoingMessage("Something went wrong during registration :sweat_smile:, I've notified the nais device team for you.", msg.Channel))
			} else {
				rtm.SendMessage(rtm.NewOutgoingMessage("Successfully registered :partyparrot:", msg.Channel))
			}

		case *slack.InvalidAuthEvent:
			log.Fatalf("slack auth failed: %v", message)
		}
	}
}

func parseRegisterMessage(text string) (string, string, error) {
	// "register publicKey serial"
	parts := strings.Split(text, " ")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("parsing register command: not enough params: \"%v\"", text)
	}
	command, publicKey, serial := parts[0], parts[1], parts[2]
	if command != "register" {
		return "", "", fmt.Errorf("parsing register command: invalid command: \"%v\"", command)
	}
	return publicKey, serial, nil
}

func (s *slackbot) getUserEmail(userID string) (string, error) {
	if info, err := s.api.GetUserInfo(userID); err != nil {
		return "", fmt.Errorf("getting user info: %w", err)
	} else {
		return info.Profile.Email, nil
	}
}

func (s *slackbot) Run() {
	go s.registrationSlackHandler()
}
