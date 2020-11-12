package basicauth

import (
	"net/http"
)

type Transport struct {
	Username string
	Password string
}

func (bat *Transport) Client() *http.Client {
	return &http.Client{Transport: bat}
}

func (bat Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(bat.Username, bat.Password)
	return http.DefaultTransport.RoundTrip(req)
}