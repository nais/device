package cli

import (
	"fmt"
	"github.com/nais/device/pkg/config"
	"github.com/nais/device/pkg/logger"
	"github.com/nais/device/pkg/notify"
	"github.com/nais/device/pkg/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var (
	ConfigDir   string
	GrpcAddress string
	LogLevel    string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "naisdevice",
	Short:   "Controlling naisdevice like a pro",
	Version: fmt.Sprintf("%s\nrevision: %s", version.Version, version.Revision),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	var err error
	ConfigDir, err = config.UserConfigDir()
	if err != nil {
		notify.Errorf("start naisdevice-cli: unable to find configuration directory: %v", err)
		os.Exit(1)
	}
	rootCmd.PersistentFlags().StringVar(&ConfigDir, "config", ConfigDir, "config file (default is $HOME/.device-cli.yaml)")
	rootCmd.PersistentFlags().StringVar(&LogLevel, "log-level", "warning", "which log level to output")
	rootCmd.PersistentFlags().StringVar(&GrpcAddress, "grpc-address", filepath.Join(ConfigDir, "agent.sock"), "path to device-agent unix socket")

	logger.SetupLogger(LogLevel, ConfigDir, "cli.log")

	log.Infof("naisdevice %s starting up", version.Version)
	log.Infof("configuration: %v, %v, %v", GrpcAddress, LogLevel, ConfigDir)

}
