package open

import (
	"fmt"
	"os/exec"
)

func Open(url string) {
	go func() {
		if err := exec.Command("open", url).Run(); err != nil {
			fmt.Println("Failed to open URL:", err)
		}
	}()
}
