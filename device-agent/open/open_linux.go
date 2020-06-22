package open

import "os/exec"

func Open(url string) error {
	command := exec.Command("xdg-open", url)
	return command.Start()
}
