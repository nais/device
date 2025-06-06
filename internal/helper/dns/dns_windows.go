package dns

import (
	"fmt"
	"os/exec"
	"strings"
)

func apply(zones []string) error {
	cmd := exec.Command("powershell", "-nologo", "-noprofile")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	go func() {
		defer stdin.Close()
		for _, zone := range []string{"nais.io", "nav.no"} {
			fmt.Fprintf(stdin, "Get-DnsClientNrptRule | Where { $_.Namespace -match '%s' } | Remove-DnsClientNrptRule -force\n", zone)
		}
		for _, zone := range zones {
			if !strings.HasPrefix(zone, ".") {
				zone = "." + zone
			}
			fmt.Fprintf(stdin, "Get-DnsClientNrptRule | Where { $_.Namespace -match '%s' } | Remove-DnsClientNrptRule -force\n", zone)
			fmt.Fprintf(stdin, "Add-DnsClientNrptRule -Namespace '%s' -NameServers @('8.8.8.8','8.8.4.4')\n", zone)
		}
	}()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running powershell: %s: %s", err, string(out))
	}

	return nil
}
