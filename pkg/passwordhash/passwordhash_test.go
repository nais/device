package passwordhash_test

import (
	"testing"

	"github.com/nais/device/pkg/passwordhash"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	salt := []byte(`hello world`)
	password := []byte(`very secret`)
	wrong := []byte(`wrong`)
	null := []byte(``)

	expectedHash := "$1$aGVsbG8gd29ybGQ=$TIcZEnPT5+xd2JPGcHOi+ZhzC+0nhQeWx621Gp7KhuUNwq0bStpEL8LU8DEFQCGOxx7DNfXcnjsTlEsWN+Vfkw=="

	key := passwordhash.HashPassword(password, salt)
	hash := passwordhash.FormatHash(key, salt)

	assert.Equal(t, expectedHash, string(hash))

	assert.NoError(t, passwordhash.Validate(password, hash))
	assert.Error(t, passwordhash.Validate(hash, password))
	assert.Error(t, passwordhash.Validate(wrong, hash))
	assert.Error(t, passwordhash.Validate(null, hash))
}
