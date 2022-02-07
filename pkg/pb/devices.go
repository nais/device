package pb

import "time"

const MaxTimeSinceKolideLastSeen = 1 * time.Hour

func (d *Device) KolideSeenRecently() bool {
	lastSeen := d.GetKolideLastSeen().AsTime()
	deadline := lastSeen.Add(MaxTimeSinceKolideLastSeen)

	return deadline.After(time.Now())
}
