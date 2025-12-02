package open

import (
	"fmt"
	"os/exec"
)

func Open(url string) {
	go func() {
		if err := exec.Command("xdg-open", url).Run(); err != nil {
			fmt.Printf("xdg-open failed: %v", err)
		}
	}()
}
