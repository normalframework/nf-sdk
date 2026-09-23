package netmgr

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/godbus/dbus/v5"
)

// deviceTypes maps NMDeviceType to display strings.
//
// These values are from NetworkManager's NMDeviceType enum. The table carried
// over from gobac had several wrong (it listed 15 as cellular and 16 as vlan,
// which are actually TEAM and TUN), so a Tailscale or VPN tunnel device showed
// up as a configurable VLAN interface.
var deviceTypes = map[uint32]string{
	1: "ethernet", 2: "wifi", 5: "bluetooth", 6: "olpc-mesh", 7: "wimax",
	8: "modem", 9: "infiniband", 10: "bond", 11: "vlan", 12: "adsl",
	13: "bridge", 14: "generic", 15: "team", 16: "tun", 17: "ip-tunnel",
	18: "macvlan", 19: "vxlan", 20: "veth", 21: "macsec", 22: "dummy",
	23: "ppp", 24: "ovs-interface", 25: "ovs-port", 26: "ovs-bridge",
	27: "wpan", 28: "6lowpan", 29: "wireguard", 30: "wifi-p2p",
	31: "vrf", 32: "loopback", 33: "hsr", 34: "ipvlan",
}

const (
	devTypeEthernet = 1
	devTypeWifi     = 2
	devTypeVLAN     = 11
)

// configurableTypes are the device types this console can meaningfully
// configure. Filtering on type rather than on name is what keeps Bluetooth
// adapters — whose NetworkManager "Interface" is a bare MAC address — and
// tunnel devices created by VPNs out of the interface list.
// Cellular modems are deliberately absent: configuring one needs a "gsm"
// section with an APN, which this version does not write, so listing a modem
// would only offer a form that always fails.
var configurableTypes = map[uint32]bool{
	devTypeEthernet: true,
	devTypeWifi:     true,
	devTypeVLAN:     true,
}

// stateName maps NMDeviceState to a human-readable string.
func stateName(state uint32) string {
	switch {
	case state >= 120:
		return "failed"
	case state >= 110:
		return "disconnecting"
	case state >= 100:
		return "connected"
	case state >= 40:
		return "connecting"
	case state >= 30:
		return "disconnected"
	case state >= 20:
		return "unavailable"
	case state >= 10:
		return "unmanaged"
	default:
		return "unknown"
	}
}

// Interface is the full view of one NIC that the web console renders.
type Interface struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	State string `json:"state"`

	MAC     string `json:"mac"`
	Carrier bool   `json:"carrier"`
	Speed   string `json:"speed,omitempty"`
	Duplex  string `json:"duplex,omitempty"`
	MTU     uint32 `json:"mtu,omitempty"`

	Method  string   `json:"method"` // "auto" (DHCP), "manual", "disabled"
	Address string   `json:"address,omitempty"`
	Netmask string   `json:"netmask,omitempty"`
	Prefix  uint32   `json:"prefix,omitempty"`
	Gateway string   `json:"gateway,omitempty"`
	DNS     []string `json:"dns,omitempty"`

	SSID      string `json:"ssid,omitempty"`
	SignalPct int32  `json:"signalPct,omitempty"`
	SignalDBm int32  `json:"signalDbm,omitempty"`

	// ActiveProfile is the connection profile currently applied to the device.
	ActiveProfile string `json:"activeProfile,omitempty"`
	// Owned reports whether netcfgd manages this interface through its own
	// nf-<iface> profile. Interfaces we do not own still display, but their
	// settings came from somewhere else and we have not touched them.
	Owned bool `json:"owned"`
	// Priority is the autoconnect-priority on our owned profile, if any.
	Priority int32 `json:"priority"`
	// Rank is the failover position derived from Priority: 1 is the most
	// preferred link. Zero means the interface has never been ranked.
	Rank int `json:"rank"`
	// RouteMetric is the ipv4.route-metric on our owned profile, if any.
	RouteMetric int64 `json:"routeMetric"`
	// Primary reports whether this device carries the default route.
	Primary bool `json:"primary"`
	// Conflicts lists other profiles bound to this interface whose
	// autoconnect-priority is at least ours, meaning they may win at boot.
	Conflicts []string `json:"conflicts,omitempty"`
}

// skipPrefixes hides balena's own virtual plumbing from the interface list.
// Device type is the primary filter; these names catch balena's own plumbing
// that shares a configurable type, such as the veth pairs behind containers.
var skipPrefixes = []string{
	"lo", "balena", "br-", "resin-", "veth", "supervisor", "docker",
}

