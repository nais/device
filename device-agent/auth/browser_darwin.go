package auth

import "os/exec"

func openDefaultBrowser(url string) error {
	command := exec.Command("open", url)
	return command.Start()
}
