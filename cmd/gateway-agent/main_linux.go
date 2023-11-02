package main

import "github.com/google/gopacket/routing"

func NewRouter() (routing.Router, error) {
	return routing.New()
}
