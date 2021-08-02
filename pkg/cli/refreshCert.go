package cli

import (
	"context"
	"fmt"
	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var refreshCertCmd = &cobra.Command{
	Use:   "cert-refresh",
	Short: "refresh certificates aka 'Give me Microsoft Access Certificate'",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("cert-refresh called")
		ctx := context.Background()

		client, err := setupClient(GrpcAddress)
		if err != nil {
			return err
		}

		getConfigResponse, err := client.GetAgentConfiguration(ctx, &pb.GetAgentConfigurationRequest{})
		if err != nil {
			return fmt.Errorf("get agent config: %v", err)
		}

		getConfigResponse.Config.CertRenewal = true
		setConfigRequest := &pb.SetAgentConfigurationRequest{Config: getConfigResponse.Config}

		_, err = client.SetAgentConfiguration(ctx, setConfigRequest)
		if err != nil {
			return fmt.Errorf("set agent config: %v", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(refreshCertCmd)
}
