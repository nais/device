package main

import (
	"log"
	"os"

	controlplanecli "github.com/nais/device/internal/controlplane-cli"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    controlplanecli.FlagAPIServer,
				Usage:   "apiserver address",
				EnvVars: []string{"NAISDEVICE_APISERVER"},
				Value:   "127.0.0.1:8099",
			},
			&cli.StringFlag{
				Name:    controlplanecli.FlagAdminPassword,
				Usage:   "naisdevice admin password",
				EnvVars: []string{"NAISDEVICE_ADMIN_PASSWORD"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "passhash",
				Usage: "generate a password hash from a password",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     controlplanecli.FlagPassword,
						Usage:    "cleartext password",
						Required: true,
					},
				},
				Action: controlplanecli.HashPassword,
			},
			{
				Name:    "kolide",
				Aliases: []string{"k"},
				Usage:   "kolide cache",
				Subcommands: []*cli.Command{
					{
						Name:   "dump",
						Usage:  "dump kolide cache",
						Action: controlplanecli.GetKolideCache,
					},
				},
			},
			{
				Name:    "session",
				Aliases: []string{"s"},
				Usage:   "options for sessions",
				Subcommands: []*cli.Command{
					{
						Name:   "list",
						Usage:  "list sessions",
						Action: controlplanecli.ListSessions,
					},
				},
			},
			{
				Name:    "gateway",
				Aliases: []string{"gw"},
				Usage:   "options for gateways",
				Subcommands: []*cli.Command{
					{
						Name:   "list",
						Usage:  "list gateways",
						Action: controlplanecli.ListGateways,
					},
					{
						Name:  "enroll",
						Usage: "enroll a gateway",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     controlplanecli.FlagName,
								Usage:    "gateway name",
								Required: true,
							},
							&cli.StringFlag{
								Name:     controlplanecli.FlagEndpoint,
								Usage:    "public ip and port used for WireGuard connection",
								Required: true,
							},
						},
						Action: controlplanecli.EnrollGateway,
					},
					{
						Name:  "edit",
						Usage: "edit gateway parameters",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     controlplanecli.FlagName,
								Usage:    "gateway name",
								Required: true,
							},
							&cli.StringFlag{
								Name:     controlplanecli.FlagPasswordHash,
								Usage:    "password hash",
								Required: false,
							},
							&cli.StringFlag{
								Name:     controlplanecli.FlagPublicKey,
								Usage:    "public key",
								Required: false,
							},
						},
						Action: controlplanecli.EditGateway,
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
