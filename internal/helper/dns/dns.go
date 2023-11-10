package dns

func Apply(zones []string) error {
	// calls os specific apply
	return apply(zones)
}
