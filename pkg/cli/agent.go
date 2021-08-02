package cli

import (
	"errors"
	"fmt"
	"github.com/nais/device/pkg/systray"
	"net"
	"os/exec"
	"syscall"

	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cobra"
)

const naisdeviceAgentProcessName = "naisdevice-agent"

var UnableToFindNaisDevice = errors.New("unable to find naisdevice process")

// startCmd represents the startAgent command
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "configure the naisdevice-agent",
}

func init() {
	agentCmd.AddCommand(stopCmd)
	agentCmd.AddCommand(startCmd)
	rootCmd.AddCommand(agentCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start the naisdevice-agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		// stolen from systray
		conn, err := net.Dial("unix", GrpcAddress)
		if err != nil {
			err = exec.Command(systray.AgentPath).Start()
			if err != nil {
				return fmt.Errorf("spawning naisdevice-agent: %v", err)
			}
		} else {
			err := conn.Close()
			if err != nil {
				return fmt.Errorf("closing connection: %v", err)
			}
		}
		fmt.Println("naisdevice-agent started")
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop the nasidevice agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := FindNaisDeviceProcess()
		if err != nil {
			return err
		}
		err = p.SendSignal(syscall.SIGINT)
		if err != nil {
			return fmt.Errorf("unable to kill naisdevice-process: %v", err)
		}
		return nil
	},
}

func FindNaisDeviceProcess() (*process.Process, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("unable to list processes: %v", err)
	}
	for _, p := range processes {
		n, err := p.Name()
		if err != nil {
			return nil, fmt.Errorf("unable to find naisdevice-process: %v, process %+v", err, p)
		}
		if n == naisdeviceAgentProcessName {
			return p, nil
		}
	}
	return nil, UnableToFindNaisDevice
}
