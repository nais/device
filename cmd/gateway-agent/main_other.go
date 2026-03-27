//go:build !linux

package main

import (
	"fmt"

	"github.com/google/gopacket/routing"
)

func NewRouter() (routing.Router, error) {
	return nil, fmt.Errorf("routing not supported on this platform")
}
