package pb

func (d *DeviceIssue) Equal(other *DeviceIssue) bool {
	if d == other {
		return true
	}

	if d == nil || other == nil {
		return false
	}

	return d.Title == other.Title &&
		d.Message == other.Message &&
		d.Severity == other.Severity &&
		d.DetectedAt.AsTime().Equal(other.DetectedAt.AsTime()) &&
		d.LastUpdated.AsTime().Equal(other.LastUpdated.AsTime()) &&
		d.ResolveBefore.AsTime().Equal(other.ResolveBefore.AsTime())
}
