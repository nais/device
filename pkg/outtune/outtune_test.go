//go:build outtune_integration
// +build outtune_integration

package outtune_test

import (
	"context"
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
