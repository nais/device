package open

import "os/exec"

func Open(url string) error {
	command := exec.Command("open", url)
	return command.Start()
}
