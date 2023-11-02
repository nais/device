package open

import (
	"os/exec"
)

func Open(url string) {
	go func() {
		exec.Command("open", url).Run()
	}()
}
