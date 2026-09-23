package netmgr

import (
	"fmt"
	"net"
	"strings"

	"github.com/godbus/dbus/v5"
)

// StaticConfig is a manual IPv4 configuration for one interface.
type StaticConfig struct {
	Address string   `json:"address"`
	Netmask string   `json:"netmask"` // dotted-quad or bare prefix length
	Gateway string   `json:"gateway"`
	DNS     []string `json:"dns"`
	Search  []string `json:"search,omitempty"`
}

// Validate checks a static configuration before anything is written to
// NetworkManager. Getting this wrong on the interface you are connected
// through takes the device off the network, so the checks are strict: the
// gateway must be inside the configured subnet, which is the mistake people
// actually make.
func (c StaticConfig) Validate() error {
	ip := net.ParseIP(strings.TrimSpace(c.Address))
	if ip == nil || ip.To4() == nil {
		return fmt.Errorf("address %q is not a valid IPv4 address", c.Address)
	}
	prefix, err := NetmaskToPrefix(c.Netmask)
	if err != nil {
		return err
	}
	if prefix == 0 || prefix > 30 {
		return fmt.Errorf("prefix /%d is not usable for a host address", prefix)
	}
	network := net.IPNet{IP: ip.Mask(net.CIDRMask(int(prefix), 32)), Mask: net.CIDRMask(int(prefix), 32)}

	if gw := strings.TrimSpace(c.Gateway); gw != "" {
		gwIP := net.ParseIP(gw)
		if gwIP == nil || gwIP.To4() == nil {
			return fmt.Errorf("gateway %q is not a valid IPv4 address", c.Gateway)
		}
		if !network.Contains(gwIP) {
			return fmt.Errorf("gateway %s is outside the subnet %s", gw, network.String())
		}
		if gwIP.Equal(ip) {
			return fmt.Errorf("gateway %s is the same as the interface address", gw)
		}
	}
	for _, d := range c.DNS {
		if net.ParseIP(d) == nil {
			return fmt.Errorf("DNS server %q is not a valid IP address", d)
		}
	}
	return nil
}

// SetStatic applies a manual IPv4 configuration to the profile netcfgd owns on
// the interface, creating that profile if needed.
func (m *Manager) SetStatic(iface string, cfg StaticConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	prefix, err := NetmaskToPrefix(cfg.Netmask)
	if err != nil {
		return err
	}

	path, s, err := m.ensureOwned(iface)
	if err != nil {
		return err
	}

	ipv4 := map[string]dbus.Variant{
		"method": dbus.MakeVariant("manual"),
		"address-data": dbus.MakeVariant([]map[string]dbus.Variant{{
			"address": dbus.MakeVariant(strings.TrimSpace(cfg.Address)),
			"prefix":  dbus.MakeVariant(prefix),
		}}),
	}
	if gw := strings.TrimSpace(cfg.Gateway); gw != "" {
		ipv4["gateway"] = dbus.MakeVariant(gw)
	}
	if len(cfg.DNS) > 0 {
		ipv4["dns"] = dbus.MakeVariant(ipToNMUint32(cfg.DNS...))
	}
	if len(cfg.Search) > 0 {
		ipv4["dns-search"] = dbus.MakeVariant(cfg.Search)
	}
	// Preserve the route metric so a static address does not silently reorder
	// interface preference.
	if metric := i64(s, "ipv4", "route-metric"); metric != 0 {
		ipv4["route-metric"] = dbus.MakeVariant(metric)
	}
	s["ipv4"] = ipv4

	return m.update(iface, path, s)
}

// SetDHCP returns an interface to automatic addressing.
func (m *Manager) SetDHCP(iface string) error {
	path, s, err := m.ensureOwned(iface)
	if err != nil {
		return err
	}
	ipv4 := map[string]dbus.Variant{"method": dbus.MakeVariant("auto")}
	if metric := i64(s, "ipv4", "route-metric"); metric != 0 {
		ipv4["route-metric"] = dbus.MakeVariant(metric)
	}
	s["ipv4"] = ipv4
	return m.update(iface, path, s)
}

// SetEnabled brings an interface up or down. Ported from setInterfaceState in
// gobac/src/nwconfig/network.go, including the wifi special case: for a radio,
// disconnecting is not enough, the radio itself has to be switched off, and
// re-enabling has to go through autoconnect because ActivateConnection("/")
// fails when there are no saved connections.
func (m *Manager) SetEnabled(iface string, enabled bool) error {
	devPath, err := m.findDevice(iface)
	if err != nil {
		return err
	}
	dev := m.object(devPath)
	devType, _ := prop[uint32](dev, nmDevIface+".DeviceType")

	if enabled {
		if devType == devTypeWifi {
			m.nm().Call(dbusProps+".Set", 0, nmIface, "WirelessEnabled", dbus.MakeVariant(true))
			dev.Call(dbusProps+".Set", 0, nmDevIface, "Autoconnect", dbus.MakeVariant(true))
			return nil
		}
		// Prefer our own profile if we have one, otherwise let NM choose.
		conn := dbus.ObjectPath("/")
		if p, _, ok := m.findOwned(iface); ok {
			conn = p
		}
		err = m.nm().Call(nmIface+".ActivateConnection", 0, conn, devPath, dbus.ObjectPath("/")).Err
		if err != nil {
			return fmt.Errorf("enable %s: %w", iface, err)
		}
		return nil
	}

	if devType == devTypeWifi {
		dev.Call(nmDevIface+".Disconnect", 0)
		dev.Call(dbusProps+".Set", 0, nmDevIface, "Autoconnect", dbus.MakeVariant(false))
		m.nm().Call(dbusProps+".Set", 0, nmIface, "WirelessEnabled", dbus.MakeVariant(false))
		return nil
	}
	if err := dev.Call(nmDevIface+".Disconnect", 0).Err; err != nil {
		return fmt.Errorf("disable %s: %w", iface, err)
	}
	return nil
}
