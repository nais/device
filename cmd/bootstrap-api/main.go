package main

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth"
	"github.com/nais/device/pkg/bootstrap"

	"github.com/dgrijalva/jwt-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

type CertificateList []*x509.Certificate

type Azure struct {
	DiscoveryURL string
	ClientID     string
}

type Config struct {
	BindAddress         string
	Azure               Azure
	PrometheusAddr      string
	PrometheusPublicKey string
	PrometheusTunnelIP  string
	CredentialEntries   []string
}

var cfg = &Config{
	Azure: Azure{
		ClientID:     "",
		DiscoveryURL: "",
	},
	CredentialEntries: nil,
	BindAddress:       ":8080",
	PrometheusAddr:    ":3000",
}

var enrollments ActiveEnrollments

func init() {
	log.SetFormatter(&log.JSONFormatter{})

	flag.StringVar(&cfg.PrometheusAddr, "prometheus-address", cfg.PrometheusAddr, "prometheus listen address")
	flag.StringVar(&cfg.BindAddress, "bind-address", cfg.BindAddress, "Bind address")
	flag.StringVar(&cfg.Azure.DiscoveryURL, "azure-discovery-url", "", "Azure discovery url")
	flag.StringVar(&cfg.Azure.ClientID, "azure-client-id", "", "Azure app client id")
	flag.StringSliceVar(&cfg.CredentialEntries, "credential-entries", nil, "Comma-separated credentials on format: '<user>:<key>'")

	flag.Parse()
}

func main() {
	enrollments.init()

	go func() {
		log.Infof("Prometheus serving metrics at %v", cfg.PrometheusAddr)
		_ = http.ListenAndServe(cfg.PrometheusAddr, promhttp.Handler())
	}()

	jwtValidator, err := createJWTValidator(cfg.Azure)
	if err != nil {
		log.Fatalf("Creating JWT validator: %v", err)
	}
	r := chi.NewRouter()
	r.Get("/isalive", func(w http.ResponseWriter, r *http.Request) {
		return
	})
	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(TokenValidatorMiddleware(jwtValidator))
			r.Post("/deviceinfo", postDeviceInfo)
			r.Get("/bootstrapconfig/{serial}", getBootstrapConfig)
		})

		r.Group(func(r chi.Router) {
			// TODO r.Use(BasicAuthValidator())
			r.Get("/deviceinfo", getDeviceInfos)
			r.Post("/bootstrapconfig/{serial}", postBootstrapConfig)
		})
	})

	log.Info("running @", cfg.BindAddress)
	log.Info(http.ListenAndServe(cfg.BindAddress, r))
}

