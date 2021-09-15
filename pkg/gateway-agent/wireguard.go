package gateway_agent

import (
	"fmt"
	"github.com/nais/device/pkg/pb"
	"io/ioutil"
	"os/exec"
	"regexp"

	log "github.com/sirupsen/logrus"
)

func SetupInterface(tunnelIP string) error {
	if err := exec.Command("ip", "link", "del", "wg0").Run(); err != nil {
		log.Infof("pre-deleting WireGuard interface (ok if this fails): %v", err)
	}

	run := func(commands [][]string) error {
		for _, s := range commands {
			cmd := exec.Command(s[0], s[1:]...)

			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("running %v: %w: %v", cmd, err, string(out))
			} else {
				log.Debugf("%v: %v\n", cmd, string(out))
			}
		}
		return nil
	}

	commands := [][]string{
		{"ip", "link", "add", "dev", "wg0", "type", "wireguard"},
		{"ip", "link", "set", "wg0", "mtu", "1360"},
		{"ip", "address", "add", "dev", "wg0", tunnelIP + "/21"},
		{"ip", "link", "set", "wg0", "up"},
	}

	return run(commands)
}

func GenerateBaseConfig(cfg Config) string {
	template := `[Interface]
PrivateKey = %s
ListenPort = 51820

[Peer] # apiserver
PublicKey = %s
AllowedIPs = %s/32
Endpoint = %s

[Peer] # prometheus
PublicKey = %s
AllowedIPs = %s/32

`

	return fmt.Sprintf(template, cfg.PrivateKey, cfg.BootstrapConfig.PublicKey, cfg.BootstrapConfig.APIServerIP, cfg.BootstrapConfig.TunnelEndpoint, cfg.PrometheusPublicKey, cfg.PrometheusTunnelIP)
}

func GenerateWireGuardPeers(devices []*pb.Device) string {
	peerTemplate := `[Peer]
PublicKey = %s
AllowedIPs = %s
`
	var peers string

	for _, device := range devices {
		peers += fmt.Sprintf(peerTemplate, device.PublicKey, device.Ip)
	}

	return peers
}

// ActuateWireGuardConfig runs syncconfig with the provided WireGuard config
func ActuateWireGuardConfig(wireGuardConfig, wireGuardConfigPath string) error {
	if err := ioutil.WriteFile(wireGuardConfigPath, []byte(wireGuardConfig), 0600); err != nil {
		return fmt.Errorf("writing WireGuard config to disk: %w", err)
	}

	cmd := exec.Command("wg", "syncconf", "wg0", wireGuardConfigPath)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running syncconf: %w", err)
	}

	log.Debugf("Actuated WireGuard config: %v", wireGuardConfigPath)

	return nil
}

func ConnectedDeviceCount() (int, error) {
	output, err := exec.Command("wg", "show", "wg0", "endpoints").Output()
	if err != nil {
		return 0, err
	}

	re := regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d{1,5}`)
	matches := re.FindAll(output, -1)

	return len(matches), nil
}
