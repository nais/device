package main_test

import (
	"bytes"
	main "github.com/nais/device/cmd/prometheus-agent"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGeneratePrometheusTargets(t *testing.T) {
	gws := []main.Gateway{{IP: "1.1.1.1"}, {IP: "2.2.2.2"}}
	buffer := bytes.NewBuffer(nil)
	expected := `[{"targets":["1.1.1.1:1234","2.2.2.2:1234"]}]` + "\n"
	err := main.WritePrometheusTargets(gws, 1234, buffer)

	assert.NoError(t, err)
	assert.Equal(t, expected, buffer.String())
}
