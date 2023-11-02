package open

import (
	"os/exec"
)

func Open(url string) error {
	go func() {
		exec.Command("open", url).Run()
	}()
}
