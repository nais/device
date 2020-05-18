package apiserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateEnrollmentToken(t *testing.T) {
	expected := "eyJzZXJpYWwiOiJzZXJpYWwiLCJwdWJsaWNLZXkiOiJwdWJsaWNfa2V5IiwicGxhdGZvcm0iOiJwbGF0Zm9ybSJ9"
	enrollmentToken, err := GenerateEnrollmentToken("serial", "platform", []byte("public_key"))

	assert.NoError(t, err)
	assert.Equal(t, expected, enrollmentToken, "interface changed, remember to change the apiserver counterpart")
}
