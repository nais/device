package open

import (
	"os/exec"

	log "github.com/sirupsen/logrus"
)

func Open(url string) {
	go func() {
		command := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		if err := command.Run(); err != nil {
			log.Errorf("open.Open(%q): %v", url, err)
		}
	}()
}
