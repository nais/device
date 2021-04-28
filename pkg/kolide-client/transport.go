package kolide_client

import "net/http"

type Transport struct {
	Token string
}

func (t Transport) Client() *http.Client {
	return &http.Client{Transport: t}
}

func (t Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.Token)
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultTransport.RoundTrip(req)
}
