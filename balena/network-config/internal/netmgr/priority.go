package netmgr

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

// Rank orders interfaces for failover. Rank 1 is the most preferred link.
// NetworkManager expresses this twice and both matter: autoconnect-priority
// decides which profile wins at boot, and route-metric decides which link owns
// the default route once several are up at once. Setting only one of them
// produces the classic "it came up on wifi even though ethernet is plugged in"
// behaviour.
type Rank struct {
	Iface string `json:"iface"`
	Rank  int    `json:"rank"`
}

const (
	// baseMetric is the route metric assigned to rank 1. Subsequent ranks step
	// up from here; lower metrics win in the kernel routing table.
	baseMetric = 100
	metricStep = 100
)

// SetPriority applies a failover ordering across interfaces. Ranks are applied
// in one pass so the ordering is always internally consistent.
func (m *Manager) SetPriority(ranks []Rank) error {
	seen := map[int]string{}
	for _, r := range ranks {
		if r.Rank < 1 {
			return fmt.Errorf("rank for %s must be 1 or greater", r.Iface)
		}
		if other, dup := seen[r.Rank]; dup {
			return fmt.Errorf("%s and %s cannot both be rank %d", other, r.Iface, r.Rank)
		}
		seen[r.Rank] = r.Iface
	}

	for _, r := range ranks {
		path, s, err := m.ensureOwned(r.Iface)
		if err != nil {
			return err
		}
		// Higher autoconnect-priority wins, so invert the rank. Staying well
		// above the inherited profiles' 0 keeps our profiles preferred.
		s["connection"]["autoconnect-priority"] = dbus.MakeVariant(ownedPriority + int32(100-r.Rank))
		if _, ok := s["ipv4"]; !ok {
			s["ipv4"] = map[string]dbus.Variant{"method": dbus.MakeVariant("auto")}
		}
		s["ipv4"]["route-metric"] = dbus.MakeVariant(int64(baseMetric + (r.Rank-1)*metricStep))

		if err := m.update(r.Iface, path, s); err != nil {
			return err
		}
	}
	return nil
}
