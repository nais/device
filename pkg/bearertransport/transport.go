package bearertransport

import "net/http"

type Transport struct {
	AccessToken string
}

func (t Transport) Client() *http.Client {
	return &http.Client{Transport: t}
}

func (t Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.AccessToken)
	return http.DefaultTransport.RoundTrip(req)
}
