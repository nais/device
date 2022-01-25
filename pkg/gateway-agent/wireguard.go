package gateway_agent

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"

	"github.com/nais/device/pkg/pb"

	log "github.com/sirupsen/logrus"
)

func (nc *networkConfigurer) SetupInterface() error {
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
		{"ip", "address", "add", "dev", "wg0", nc.config.DeviceIP + "/21"},
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

	return fmt.Sprintf(
		template,
		cfg.PrivateKey,
		cfg.APIServerPublicKey,
		cfg.APIServerPrivateIP,
		cfg.APIServerEndpoint,
		cfg.PrometheusPublicKey,
		cfg.PrometheusTunnelIP,
	)
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

// ApplyWireGuardConfig runs syncconfig with the provided WireGuard config
func (nc *networkConfigurer) ApplyWireGuardConfig(devices []*pb.Device) error {
	wireGuardConfig := fmt.Sprintf(
		"%s%s",
		GenerateBaseConfig(nc.config),
		GenerateWireGuardPeers(devices),
	)

	if err := ioutil.WriteFile(nc.config.WireGuardConfigPath, []byte(wireGuardConfig), 0600); err != nil {
		return fmt.Errorf("writing WireGuard config to disk: %w", err)
	}

	cmd := exec.Command("wg", "syncconf", "wg0", nc.config.WireGuardConfigPath)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running syncconf: %w", err)
	}

	log.Debugf("Actuated WireGuard config: %v", nc.config.WireGuardConfigPath)

	return nil
}

func (nc *networkConfigurer) ConnectedDeviceCount() (int, error) {
	output, err := exec.Command("wg", "show", "wg0", "endpoints").Output()
	if err != nil {
		return 0, err
	}

	re := regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d{1,5}`)
	matches := re.FindAll(output, -1)

	return len(matches), nil
}
