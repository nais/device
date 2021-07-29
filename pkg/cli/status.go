package cli

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sort"
)

func execStatus(_ *cobra.Command, _ []string) error {
	log.Info("status called")
	ctx := context.Background()

	client, err := setupClient(GrpcAddress)
	if err != nil {
		return err
	}

	agentStatus, err := getAgentStatus(ctx, client)
	if err != nil {
		return err
	}

	fmt.Printf("Status: %s\n", agentStatus.GetConnectionState())
	fmt.Printf("Connected since: %s\n", agentStatus.GetConnectedSince().AsTime().Format("15:04:05"))
	fmt.Println("Currently connected to the following gateways:")
	gateways := agentStatus.GetGateways()
	sort.Slice(gateways, func(i, j int) bool {
		return gateways[i].GetName() < gateways[j].GetName()
	})
	for _, gw := range gateways {
		s := "❌"
		if gw.Healthy {
			s = "✅"
		}
		fmt.Printf(" %s : %s\n", s, gw.GetName())
	}
	return nil
}

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "get current connection status",
	RunE:  execStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
