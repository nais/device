package passwordhash_test

import (
	"testing"

	"github.com/nais/device/pkg/passwordhash"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	var salt = []byte(`hello world`)
	var password = []byte(`very secret`)
	var wrong = []byte(`wrong`)
	var null = []byte(``)

	var expectedHash = "$1$aGVsbG8gd29ybGQ=$TIcZEnPT5+xd2JPGcHOi+ZhzC+0nhQeWx621Gp7KhuUNwq0bStpEL8LU8DEFQCGOxx7DNfXcnjsTlEsWN+Vfkw=="

	key := passwordhash.HashPassword(password, salt)
	hash := passwordhash.FormatHash(key, salt)

	assert.Equal(t, expectedHash, string(hash))

	assert.NoError(t, passwordhash.Validate(password, hash))
	assert.Error(t, passwordhash.Validate(hash, password))
	assert.Error(t, passwordhash.Validate(wrong, hash))
	assert.Error(t, passwordhash.Validate(null, hash))
}
