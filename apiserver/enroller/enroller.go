package enroller

import "net/http"

type enroller struct {
	Client *http.Client
}
