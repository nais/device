package kolide_test

import (
	"encoding/json"
	"testing"

	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/stretchr/testify/assert"
)

func TestJSONMarshal(t *testing.T) {
	c := &kolide.Cache[string, string]{}

	val, ok := c.Get("key")
	assert.False(t, ok)
	assert.Equal(t, "", val)

	c.Set("key", "value")
	actual, err := json.Marshal(c)
	assert.NoError(t, err)
	assert.Equal(t, `{"key":"value"}`, string(actual))
}
