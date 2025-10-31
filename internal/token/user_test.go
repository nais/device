package token

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEmail_WithoutContext(t *testing.T) {
	ctx := context.Background()
	email := GetEmail(ctx)
	assert.Equal(t, "", email)
}

func TestWithEmail_AndRetrieve(t *testing.T) {
	ctx := context.Background()
	email := "test@example.com"

	newCtx := WithEmail(ctx, email)
	retrievedEmail := GetEmail(newCtx)

	assert.Equal(t, email, retrievedEmail)
}
