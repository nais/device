package bootstrap_api

import (
	"fmt"
	"strings"
)

func Credentials(credentialEntries []string) (map[string]string, error) {
	credentials := make(map[string]string)
	for _, key := range credentialEntries {
		entry := strings.Split(key, ":")
		if len(entry) > 2 {
			return nil, fmt.Errorf("invalid format on credentials, should be comma-separated entries on format 'user:key'")
		}

		credentials[entry[0]] = entry[1]
	}

	return credentials, nil
}
