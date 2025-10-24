package agenthttp

import (
	"testing"
)

func TestPath(t *testing.T) {
	addr = "localhost" // Set a fixed address for testing
	defer func() { addr = "" }()

	tests := []struct {
		name       string
		path       string
		withSecret bool
		want       string
	}{
		{
			name:       "no existing query, without secret",
			path:       "/path",
			withSecret: false,
			want:       "http://" + addr + "/path",
		},
		{
			name:       "no existing query, with secret",
			path:       "/path",
			withSecret: true,
			want:       "http://" + addr + "/path?s=" + mux.secret,
		},
		{
			name:       "existing query, without secret",
			path:       "/path?foo=bar",
			withSecret: false,
			want:       "http://" + addr + "/path?foo=bar",
		},
		{
			name:       "existing query, with secret",
			path:       "/path?foo=bar",
			withSecret: true,
			want:       "http://" + addr + "/path?foo=bar&s=" + mux.secret,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Path(tt.path, tt.withSecret); got != tt.want {
				t.Errorf("Path() = %v, want %v", got, tt.want)
			}
		})
	}
}
