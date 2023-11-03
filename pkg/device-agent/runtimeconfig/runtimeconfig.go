package runtimeconfig

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/nais/device/pkg/pubsubenroll"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nais/device/pkg/bootstrap"
	"github.com/nais/device/pkg/device-agent/auth"
	"github.com/nais/device/pkg/device-agent/config"
	"github.com/nais/device/pkg/device-agent/wireguard"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
)

const (
	tenantDiscoveryBucket = "naisdevice-enroll-discovery"
)

type RuntimeConfig interface {
	DialAPIServer(context.Context) (*grpc.ClientConn, error)
	APIServerPeer() *pb.Gateway

	EnsureEnrolled(context.Context, string) error
	ResetEnrollConfig()
	LoadEnrollConfig() error
	SaveEnrollConfig() error

	Tenants() []*pb.Tenant
	GetActiveTenant() *pb.Tenant
	SetActiveTenant(string) error
	PopulateTenants(context.Context) error

	GetTenantSession() (*pb.Session, error)

	GetToken(context.Context) (string, error)
	SetToken(*auth.Tokens)
	SetTenantSession(*pb.Session) error

	BuildHelperConfiguration(peers []*pb.Gateway) *pb.Configuration
}

var _ RuntimeConfig = &runtimeConfig{}

type runtimeConfig struct {
	enrollConfig *bootstrap.Config // TODO: convert to enroll.Config
	config       *config.Config
	privateKey   []byte
	tokens       *auth.Tokens
	tenants      []*pb.Tenant
	log          *logrus.Entry
}

func (rc *runtimeConfig) DialAPIServer(ctx context.Context) (*grpc.ClientConn, error) {
	rc.log.Infof("Attempting gRPC connection to API server on %s...", rc.apiServerGRPCAddress())
	return grpc.DialContext(
		ctx,
		rc.apiServerGRPCAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithReturnConnectionError(),
	)
}

func (rc *runtimeConfig) APIServerPeer() *pb.Gateway {
	return rc.enrollConfig.APIServerPeer()
}

func (rc *runtimeConfig) BuildHelperConfiguration(peers []*pb.Gateway) *pb.Configuration {
	return &pb.Configuration{
		PrivateKey: base64.StdEncoding.EncodeToString(rc.privateKey),
		DeviceIPv4: rc.enrollConfig.DeviceIPv4,
		DeviceIPv6: rc.enrollConfig.DeviceIPv6,
		Gateways:   peers,
	}
}

func (rc *runtimeConfig) Tenants() []*pb.Tenant {
	return rc.tenants
}

func (rc *runtimeConfig) SetActiveTenant(name string) error {
	// Mark all tenants inactive
	for i := range rc.tenants {
		rc.tenants[i].Active = false
	}

	for i, tenant := range rc.tenants {
		if strings.EqualFold(tenant.Name, name) {
			rc.tenants[i].Active = true
			return nil
		}
	}

	return fmt.Errorf("tenant not found")
}

func (rc *runtimeConfig) GetActiveTenant() *pb.Tenant {
	for _, tenant := range rc.tenants {
		if tenant.Active {
			return tenant
		}
	}
	return nil
}

func (rc *runtimeConfig) apiServerGRPCAddress() string {
	return net.JoinHostPort(rc.enrollConfig.APIServerIP, "8099")
}

func New(log *logrus.Entry, cfg *config.Config) (*runtimeConfig, error) {
	rc := &runtimeConfig{
		config:  cfg,
		tenants: defaultTenants,
		log:     log,
	}

	var err error

	if rc.privateKey, err = wireguard.EnsurePrivateKey(rc.config.PrivateKeyPath); err != nil {
		return nil, fmt.Errorf("ensuring private key: %w", err)
	}

	rc.log.Infof("Runtime config initialized with public key: %s", wireguard.PublicKey(rc.privateKey))

	return rc, nil
}

func (r *runtimeConfig) EnsureEnrolled(ctx context.Context, serial string) error {
	r.log.Infoln("Enrolling device")

	var err error
	if r.GetActiveTenant().AuthProvider == pb.AuthProvider_Google {
		err = r.enroll(ctx, serial, r.tokens.IDToken)
	} else {
		err = r.enroll(ctx, serial, r.tokens.Token.AccessToken)
	}

	if err != nil {
		return fmt.Errorf("enroll device: %w", err)
	}

	return r.SaveEnrollConfig()
}

