package wireguard

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/nais/device/pkg/apiserver/config"
	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/pb"
)

type WireGuard interface {
	Sync(ctx context.Context) error
}

type wireguard struct {
	cfg        config.Config
	db         database.APIServer
	privateKey string
}

func New(config config.Config, db database.APIServer, privateKey string) WireGuard {
	return &wireguard{
		cfg:        config,
		privateKey: privateKey,
		db:         db,
	}
}

func (w wireguard) Sync(ctx context.Context) error {
	log.Debug("Synchronizing configuration")
	devices, err := w.db.ReadDevices(ctx)
	if err != nil {
		return fmt.Errorf("reading devices from database: %w", err)
	}

	gateways, err := w.db.ReadGateways(ctx)
	if err != nil {
		return fmt.Errorf("reading gateways from database: %w", err)
	}

	wgConfigContent := generateWGConfig(devices, gateways, w.privateKey, w.cfg)

	if err := os.WriteFile(w.cfg.WireGuardConfigPath, wgConfigContent, 0o600); err != nil {
		return fmt.Errorf("writing WireGuard config to disk: %w", err)
	} else {
		log.Debugf("Successfully wrote WireGuard config to: %v", w.cfg.WireGuardConfigPath)
	}

	syncConf := exec.CommandContext(ctx, "wg", "syncconf", "wg0", w.cfg.WireGuardConfigPath)
	b, err := syncConf.CombinedOutput()
	if err != nil {
		return fmt.Errorf("synchronizing WireGuard config: %w: %v", err, string(b))
	}

	return nil
}

// TODO: merge with marshal code in device-agent
func generateWGConfig(devices []*pb.Device, gateways []*pb.Gateway, privateKey string, conf config.Config) []byte {
	interfaceTemplate := `[Interface]
PrivateKey = %s
ListenPort = 51820

`

	wgConfig := fmt.Sprintf(interfaceTemplate, strings.TrimSuffix(privateKey, "\n"))

	peerTemplate := `[Peer]
AllowedIPs = %s/32
PublicKey = %s
`
	wgConfig += fmt.Sprintf(peerTemplate, conf.PrometheusTunnelIP, conf.PrometheusPublicKey)

	for _, device := range devices {
		wgConfig += fmt.Sprintf(peerTemplate, device.Ip, device.PublicKey)
	}

	for _, gateway := range gateways {
		wgConfig += fmt.Sprintf(peerTemplate, gateway.Ip, gateway.PublicKey)
	}

	return []byte(wgConfig)
}
