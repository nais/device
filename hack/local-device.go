package main

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/davecgh/go-spew/spew"
	"github.com/kelseyhightower/envconfig"
	"github.com/nais/device/internal/apiserver/auth"
	"github.com/nais/device/internal/apiserver/config"
	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/apiserver/ip"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()

	cfg := config.DefaultConfig()

	err := envconfig.Process("APISERVER", &cfg)
	if err != nil {
		panic(fmt.Sprint("unable to process environment variables:", err))
	}

	if cfg.WireGuardIPv6 == "" {
		cfg.WireGuardIPv6 = "fd00::1"
	}

	err = cfg.Parse()
	if err != nil {
		panic(fmt.Sprint("unable to parse config:", err))
	}

	wireguardPrefix, err := netip.ParsePrefix(cfg.WireGuardNetworkAddress)
	if err != nil {
		panic(fmt.Sprint("parse wireguard network address:", err))
	}
	v4Allocator := ip.NewV4Allocator(wireguardPrefix, []string{cfg.WireGuardIPv4Prefix.Addr().String()})
	v6Allocator := ip.NewV6Allocator(cfg.WireGuardIPv6Prefix)
	db, err := database.New(cfg.DBPath, v4Allocator, v6Allocator, !cfg.KolideEventHandlerEnabled, logrus.New())
	if err != nil {
		panic(fmt.Sprint("initialize database:", err))
	}

	spew.Dump(cfg)

	if err := db.AddDevice(ctx, auth.MockDevice()); err != nil {
		panic(fmt.Sprint("add mock device:", err))
	}
}
