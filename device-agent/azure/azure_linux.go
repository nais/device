package azure

import "os/exec"

func openDefaultBrowser(url string) error {
	command := exec.Command("xdg-open", url)
	return command.Start()
}
