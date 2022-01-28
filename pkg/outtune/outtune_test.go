package outtune_test

import (
	"context"
	"testing"

	"github.com/nais/device/pkg/outtune"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetCertificate(t *testing.T) {
	helper := &pb.MockDeviceHelperClient{}
	helper.On("GetSerial", mock.Anything, mock.Anything).Return(&pb.GetSerialResponse{Serial: "abc"}, nil)
	o := outtune.NewOuttune(helper)
	err := o.GetCertificate(context.Background())
	assert.NoError(t, err)
}

func TestPurge(t *testing.T) {
	helper := &pb.MockDeviceHelperClient{}
	helper.On("GetSerial", mock.Anything, mock.Anything).Return(&pb.GetSerialResponse{Serial: "abc"}, nil)
	o := outtune.NewOuttune(helper)
	err := o.Purge(context.Background())
	assert.NoError(t, err)
}
