package prometheusagent_test

import (
	"bytes"
	"testing"

	prometheusagent "github.com/nais/device/internal/prometheus-agent"
	"github.com/stretchr/testify/assert"
)

func TestGeneratePrometheusTargets(t *testing.T) {
	gws := []string{"1.1.1.1", "2.2.2.2"}
	buffer := bytes.NewBuffer(nil)
	expected := `[{"targets":["1.1.1.1:1234","2.2.2.2:1234"]}]` + "\n"
	err := prometheusagent.EncodePrometheusTargets(gws, 1234, buffer)

	assert.NoError(t, err)
	assert.Equal(t, expected, buffer.String())
}
