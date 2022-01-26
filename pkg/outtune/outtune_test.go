package outtune_test

import (
	"context"
	"testing"

	"github.com/nais/device/pkg/outtune"
	"github.com/stretchr/testify/assert"
)

func TestGetCertificate(t *testing.T) {
	err := outtune.GetCertificate(context.Background())
	assert.NoError(t, err)
}

func TestPurge(t *testing.T) {
	err := outtune.Purge(context.Background())
	assert.NoError(t, err)
}
