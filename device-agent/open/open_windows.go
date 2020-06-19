package open

import "os/exec"

func Open(url string) error {
	command := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	return command.Start()
}
