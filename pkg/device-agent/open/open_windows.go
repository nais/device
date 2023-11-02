package open

import (
	"os/exec"
)

func Open(url string) {
	go func() {
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Run()
	}()
}
