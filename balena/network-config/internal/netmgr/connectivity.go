package netmgr

// Connectivity mirrors NMConnectivityState.
type Connectivity string

const (
	ConnUnknown Connectivity = "unknown"
	ConnNone    Connectivity = "none"
	ConnPortal  Connectivity = "portal"
	ConnLimited Connectivity = "limited"
	ConnFull    Connectivity = "full"
)

var connectivityNames = map[uint32]Connectivity{
	0: ConnUnknown, 1: ConnNone, 2: ConnPortal, 3: ConnLimited, 4: ConnFull,
}

// Connectivity returns NetworkManager's cached view of internet reachability.
// "portal" is the interesting one in the field: it means a captive portal is
// intercepting, which looks identical to a broken proxy from the application's
// point of view.
func (m *Manager) Connectivity() Connectivity {
	v, ok := prop[uint32](m.nm(), nmIface+".Connectivity")
	if !ok {
		return ConnUnknown
	}
	return connectivityNames[v]
}

// CheckConnectivity forces a fresh connectivity check rather than reading the
// cached value.
func (m *Manager) CheckConnectivity() Connectivity {
	var state uint32
	if err := m.nm().Call(nmIface+".CheckConnectivity", 0).Store(&state); err != nil {
		return ConnUnknown
	}
	return connectivityNames[state]
}
