package apiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Gateways []*Gateway

type Gateway struct {
	PublicKey string   `json:"publicKey"`
	Endpoint  string   `json:"endpoint"`
	IP        string   `json:"ip"`
	Routes    []string `json:"routes"`
	Name      string   `json:"name"`
	Healthy   bool     `json:"-"`
}

type UnauthorizedError struct{}

func (u *UnauthorizedError) Error() string {
	return "unauthorized"
}

func (gws Gateways) MarshalIni() []byte {
	output := bytes.NewBufferString("")
	for _, gw := range gws {
		payload := gw.MarshalIni()
		output.Write(payload)
	}
	return output.Bytes()
}

func (gw *Gateway) MarshalIni() []byte {
	peerTemplate := `[Peer]
PublicKey = %s
AllowedIPs = %s
Endpoint = %s
`
	allowedIPs := make([]string, 0)
	allowedIPs = append(allowedIPs, gw.IP+"/32")
	if gw.Healthy == true {
		allowedIPs = append(allowedIPs, gw.Routes...)
	}

	s := fmt.Sprintf(peerTemplate, gw.PublicKey, strings.Join(allowedIPs, ","), gw.Endpoint)
	return []byte(s)
}

func GetGateways(sessionKey, apiServerURL string, ctx context.Context) (Gateways, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	deviceConfigAPI := fmt.Sprintf("%s/deviceconfig", apiServerURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, deviceConfigAPI, nil)
	if err != nil {
		return nil, fmt.Errorf("creating get request: %w", err)
	}
	req.Header.Add("x-naisdevice-session-key", sessionKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting device config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &UnauthorizedError{}
	}

	var gateways Gateways
	if err := json.NewDecoder(resp.Body).Decode(&gateways); err != nil {
		return nil, fmt.Errorf("unmarshalling response body into gateways: %w", err)
	}

	return gateways, nil
}
