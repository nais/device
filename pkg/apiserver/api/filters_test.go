package api

import (
	"testing"

	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
)

func TestUnique(t *testing.T) {
	devices := []*pb.Device{
		{
			Id: 1,
		},
		{
			Id: 1,
		},
		{
			Id: 2,
		},
	}

	assert.Len(t, unique(devices), 2)
}
