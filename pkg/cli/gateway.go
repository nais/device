package cli

import (
	"context"
	"fmt"
	"github.com/nais/device/device-agent/open"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sort"

	"github.com/AlecAivazis/survey/v2"
)

// gatewayCmd represents the status command
var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "get current gateway status",
	RunE:  execStatus,
}

var gatewayConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "connect to a gateway",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("gateway connect called")
		ctx := context.Background()

		client, err := setupClient(GrpcAddress)
		if err != nil {
			return err
		}

		agentStatus, err := getAgentStatus(ctx, client)
		if err != nil {
			return err
		}

		gateways := agentStatus.GetGateways()
		sort.Slice(gateways, func(i, j int) bool {
			return gateways[i].GetName() < gateways[j].GetName()
		})

		var gwNames []string
		for _, gw := range gateways {
			if gw.GetRequiresPrivilegedAccess() {
				gwNames = append(gwNames, gw.GetName())
			}
		}

		if len(gwNames) == 0 {
			fmt.Print("already connected to all available gateways")
			return nil
		}

		gatewayName := ""
		prompt := &survey.Select{
			Message: "Choose a gateway:",
			Options: gwNames,
		}
		err = survey.AskOne(prompt, &gatewayName)
		if err != nil {
			return err
		}

		err = open.Open(fmt.Sprintf("https://naisdevice-jita.nais.io/?gateway=%s", gatewayName))
		if err != nil {
			return fmt.Errorf("error opening browser: %v", err)
		}
		fmt.Println("Go to browser to continue connection")
		return nil
	},
}

func init() {
	gatewayCmd.AddCommand(gatewayConnectCmd)
	rootCmd.AddCommand(gatewayCmd)
}
