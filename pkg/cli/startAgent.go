package cli

import (
	"context"
	"github.com/nais/device/pkg/systray"
	log "github.com/sirupsen/logrus"
	"net"
	"os/exec"

	"github.com/spf13/cobra"
)

// startAgentCmd represents the startAgent command
var startAgentCmd = &cobra.Command{
	Use:   "start-agent",
	Short: "start the naisdevice-agent",
	Long:  `TODO: write longer docs`,
	Run: func(cmd *cobra.Command, args []string) {
		// stolen from systray
		conn, err := net.Dial("unix", GrpcAddress)
		if err != nil {
			ctx, cancel := context.WithCancel(context.Background())
			err = exec.CommandContext(ctx, systray.AgentPath).Start()
			if err != nil {
				log.Fatalf("spawning naisdevice-agent: %v", err)
			}
			defer cancel()
		} else {
			err := conn.Close()
			if err != nil {
				log.Fatalf("closing connection: %v", err)
			}
		}
		log.Info("naisdevice-agent started")
	},
}

func init() {
	rootCmd.AddCommand(startAgentCmd)
}
