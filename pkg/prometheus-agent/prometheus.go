package prometheusagent

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type SDConfig struct {
	Targets []string `json:"targets"`
}

func EncodePrometheusTargets(targetIPs []string, port int, writer io.Writer) error {
	var configTargets []string
	for _, ip := range targetIPs {
		configTargets = append(configTargets, fmt.Sprintf("%v:%v", ip, port))
	}

	return json.NewEncoder(writer).Encode([]SDConfig{{Targets: configTargets}})
}

func UpdateConfiguration(targetIPs []string) error {
	nodeTargetsFile, err := os.Create("/etc/prometheus/node-targets.json")
	if err != nil {
		return fmt.Errorf("unable to open file: %w", err)
	}
	defer nodeTargetsFile.Close()

	err = EncodePrometheusTargets(targetIPs, 9100, nodeTargetsFile)
	if err != nil {
		return fmt.Errorf("unable to write prometheus node config: %w", err)
	}

	gatewayTargetsFile, err := os.Create("/etc/prometheus/gateway-targets.json")
	if err != nil {
		return fmt.Errorf("unable to open file: %w", err)
	}
	defer gatewayTargetsFile.Close()

	err = EncodePrometheusTargets(targetIPs, 3000, gatewayTargetsFile)
	if err != nil {
		return fmt.Errorf("unable to write prometheus gateway config: %w", err)
	}

	return nil
}