func skipDevice(name string) bool {
	for _, p := range skipPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// Interfaces enumerates every configurable NIC with its current state.
func (m *Manager) Interfaces() ([]Interface, error) {
	paths, err := m.devicePaths()
	if err != nil {
		return nil, err
	}

	// The device holding the default route, so the UI can show which link
	// traffic actually leaves by. This is the question people are really
	// asking when they configure interface priority.
	primary := m.primaryDevice()

	owned, err := m.ownedProfiles()
	if err != nil {
		return nil, err
	}

	out := make([]Interface, 0, len(paths))
	for _, path := range paths {
		dev := m.object(path)

		name, ok := prop[string](dev, nmDevIface+".Interface")
		if !ok || skipDevice(name) {
			continue
		}
		devType, _ := prop[uint32](dev, nmDevIface+".DeviceType")
		if !configurableTypes[devType] {
			continue
		}

		iface := Interface{Name: name, Method: "auto"}
		iface.Type = deviceTypes[devType]
		if iface.Type == "" {
			iface.Type = "unknown"
		}

		if state, ok := prop[uint32](dev, nmDevIface+".State"); ok {
			iface.State = stateName(state)
		}
		iface.MTU, _ = prop[uint32](dev, nmDevIface+".Mtu")
		iface.Primary = path == primary

		if ip4, ok := prop[dbus.ObjectPath](dev, nmDevIface+".Ip4Config"); ok && ip4 != "/" && ip4 != "" {
			m.fillIPv4(ip4, &iface)
		}

		// A live DHCP4 lease is the authoritative answer to "is this DHCP?",
		// independent of what any profile claims.
		if dhcp, ok := prop[dbus.ObjectPath](dev, nmDevIface+".Dhcp4Config"); ok && dhcp != "/" && dhcp != "" {
			iface.Method = "auto"
		} else if iface.Address != "" {
			iface.Method = "manual"
		}

		if devType == devTypeWifi {
			if ap, ok := prop[dbus.ObjectPath](dev, nmWifiIface+".ActiveAccessPoint"); ok && ap != "/" && ap != "" {
				m.fillWifi(ap, &iface)
			}
		}

		if ac, ok := prop[dbus.ObjectPath](dev, nmDevIface+".ActiveConnection"); ok && ac != "/" && ac != "" {
			if id, ok := prop[string](m.object(ac), nmActive+".Id"); ok {
				iface.ActiveProfile = id
			}
		}

		if p, ok := owned[name]; ok {
			iface.Owned = true
			iface.Priority = p.priority
			iface.RouteMetric = p.metric
			// SetPriority encodes rank as ownedPriority + (100 - rank), so a
			// profile still sitting at the base priority has no rank yet.
			if p.priority > ownedPriority {
				iface.Rank = int(ownedPriority + 100 - p.priority)
			}
			// Only a profile we own can be reasoned about; the method it
			// records is what will be applied at the next boot.
			if p.method != "" {
				iface.Method = p.method
			}
		}
		iface.Conflicts = m.conflictingProfiles(name)

		fillLinkState(name, &iface)
		out = append(out, iface)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Interface returns a single interface by name.
func (m *Manager) Interface(name string) (Interface, error) {
	ifaces, err := m.Interfaces()
	if err != nil {
		return Interface{}, err
	}
	for _, i := range ifaces {
		if i.Name == name {
			return i, nil
		}
	}
	return Interface{}, fmt.Errorf("interface %q not found", name)
}

// primaryDevice returns the device path carrying the default route.
func (m *Manager) primaryDevice() dbus.ObjectPath {
	ac, ok := prop[dbus.ObjectPath](m.nm(), nmIface+".PrimaryConnection")
	if !ok || ac == "/" || ac == "" {
		return ""
	}
	devs, ok := prop[[]dbus.ObjectPath](m.object(ac), nmActive+".Devices")
	if !ok || len(devs) == 0 {
		return ""
	}
	return devs[0]
}

func (m *Manager) fillIPv4(path dbus.ObjectPath, iface *Interface) {
	ip4 := m.object(path)

	if addrs, ok := prop[[]map[string]dbus.Variant](ip4, nmIP4Iface+".AddressData"); ok && len(addrs) > 0 {
		if v, ok := addrs[0]["address"]; ok {
			iface.Address, _ = v.Value().(string)
		}
		if v, ok := addrs[0]["prefix"]; ok {
			if p, ok := v.Value().(uint32); ok {
				iface.Prefix = p
				iface.Netmask = PrefixToNetmask(p)
			}
		}
	}
	iface.Gateway, _ = prop[string](ip4, nmIP4Iface+".Gateway")

	if ns, ok := prop[[]map[string]dbus.Variant](ip4, nmIP4Iface+".NameserverData"); ok {
		for _, entry := range ns {
			if v, ok := entry["address"]; ok {
				if s, _ := v.Value().(string); s != "" {
					iface.DNS = append(iface.DNS, s)
				}
			}
		}
	}
}

func (m *Manager) fillWifi(path dbus.ObjectPath, iface *Interface) {
	ap := m.object(path)
	if ssid, ok := prop[[]byte](ap, nmAPIface+".Ssid"); ok {
		iface.SSID = string(ssid)
	}
	if strength, ok := prop[uint8](ap, nmAPIface+".Strength"); ok {
		iface.SignalPct = int32(strength)
		iface.SignalDBm = percentTodBm(int32(strength))
	}
}

// fillLinkState reads carrier, speed, duplex and MAC from sysfs. These are not
// exposed consistently over D-Bus across NetworkManager versions, and carrier
// in particular is the first thing to check when an interface will not come up.
func fillLinkState(name string, iface *Interface) {
	base := "/sys/class/net/" + name + "/"
	read := func(f string) string {
		data, err := os.ReadFile(base + f)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}
	iface.Carrier = read("carrier") == "1"
	if s := read("speed"); s != "" && s != "-1" {
		iface.Speed = s
	}
	if d := read("duplex"); d != "" && d != "unknown" {
		iface.Duplex = d
	}
	if a := read("address"); a != "" {
		iface.MAC = a
	}
}
