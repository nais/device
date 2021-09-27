package azure

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

	"github.com/go-chi/jwtauth/v5"
	"github.com/golang-jwt/jwt"
	log "github.com/sirupsen/logrus"
)

type CertificateList []*x509.Certificate

type Azure struct {
	DiscoveryURL string
	ClientID     string
}

func CreateJWTValidator(azure Azure) (jwt.Keyfunc, error) {
	if len(azure.ClientID) == 0 || len(azure.DiscoveryURL) == 0 {
		return nil, fmt.Errorf("missing required azure configuration")
	}

	certificates, err := FetchCertificates(azure)
	if err != nil {
		return nil, fmt.Errorf("retrieving azure ad certificates for token validation: %v", err)
	}

	return JWTValidator(certificates, azure.ClientID), nil
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

func TokenValidatorMiddleware(jwtValidator jwt.Keyfunc) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			var claims jwt.MapClaims

			token := jwtauth.TokenFromHeader(r)

			_, err := jwt.ParseWithClaims(token, &claims, jwtValidator)
			if err != nil {
				log.Errorf("parsing token: %v", err)
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
