package auth

import "os/exec"

func openDefaultBrowser(url string) error {
	command := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	return command.Start()
}
