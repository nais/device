package device_agent

import (
	"time"
)

// device-helper on Windows needs a _lot_ of time to configure the interface for the first time.
const helperTimeout = 20 * time.Second
