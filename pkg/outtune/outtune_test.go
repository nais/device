//go:build outtune_integration
// +build outtune_integration

package outtune_test

import (
	"context"
	"crypto/x509"
	"math/big"
	"testing"

	"github.com/nais/device/pkg/outtune"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const ser = "insert your device serial here to test locally"

func TestGetCertificate(t *testing.T) {
	helper := &pb.MockDeviceHelperClient{}
	helper.On("GetSerial", mock.Anything, mock.Anything).Return(&pb.GetSerialResponse{Serial: ser}, nil)
	o := outtune.New(helper)
	err := o.Install(context.Background())
	assert.NoError(t, err)
}

func TestGenerateDBKey(t *testing.T) {
	cert := &x509.Certificate{
		RawIssuer:    []byte("i dont care"),
		SerialNumber: big.NewInt(1337),
	}
	dbkey, err := outtune.GenerateDBKey(cert)
	assert.NoError(t, err)
	assert.Equal(t, "AAAAAAAAAAAAAAACAAAACwU5aSBkb250IGNhcmU=", dbkey)
}
