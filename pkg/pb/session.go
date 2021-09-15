package pb

import "time"

func (x *Session) Expired() bool {
	if x == nil {
		return true
	}

	return x.Expiry.AsTime().Before(time.Now())
}
