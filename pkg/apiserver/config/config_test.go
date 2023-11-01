package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIPOrPrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"10.255.240.1", "10.255.240.1/21"},
		{"10.255.240.1/21", "10.255.240.1/21"},
		{"10.255.240.1/22", "10.255.240.1/22"},
		{"2000::1", "2000::1/64"},
		{"2000::1/64", "2000::1/64"},
		{"2000::1/65", "2000::1/65"},
	}

	for _, tt := range tests {
		out, err := parsePrefixOrIP(tt.input)
		assert.NoError(t, err)
		assert.Equal(t, tt.expected, out.String())
	}
}
