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
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/nais/device/internal/bootstrap"
	"github.com/nais/device/internal/device-agent/auth"
	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/wireguard"
	"github.com/nais/device/internal/enroll"
	"github.com/nais/device/internal/ioconvenience"
	"github.com/nais/device/internal/otel"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	grpcstatus "google.golang.org/grpc/status"
)

const (
	tenantDiscoveryBucket = "naisdevice-enroll-discovery"
	apiserverDialTimeout  = 1 * time.Second // sleep time between failed configuration syncs
)

type RuntimeConfig interface {
	DialAPIServer(context.Context) (*grpc.ClientConn, error)
	APIServerPeer() *pb.Gateway
	ConnectToAPIServer(context.Context) (pb.APIServerClient, func(), error)

	EnsureEnrolled(context.Context, string) error
	ResetEnrollConfig()
	LoadEnrollConfig() error
	SaveEnrollConfig() error

	Tenants() []*pb.Tenant
	GetActiveTenant() *pb.Tenant
	SetActiveTenant(string) error
	PopulateTenants(context.Context) error

	GetTenantSession() (*pb.Session, error)

	GetDomainFromToken() string
	GetToken(context.Context) (string, error)
	SetToken(*auth.Tokens)
	SetTenantSession(*pb.Session) error

	BuildHelperConfiguration(peers []*pb.Gateway) *pb.Configuration

	WithAPIServer(func(pb.APIServerClient, string) error) error
	SetAPIServerInfo(pb.APIServerClient, string)
}

var _ RuntimeConfig = &runtimeConfig{}

type runtimeConfig struct {
	enrollConfig *bootstrap.Config // TODO: convert to enroll.Config
	config       *config.Config
	privateKey   []byte
	tokens       *auth.Tokens
	tenants      []*pb.Tenant
	log          *logrus.Entry

	apiserverClient pb.APIServerClient
	apiserverKey    string
	apiserverLock   sync.RWMutex
}

func (rc *runtimeConfig) SetAPIServerInfo(apiServerClient pb.APIServerClient, key string) {
	rc.apiserverLock.Lock()
	defer rc.apiserverLock.Unlock()

	rc.apiserverClient = apiServerClient
	rc.apiserverKey = key
}

func (rc *runtimeConfig) WithAPIServer(f func(pb.APIServerClient, string) error) error {
	rc.apiserverLock.RLock()
	defer rc.apiserverLock.RUnlock()

	return f(rc.apiserverClient, rc.apiserverKey)
}

func (rc *runtimeConfig) DialAPIServer(ctx context.Context) (*grpc.ClientConn, error) {
	rc.log.WithField("address", rc.apiServerGRPCAddress()).Info("setting up gRPC connection to apiserver")
	return grpc.NewClient(
		rc.apiServerGRPCAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.Second * 10,
			Timeout:             time.Second * 2,
			PermitWithoutStream: false,
		}),
		grpc.WithStatsHandler(otel.NewGRPCClientHandler(
			pb.APIServer_GetDeviceConfiguration_FullMethodName,
		)),
	)
}

func (rc *runtimeConfig) APIServerPeer() *pb.Gateway {
	return rc.enrollConfig.APIServerPeer()
}

func (rc *runtimeConfig) ConnectToAPIServer(ctx context.Context) (pb.APIServerClient, func(), error) {
	dialContext, cancel := context.WithTimeout(ctx, apiserverDialTimeout)

	conn, err := rc.DialAPIServer(dialContext)
	if err != nil {
		cancel()
		return nil, func() {}, grpcstatus.Errorf(codes.Unavailable, "dial: %v", err)
	}

	return pb.NewAPIServerClient(conn), func() {
		cancel()
		_ = conn.Close()
	}, nil
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

func New(log *logrus.Entry, cfg *config.Config) (RuntimeConfig, error) {
	rc := &runtimeConfig{
		config:  cfg,
		tenants: defaultTenants,
		log:     log,
	}

	var err error

	if rc.privateKey, err = wireguard.EnsurePrivateKey(rc.config.PrivateKeyPath); err != nil {
		return nil, fmt.Errorf("ensuring private key: %w", err)
	}

	rc.log.WithField("public_key", wireguard.PublicKey(rc.privateKey)).Info("runtime config initialized")

	return rc, nil
}

func (r *runtimeConfig) EnsureEnrolled(ctx context.Context, serial string) error {
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
	req := &enroll.DeviceRequest{
		Platform:           r.config.Platform,
		Serial:             serial,
		WireGuardPublicKey: wireguard.PublicKey(r.privateKey),
	}

	buf := &bytes.Buffer{}
	_ = json.NewEncoder(buf).Encode(req)

	url, err := r.getEnrollURL(ctx)
	if err != nil {
		return fmt.Errorf("determine enroll url: %w", err)
	}

	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, buf)
	if err != nil {
		return fmt.Errorf("creating enroll request: %w", err)
	}

	hreq.Header.Set("Content-Type", "application/json")
	hreq.Header.Set("Authorization", "Bearer "+token)

	hresp, err := http.DefaultClient.Do(hreq)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = hresp.Body.Close() }()

	if hresp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(hresp.Body)
		if err != nil {
			return fmt.Errorf("reading response body: %w", err)
		}

		return fmt.Errorf("got status code %d from device enrollment service: %v", hresp.StatusCode, string(body))
	}

	resp := &enroll.Response{}
	if err := json.NewDecoder(hresp.Body).Decode(resp); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	apiserverPeer := findPeer(resp.Peers, "apiserver")
	if apiserverPeer == nil {
		return fmt.Errorf("enrollment peer not found")
	}

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
	if r.GetDomainFromToken() == "mock" || r.config.CustomEnrollURL != "" {
		return nil
	}

	f, err := os.OpenFile(r.path(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer ioconvenience.CloseWithLog(r.log, f)

	return json.NewEncoder(f).Encode(r.enrollConfig)
}

func (r *runtimeConfig) LoadEnrollConfig() error {
	if r.enrollConfig != nil {
		return nil
	}

	if r.GetDomainFromToken() == "mock" {
		r.enrollConfig = &bootstrap.Config{
			DeviceIPv4:     "",
			APIServerIP:    "127.0.0.1",
			DeviceIPv6:     "",
			PublicKey:      "",
			TunnelEndpoint: "",
		}
		return nil
	}

	if r.config.CustomEnrollURL != "" {
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

	// IPv6 is only enabled in NAV
	if strings.EqualFold(r.GetActiveTenant().Name, "NAV") {
		if ec.DeviceIPv6 == "" || strings.HasPrefix(ec.DeviceIPv6, "fd") {
			return fmt.Errorf("bootstrap config does not contain a valid IPv6 address, should re-enroll")
		}
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
	if r.config.CustomEnrollURL != "" {
		return r.config.CustomEnrollURL, nil
	}
	domain := r.GetDomainFromToken()

	url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", tenantDiscoveryBucket, domain)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(b) + "/enroll", nil
}

func (r *runtimeConfig) GetDomainFromToken() string {
	if r.config.LocalAPIServer {
		return "mock"
	}

	if r.GetActiveTenant().Domain != "" {
		return r.GetActiveTenant().Domain
	}

	if r.tokens != nil && r.tokens.IDToken != "" {
		t, err := jwt.ParseString(r.tokens.IDToken, jwt.WithValidate(false))
		if err == nil {
			hd, _ := t.Get("hd")
			return hd.(string)
		} else {
			r.log.WithError(err).Warn("parse id token")
		}
	}

	return "default"
}

func (r *runtimeConfig) path() string {
	domain := r.GetDomainFromToken()
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
