package cli

import (
	"context"
	"fmt"
	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "connect to a NAV-cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("connect called")
		ctx := context.Background()

		client, err := setupClient(GrpcAddress)
		if err != nil {
			return err
		}

		agentStatus, err := getAgentStatus(ctx, client)
		if err != nil {
			return err
		}

		switch agentStatus.GetConnectionState() {
		case pb.AgentState_Connected:
			fmt.Println("Already connected")

			return nil
		case pb.AgentState_Disconnected:
			fmt.Println("Connecting")
			_, err := client.Login(context.Background(), &pb.LoginRequest{})
			if err != nil {
				return err
			}
			fmt.Println("Successfully connected! ✅")

			return nil
		}

		return fmt.Errorf("bad connection state: %v", agentStatus.GetConnectionState())
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
}

func setupClient(address string) (pb.DeviceAgentClient, error) {
	log.Debugf("naisdevice-agent on unix socket %s", GrpcAddress)
	connection, err := grpc.Dial(
		"unix:"+address,
		grpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to naisdevice-agent grpc server: %v", err)
	}

	return pb.NewDeviceAgentClient(connection), nil
}

func getAgentStatus(ctx context.Context, client pb.DeviceAgentClient) (*pb.AgentStatus, error) {
	log.Info("Requesting status updates from naisdevice-agent")

	statusStream, err := client.Status(ctx, &pb.AgentStatusRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to naisdevice-agent: %v", err)
	}

	log.Debugf("naisdevice-agent status stream established")

	as, err := statusStream.Recv()
	if err != nil {
		log.Errorf("Receive status from device-agent stream: %v", err)

		return nil, fmt.Errorf("failed to connect to naisdevice-agent: %v", err)
	}

	return as, nil
}
