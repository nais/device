package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/passwordhash"
)

type gatewayAuthenticator struct {
	db database.Database
}

func NewGatewayAuthenticator(db database.Database) UsernamePasswordAuthenticator {
	return &gatewayAuthenticator{
		db: db,
	}
}

func (a *gatewayAuthenticator) Authenticate(username, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	gw, err := a.db.ReadGateway(ctx, username)
	if err != nil {
		return err
	}

	err = passwordhash.Validate([]byte(password), []byte(gw.PasswordHash))
	if err != nil {
		return fmt.Errorf("invalid username or password")
	}
	return nil
}
