// Package enroller implements a local auto-enrollment mechanism for devices and/or gateways.
package enroller

import (
	"context"
)

type AutoEnroller interface {
	Run(ctx context.Context) error
}
