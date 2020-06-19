package browser

import "os/exec"

func OpenDefaultBrowser(url string) error {
	command := exec.Command("open", url)
	return command.Start()
}
