package runtimeconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/nais/device/pkg/bearertransport"
	"github.com/nais/device/pkg/pubsubenroll"

	"github.com/nais/device/pkg/bootstrap"
	"github.com/nais/device/pkg/device-agent/auth"
	"github.com/nais/device/pkg/device-agent/bootstrapper"
	"github.com/nais/device/pkg/device-agent/config"
	"github.com/nais/device/pkg/device-agent/wireguard"
	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
)

type RuntimeConfig struct {
	BootstrapConfig *bootstrap.Config
	Config          *config.Config
	PrivateKey      []byte
	SessionInfo     *pb.Session
	Tokens          *auth.Tokens
}

func New(cfg *config.Config) (*RuntimeConfig, error) {
	rc := &RuntimeConfig{
		Config: cfg,
	}

	var err error

	if rc.PrivateKey, err = wireguard.EnsurePrivateKey(rc.Config.PrivateKeyPath); err != nil {
		return nil, fmt.Errorf("ensuring private key: %w", err)
	}

	rc.BootstrapConfig, err = readBootstrapConfigFromFile(rc.Config.BootstrapConfigPath)
	if err != nil {
		log.Infof("Unable to read bootstrap config from file: %v", err)
	} else {
		log.Infof("Read bootstrap config from file: %v", rc.Config.BootstrapConfigPath)
	}

	log.Infof("Runtime config initialized with public key: %s", wireguard.PublicKey(rc.PrivateKey))

	return rc, nil
}

func EnsureBootstrapping(ctx context.Context, rc *RuntimeConfig, serial string) (cfg *bootstrap.Config, err error) {
	log.Infoln("Bootstrapping device")

	if rc.Config.EnableGoogleAuth {
		cfg, err = googleBootstrap(ctx, rc, serial)
	} else {
		cfg, err = azureBootstrap(ctx, rc, serial)
	}

	if err != nil {
		return nil, fmt.Errorf("bootstrapping device: %w", err)
	}

	return cfg, writeToJSONFile(cfg, rc.Config.BootstrapConfigPath)
}

func azureBootstrap(ctx context.Context, rc *RuntimeConfig, serial string) (*bootstrap.Config, error) {
	client := bearertransport.Transport{AccessToken: rc.Tokens.Token.AccessToken}.Client()

	cfg, err := bootstrapper.BootstrapDevice(
		ctx,
		&bootstrap.DeviceInfo{
			PublicKey: string(wireguard.PublicKey(rc.PrivateKey)),
			Serial:    serial,
			Platform:  rc.Config.Platform,
		},
		rc.Config.BootstrapAPI,
		client,
	)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func googleBootstrap(ctx context.Context, rc *RuntimeConfig, serial string) (*bootstrap.Config, error) {
	req := &pubsubenroll.DeviceRequest{
		Platform:           rc.Config.Platform,
		Serial:             serial,
		WireGuardPublicKey: wireguard.PublicKey(rc.PrivateKey),
	}

	buf := &bytes.Buffer{}
	_ = json.NewEncoder(buf).Encode(req)
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://naisdevice-device-enroller-wvjph2xazq-lz.a.run.app/enroll", buf)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	hreq.Header.Set("Content-Type", "application/json")
	hreq.Header.Set("Authorization", "Bearer "+rc.Tokens.IDToken)

	hresp, err := http.DefaultClient.Do(hreq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	if hresp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(hresp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading response body: %w", err)
		}

		return nil, fmt.Errorf("got status code %d from device enrollment service: %v", hresp.StatusCode, string(body))
	}

	resp := &pubsubenroll.Response{}
	if err := json.NewDecoder(hresp.Body).Decode(resp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	apiserverPeer := findPeer(resp.Peers, "apiserver")
	return &bootstrap.Config{
		DeviceIP:       resp.WireGuardIP,
		PublicKey:      apiserverPeer.PublicKey,
		TunnelEndpoint: apiserverPeer.Endpoint,
		APIServerIP:    apiserverPeer.Ip,
	}, nil
}

func writeToJSONFile(strct any, path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(strct)
}

func readBootstrapConfigFromFile(bootstrapConfigPath string) (*bootstrap.Config, error) {
	var bc bootstrap.Config
	b, err := ioutil.ReadFile(bootstrapConfigPath)
	if err != nil {
		return nil, fmt.Errorf("reading bootstrap config from disk: %w", err)
	}
	if err := json.Unmarshal(b, &bc); err != nil {
		return nil, fmt.Errorf("unmarshaling bootstrap config: %w", err)
	}
	return &bc, nil
}

func findPeer(gateway []*pb.Gateway, s string) *pb.Gateway {
	for _, gw := range gateway {
		if gw.Name == s {
			return gw
		}
	}

	return nil
}
