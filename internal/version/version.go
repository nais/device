package version

var (
	Revision = "unknown" // Git commit hash
	Version  = "unknown" // Numeric version

	// for use when using structured logging
	LogFields = map[string]any{
		"revision": Revision,
		"version":  Version,
	}
)
