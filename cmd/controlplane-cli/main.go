package main

import (
	"log"
	"os"

	controlplanecli "github.com/nais/device/pkg/controlplane-cli"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    controlplanecli.FlagAdminPassword,
				Usage:   "naisdevice admin password",
				EnvVars: []string{"NAISDEVICE_ADMIN_PASSWORD"},
			},
		},
		Commands: []*cli.Command{
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
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