func postBootstrapConfig(w http.ResponseWriter, r *http.Request) {
	serial := chi.URLParam(r, "serial")

	var bootstrapConfig bootstrap.Config
	err := json.NewDecoder(r.Body).Decode(&bootstrapConfig)
	if err != nil {
		log.Errorf("Decoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	enrollments.addBootstrapConfig(serial, bootstrapConfig)
	log.Infof("%s %s: serial: %s bootstrapconfig: %v", r.Method, r.URL, serial, bootstrapConfig)

	w.WriteHeader(http.StatusCreated)
}

func getBootstrapConfig(w http.ResponseWriter, r *http.Request) {
	serial := chi.URLParam(r, "serial")

	bootstrapConfig := enrollments.getBootstrapConfig(serial)
	log.Infof("%s %s: serial: %s bootstrapconfig: %v", r.Method, r.URL, serial, bootstrapConfig)

	if bootstrapConfig == nil {
		w.WriteHeader(http.StatusNotFound)
		log.Warnf("no bootstrap config for serial: %v", serial)
		return
	}

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(bootstrapConfig)
	if err != nil {
		log.Errorf("Encoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func postDeviceInfo(w http.ResponseWriter, r *http.Request) {
	var deviceInfo bootstrap.DeviceInfo
	err := json.NewDecoder(r.Body).Decode(&deviceInfo)

	if err != nil {
		log.Errorf("Decoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	deviceInfo.Owner = r.Context().Value("preferred_username").(string)
	if len(deviceInfo.Owner) == 0 {
		log.Errorf("deviceInfo without owner, abort enroll")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	enrollments.addDeviceInfo(deviceInfo)
	log.Infof("%s %s: %v", r.Method, r.URL, deviceInfo)
	w.WriteHeader(http.StatusCreated)
}

func getDeviceInfos(w http.ResponseWriter, r *http.Request) {
	deviceInfos := enrollments.getDeviceInfos()
	log.Infof("%s %s: %v", r.Method, r.URL, deviceInfos)

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(deviceInfos)
	if err != nil {
		log.Errorf("Encoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type ActiveEnrollments struct {
	deviceInfos     []bootstrap.DeviceInfo
	deviceInfosLock sync.Mutex

	bootstrapConfigs     map[string]bootstrap.Config
	bootstrapConfigsLock sync.Mutex
}

func (a *ActiveEnrollments) init() {
	a.bootstrapConfigs = make(map[string]bootstrap.Config)
}

func (a *ActiveEnrollments) getDeviceInfos() []bootstrap.DeviceInfo {
	a.deviceInfosLock.Lock()
	defer a.deviceInfosLock.Unlock()

	var deviceInfosToReturn []bootstrap.DeviceInfo
	deviceInfosToReturn = append(deviceInfosToReturn, a.deviceInfos...)

	a.deviceInfos = nil

	return deviceInfosToReturn
}

func (a *ActiveEnrollments) addDeviceInfo(deviceInfo bootstrap.DeviceInfo) {
	a.deviceInfosLock.Lock()
	defer a.deviceInfosLock.Unlock()

	a.deviceInfos = append(a.deviceInfos, deviceInfo)
}

func (a *ActiveEnrollments) addBootstrapConfig(serial string, bootstrapConfig bootstrap.Config) {
	a.bootstrapConfigsLock.Lock()
	defer a.bootstrapConfigsLock.Unlock()

	a.bootstrapConfigs[serial] = bootstrapConfig
}

func (a *ActiveEnrollments) getBootstrapConfig(serial string) *bootstrap.Config {
	a.bootstrapConfigsLock.Lock()
	defer a.bootstrapConfigsLock.Unlock()

	if val, ok := a.bootstrapConfigs[serial]; ok {
		delete(a.bootstrapConfigs, serial)
		return &val
	} else {
		return nil
	}
}

func createJWTValidator(azure Azure) (jwt.Keyfunc, error) {
	if len(azure.ClientID) == 0 || len(azure.DiscoveryURL) == 0 {
		return nil, fmt.Errorf("missing required azure configuration")
	}

	certificates, err := FetchCertificates(cfg.Azure)
	if err != nil {
		return nil, fmt.Errorf("retrieving azure ad certificates for token validation: %v", err)
	}

	return JWTValidator(certificates, cfg.Azure.ClientID), nil
}

func FetchCertificates(azure Azure) (map[string]CertificateList, error) {
	log.Infof("Discover Microsoft signing certificates from %s", azure.DiscoveryURL)
	azureKeyDiscovery, err := DiscoverURL(azure.DiscoveryURL)
	if err != nil {
		return nil, err
	}

	log.Infof("Decoding certificates for %d keys", len(azureKeyDiscovery.Keys))
	azureCertificates, err := azureKeyDiscovery.Map()
	if err != nil {
		return nil, err
	}
	return azureCertificates, nil
}

func (c *Config) Credentials() (map[string]string, error) {
	credentials := make(map[string]string)
	for _, key := range c.CredentialEntries {
		entry := strings.Split(key, ":")
		if len(entry) > 2 {
			return nil, fmt.Errorf("invalid format on credentials, should be comma-separated entries on format 'user:key'")
		}

		credentials[entry[0]] = entry[1]
	}

	return credentials, nil
}

func TokenValidatorMiddleware(jwtValidator jwt.Keyfunc) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			var claims jwt.MapClaims

			token := jwtauth.TokenFromHeader(r)

			_, err := jwt.ParseWithClaims(token, &claims, jwtValidator)
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				_, err = fmt.Fprintf(w, "Unauthorized access: %s", err.Error())
				if err != nil {
					log.Errorf("Writing http response: %v", err)
				}
				return
			}

			var groups []string
			groupInterface := claims["groups"].([]interface{})
			groups = make([]string, len(groupInterface))
			for i, v := range groupInterface {
				groups[i] = v.(string)
			}
			r = r.WithContext(context.WithValue(r.Context(), "groups", groups))

			username := claims["preferred_username"].(string)
			r = r.WithContext(context.WithValue(r.Context(), "preferred_username", username))

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func JWTValidator(certificates map[string]CertificateList, audience string) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		var certificateList CertificateList
		var kid string
		var ok bool

		if claims, ok := token.Claims.(*jwt.MapClaims); !ok {
			return nil, fmt.Errorf("unable to retrieve claims from token")
		} else {
			if valid := claims.VerifyAudience(audience, true); !valid {
				return nil, fmt.Errorf("the token is not valid for this application")
			}
		}

		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		if kid, ok = token.Header["kid"].(string); !ok {
			return nil, fmt.Errorf("field 'kid' is of invalid type %T, should be string", token.Header["kid"])
		}

		if certificateList, ok = certificates[kid]; !ok {
			return nil, fmt.Errorf("kid '%s' not found in certificate list", kid)
		}

		for _, certificate := range certificateList {
			return certificate.PublicKey, nil
		}

		return nil, fmt.Errorf("no certificate candidates for kid '%s'", kid)
	}
}

type EncodedCertificate string

type KeyDiscovery struct {
	Keys []Key `json:"keys"`
}

type Key struct {
	Kid string               `json:"kid"`
	X5c []EncodedCertificate `json:"x5c"`
}

func DiscoverURL(url string) (*KeyDiscovery, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	return Discover(response.Body)
}

func Discover(reader io.Reader) (*KeyDiscovery, error) {
	document, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	keyDiscovery := &KeyDiscovery{}
	err = json.Unmarshal(document, keyDiscovery)

	return keyDiscovery, err
}

// Transform a KeyDiscovery object into a dictionary with "kid" as key
// and lists of decoded X509 certificates as values.
//
// Returns an error if any certificate does not decode.
func (k *KeyDiscovery) Map() (result map[string]CertificateList, err error) {
	result = make(map[string]CertificateList)

	for _, key := range k.Keys {
		certList := make(CertificateList, 0)
		for _, encodedCertificate := range key.X5c {
			certificate, err := encodedCertificate.Decode()
			if err != nil {
				return nil, err
			}
			certList = append(certList, certificate)
		}
		result[key.Kid] = certList
	}

	return
}

// Decode a base64 encoded certificate into a X509 structure.
func (c EncodedCertificate) Decode() (*x509.Certificate, error) {
	stream := strings.NewReader(string(c))
	decoder := base64.NewDecoder(base64.StdEncoding, stream)
	key, err := ioutil.ReadAll(decoder)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(key)
}
