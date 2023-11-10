package auth

import (
	"context"
	"time"

	"github.com/nais/device/internal/pb"
)

const (
	SessionDuration = time.Hour * 10
)

type Authenticator interface {
	Login(ctx context.Context, token, serial, platform string) (*pb.Session, error)
}
