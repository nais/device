package dns

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	b := bytes.Buffer{}
	_ = write(&b, []string{"exampe.com", "internal.local"})
	expected := `[Resolve]
DNS=8.8.8.8#dns.google 8.8.4.4#dns.google 2001:4860:4860::8888#dns.google 2001:4860:4860::8844#dns.google
Domains=~exampe.com ~internal.local
DNSOverTLS=opportunistic
`

	assert.Equal(t, expected, b.String())
}
