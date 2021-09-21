// +build linux darwin

package filesystem

import (
	"fmt"

	"github.com/nais/device/pkg/device-agent/config"
)

func ensurePlatformPrerequisites(c *config.Config) error {
	if err := filesExist(c.WireGuardGoBinary); err != nil {
		return fmt.Errorf("verifying if file exists: %w", err)
	}

	return nil
}
