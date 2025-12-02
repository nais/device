package open

import (
	"fmt"
	"os/exec"
)

func Open(url string) {
	go func() {
		if err := exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Run(); err != nil {
			fmt.Println("Failed to open URL:", err)
		}
	}()
}
