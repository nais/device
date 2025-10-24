package open

import (
	"fmt"
	"os/exec"
)

func Open(url string) {
	go func() {
		if err := exec.Command("xdg-open", url).Run(); err != nil {
			// Fallback to using "gio open" if "xdg-open" fails
			fmt.Printf("xdg-open failed: %v", err)
		}
	}()
}