func (r *runtimeConfig) enroll(ctx context.Context, serial, token string) error {
	req := &pubsubenroll.DeviceRequest{
		Platform:           r.config.Platform,
		Serial:             serial,
		WireGuardPublicKey: wireguard.PublicKey(r.privateKey),
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
	hreq.Header.Set("Authorization", "Bearer "+token)

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

	r.enrollConfig = &bootstrap.Config{
		DeviceIPv4:     resp.WireGuardIPv4,
		DeviceIPv6:     resp.WireGuardIPv6,
		PublicKey:      apiserverPeer.PublicKey,
		TunnelEndpoint: apiserverPeer.Endpoint,
		APIServerIP:    apiserverPeer.Ipv4,
	}
	return nil
}

func (r *runtimeConfig) SaveEnrollConfig() error {
	f, err := os.OpenFile(r.path(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(r.enrollConfig)
}

func (r *runtimeConfig) LoadEnrollConfig() error {
	if r.enrollConfig != nil {
		return nil
	}

	path := r.path()
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading bootstrap config from %q: %w", path, err)
	}

	ec := &bootstrap.Config{}
	if err := json.Unmarshal(b, ec); err != nil {
		return err
	}

	if ec.DeviceIPv6 == "" || strings.HasPrefix(ec.DeviceIPv6, "fd") {
		return fmt.Errorf("bootstrap config does not contain a valid IPv6 address, should re-enroll")
	}

	r.enrollConfig = ec
	return nil
}

func (rc *runtimeConfig) ResetEnrollConfig() {
	rc.enrollConfig = nil
}

func findPeer(gateway []*pb.Gateway, s string) *pb.Gateway {
	for _, gw := range gateway {
		if gw.Name == s {
			return gw
		}
	}

	return nil
}

func (r *runtimeConfig) getEnrollURL(ctx context.Context) (string, error) {
	domain, err := r.getPartnerDomain()
	if err != nil {
		r.log.WithError(err).Error("could not determine partner domain, falling back to default")
		domain = "default"
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

func (r *runtimeConfig) getPartnerDomain() (string, error) {
	if r.GetActiveTenant().Domain != "" {
		return r.GetActiveTenant().Domain, nil
	}

	if r.tokens != nil {
		t, err := jwt.ParseString(r.tokens.IDToken)
		if err != nil {
			return "", fmt.Errorf("parse token: %w", err)
		}

		hd, _ := t.Get("hd")

		return hd.(string), nil
	} else {
		return "", fmt.Errorf("unable to identify tenant domain")
	}
}

func (r *runtimeConfig) path() string {
	domain, err := r.getPartnerDomain()
	if err != nil {
		r.log.WithError(err).Error("could not determine partner domain")
		domain = "unknown"
	}

	filename := fmt.Sprintf("enroll-%s.json", domain)
	return filepath.Join(r.config.ConfigDir, filename)
}

func (r *runtimeConfig) PopulateTenants(ctx context.Context) error {
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

		r.tenants = append(r.tenants, &pb.Tenant{
			Name:         obj.Name,
			AuthProvider: pb.AuthProvider_Google,
			Domain:       obj.Name,
			Active:       false,
		})

	}

	// set first tenant as active by default
	r.tenants[0].Active = true

	return nil
}

func (rc *runtimeConfig) SetTenantSession(session *pb.Session) error {
	if tenant := rc.GetActiveTenant(); tenant != nil {
		tenant.Session = session
		return nil
	}

	return fmt.Errorf("no active tenant. tenants: %+v", rc.tenants)
}

func (rc *runtimeConfig) GetTenantSession() (*pb.Session, error) {
	if tenant := rc.GetActiveTenant(); tenant != nil {
		return tenant.Session, nil
	}

	return nil, fmt.Errorf("no active tenant. tenants: %+v", rc.tenants)
}

func (rc *runtimeConfig) SetToken(token *auth.Tokens) {
	rc.tokens = token
}

func (rc *runtimeConfig) GetToken(ctx context.Context) (string, error) {
	if rc.tokens == nil {
		return "", fmt.Errorf("no tokens in runtimeconfig")
	}

	if rc.GetActiveTenant().AuthProvider == pb.AuthProvider_Google {
		return rc.tokens.IDToken, nil
	}

	return rc.tokens.Token.AccessToken, nil
}
