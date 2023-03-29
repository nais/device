package runtimeconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/nais/device/pkg/bearertransport"
	"github.com/nais/device/pkg/pubsubenroll"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/nais/device/pkg/bootstrap"
	"github.com/nais/device/pkg/device-agent/auth"
	"github.com/nais/device/pkg/device-agent/bootstrapper"
	"github.com/nais/device/pkg/device-agent/config"
	"github.com/nais/device/pkg/device-agent/wireguard"
	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
)

const (
	tenantDiscoveryBucket = "naisdevice-enroll-discovery"
)

type RuntimeConfig struct {
	EnrollConfig *bootstrap.Config // TODO: convert to enroll.Config
	Config       *config.Config
	PrivateKey   []byte
	Tokens       *auth.Tokens
	Tenants      []*pb.Tenant
}

func (rc *RuntimeConfig) GetActiveTenant() *pb.Tenant {
	for _, tenant := range rc.Tenants {
		if tenant.Active {
			return tenant
		}
	}
	return nil
}

func New(cfg *config.Config) (*RuntimeConfig, error) {
	rc := &RuntimeConfig{
		Config: cfg,
	}

	var err error

	if rc.PrivateKey, err = wireguard.EnsurePrivateKey(rc.Config.PrivateKeyPath); err != nil {
		return nil, fmt.Errorf("ensuring private key: %w", err)
	}

	log.Infof("Runtime config initialized with public key: %s", wireguard.PublicKey(rc.PrivateKey))

	return rc, nil
}

func (r *RuntimeConfig) EnsureEnrolled(ctx context.Context, serial string) error {
	log.Infoln("Enrolling device")

	var err error
	if r.GetActiveTenant().AuthProvider == pb.AuthProvider_Google {
		err = r.enrollGoogle(ctx, serial)
	} else {
		err = r.enrollAzure(ctx, serial)
	}

	if err != nil {
		return fmt.Errorf("enroll device: %w", err)
	}

	return r.SaveEnrollConfig()
}

func (r *RuntimeConfig) enrollAzure(ctx context.Context, serial string) error {
	client := bearertransport.Transport{AccessToken: r.Tokens.Token.AccessToken}.Client()

	cfg, err := bootstrapper.BootstrapDevice(
		ctx,
		&bootstrap.DeviceInfo{
			PublicKey: string(wireguard.PublicKey(r.PrivateKey)),
			Serial:    serial,
			Platform:  r.Config.Platform,
		},
		r.Config.BootstrapAPI,
		client,
	)
	if err != nil {
		return err
	}
	r.EnrollConfig = cfg

	return nil
}

func (r *RuntimeConfig) enrollGoogle(ctx context.Context, serial string) error {
	req := &pubsubenroll.DeviceRequest{
		Platform:           r.Config.Platform,
		Serial:             serial,
		WireGuardPublicKey: wireguard.PublicKey(r.PrivateKey),
	}

	buf := &bytes.Buffer{}
	_ = json.NewEncoder(buf).Encode(req)

	url, err := r.getEnrollURL(ctx)
	if err != nil {
		return fmt.Errorf("determine enroll url from token: %w", err)
	}

	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, buf)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	hreq.Header.Set("Content-Type", "application/json")
	hreq.Header.Set("Authorization", "Bearer "+r.Tokens.IDToken)

	hresp, err := http.DefaultClient.Do(hreq)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}

	if hresp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(hresp.Body)
		if err != nil {
			return fmt.Errorf("reading response body: %w", err)
		}

		return fmt.Errorf("got status code %d from device enrollment service: %v", hresp.StatusCode, string(body))
	}

	resp := &pubsubenroll.Response{}
	if err := json.NewDecoder(hresp.Body).Decode(resp); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	apiserverPeer := findPeer(resp.Peers, "apiserver")

	r.EnrollConfig = &bootstrap.Config{
		DeviceIP:       resp.WireGuardIP,
		PublicKey:      apiserverPeer.PublicKey,
		TunnelEndpoint: apiserverPeer.Endpoint,
		APIServerIP:    apiserverPeer.Ip,
	}
	return nil
}

func (r *RuntimeConfig) SaveEnrollConfig() error {
	f, err := os.OpenFile(r.path(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(r.EnrollConfig)
}

func (r *RuntimeConfig) LoadEnrollConfig() error {
	path := r.path()
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading bootstrap config from %q: %w", path, err)
	}

	ec := &bootstrap.Config{}
	if err := json.Unmarshal(b, ec); err != nil {
		return err
	}

	r.EnrollConfig = ec
	return nil
}

func findPeer(gateway []*pb.Gateway, s string) *pb.Gateway {
	for _, gw := range gateway {
		if gw.Name == s {
			return gw
		}
	}

	return nil
}

func (r *RuntimeConfig) getEnrollURL(ctx context.Context) (string, error) {
	domain, err := r.getPartnerDomain()
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", tenantDiscoveryBucket, domain)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(b) + "/enroll", nil
}

func (r *RuntimeConfig) getPartnerDomain() (string, error) {
	if r.GetActiveTenant().Domain != "" {
		return r.GetActiveTenant().Domain, nil
	}

	if r.Tokens != nil {
		t, err := jwt.ParseString(r.Tokens.IDToken)
		if err != nil {
			return "", fmt.Errorf("parse token: %w", err)
		}

		hd, _ := t.Get("hd")

		return hd.(string), nil
	} else {
		return "", fmt.Errorf("unable to identify tenant domain")
	}
}

func (r *RuntimeConfig) path() string {
	domain, err := r.getPartnerDomain()
	if err != nil {
		log.WithError(err).Error("could not determine partner domain")
		domain = "unknown"
	}

	filename := fmt.Sprintf("enroll-%s.json", domain)
	return filepath.Join(r.Config.ConfigDir, filename)
}

func (r *RuntimeConfig) PopulateTenants(ctx context.Context) error {
	client, err := storage.NewClient(ctx, option.WithoutAuthentication())
	if err != nil {
		return err
	}

	bucket := client.Bucket(tenantDiscoveryBucket)

	objs := bucket.Objects(ctx, &storage.Query{})

	for {
		obj, err := objs.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return err
		}

		if obj == nil {
			break
		}

		r.Tenants = append(r.Tenants, &pb.Tenant{
			Name:         obj.Name,
			AuthProvider: pb.AuthProvider_Google,
			Domain:       obj.Name,
		})

	}

	return nil
}

func (rc *RuntimeConfig) SetTenantSession(session *pb.Session) error {
	if tenant := rc.GetActiveTenant(); tenant != nil {
		tenant.Session = session
		return nil
	}

	return fmt.Errorf("no active tenant. tenants: %+v", rc.Tenants)
}

func (rc *RuntimeConfig) GetTenantSession() (*pb.Session, error) {
	if tenant := rc.GetActiveTenant(); tenant != nil {
		return tenant.Session, nil
	}

	return nil, fmt.Errorf("no active tenant. tenants: %+v", rc.Tenants)
}

func (rc *RuntimeConfig) GetToken(ctx context.Context) (string, error) {
	if rc.Tokens == nil {
		return "", fmt.Errorf("no tokens in runtimeconfig")
	}

	if rc.GetActiveTenant().AuthProvider == pb.AuthProvider_Google {
		return rc.Tokens.IDToken, nil
	}
	return rc.Tokens.Token.AccessToken, nil
}
