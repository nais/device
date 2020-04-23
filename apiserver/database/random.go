package database

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
)

func Bytes(n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}

func RandomString(n int, letters string) string {
	letterRunes := []rune(letters)

	var buf bytes.Buffer
	buf.Grow(n)
	l := uint32(len(letterRunes))
	// on each loop, generate one random rune and append to output
	for i := 0; i < n; i++ {
		buf.WriteRune(letterRunes[binary.BigEndian.Uint32(Bytes(4))%l])
	}
	return buf.String()
}
