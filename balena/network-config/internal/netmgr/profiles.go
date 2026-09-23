package netmgr

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

// ProfileID is the connection id netcfgd uses for the profile it owns on an
// interface. Ownership is the whole persistence story: balenaOS re-copies
// /mnt/boot/system-connections over /etc/NetworkManager/system-connections on
// every boot, so a profile inherited from the boot partition reverts whatever
// we write to it. A profile we create over D-Bus lives on the state partition
// and survives both reboot and OS upgrade.
func ProfileID(iface string) string { return "nf-" + iface }

type ownedProfile struct {
	path     dbus.ObjectPath
	priority int32
	metric   int64
	method   string
}

// listConnections returns every saved connection profile.
func (m *Manager) listConnections() ([]dbus.ObjectPath, error) {
	var paths []dbus.ObjectPath
	err := m.settings().Call(nmSettings+".ListConnections", 0).Store(&paths)
	if err != nil {
		return nil, fmt.Errorf("ListConnections: %w", err)
	}
	return paths, nil
}

// getSettings reads a profile's settings dictionary.
func (m *Manager) getSettings(path dbus.ObjectPath) (Settings, error) {
	var s Settings
	err := m.object(path).Call(nmConnIface+".GetSettings", 0).Store(&s)
	if err != nil {
		return nil, fmt.Errorf("GetSettings %s: %w", path, err)
	}
	return s, nil
}

// ownedProfiles indexes the nf-<iface> profiles by interface name.
func (m *Manager) ownedProfiles() (map[string]ownedProfile, error) {
	paths, err := m.listConnections()
	if err != nil {
		return nil, err
	}
	out := map[string]ownedProfile{}
	for _, path := range paths {
		s, err := m.getSettings(path)
		if err != nil {
			continue
		}
		id := str(s, "connection", "id")
		iface := str(s, "connection", "interface-name")
		if iface == "" || id != ProfileID(iface) {
			continue
		}
		out[iface] = ownedProfile{
			path:     path,
			priority: i32(s, "connection", "autoconnect-priority"),
			metric:   i64(s, "ipv4", "route-metric"),
			method:   str(s, "ipv4", "method"),
		}
	}
	return out, nil
}

// findOwned returns the profile we own for an interface, if it exists.
func (m *Manager) findOwned(iface string) (dbus.ObjectPath, Settings, bool) {
	paths, err := m.listConnections()
	if err != nil {
		return "", nil, false
	}
	for _, path := range paths {
		s, err := m.getSettings(path)
		if err != nil {
			continue
		}
		if str(s, "connection", "id") == ProfileID(iface) &&
			str(s, "connection", "interface-name") == iface {
			return path, s, true
		}
	}
	return "", nil, false
}

// conflictingProfiles lists other saved profiles bound to this interface whose
// autoconnect-priority is at least as high as ours. Those may win autoconnect
// at the next boot and quietly undo our configuration, which is exactly the
// balena boot-partition trap, so the console surfaces them as a warning.
func (m *Manager) conflictingProfiles(iface string) []string {
	paths, err := m.listConnections()
	if err != nil {
		return nil
	}
	var out []string
	for _, path := range paths {
		s, err := m.getSettings(path)
		if err != nil {
			continue
		}
		id := str(s, "connection", "id")
		if id == ProfileID(iface) {
			continue
		}
		// A profile with no interface-name can bind to any device of a
		// matching type, so it is a candidate too.
		bound := str(s, "connection", "interface-name")
		if bound != "" && bound != iface {
			continue
		}
		if i32(s, "connection", "autoconnect-priority") >= ownedPriority {
			out = append(out, id)
		}
	}
	return out
}

// connType returns the NetworkManager setting name for a device type.
func connType(devType uint32) (string, error) {
	switch devType {
	case devTypeEthernet:
		return "802-3-ethernet", nil
	case devTypeWifi:
		return "802-11-wireless", nil
	default:
		return "", fmt.Errorf("unsupported device type %d", devType)
	}
}

// ensureOwned returns the profile netcfgd owns for an interface, creating it if
// necessary. Creation is deliberately lazy: reads never create profiles, so
// simply opening the console does not change how a device is configured. The
// new profile is seeded from whatever is currently active, which means creating
// it is a no-op in behaviour until the caller writes to it.
func (m *Manager) ensureOwned(iface string) (dbus.ObjectPath, Settings, error) {
	if path, s, ok := m.findOwned(iface); ok {
		return path, s, nil
	}

	devPath, err := m.findDevice(iface)
	if err != nil {
		return "", nil, err
	}
	devType, _ := prop[uint32](m.object(devPath), nmDevIface+".DeviceType")
	ctype, err := connType(devType)
	if err != nil {
		return "", nil, err
	}

	uuid, err := uuid4()
	if err != nil {
		return "", nil, fmt.Errorf("generate uuid: %w", err)
	}

	s := Settings{
		"connection": {
			"id":                   dbus.MakeVariant(ProfileID(iface)),
			"uuid":                 dbus.MakeVariant(uuid),
			"type":                 dbus.MakeVariant(ctype),
			"interface-name":       dbus.MakeVariant(iface),
			"autoconnect":          dbus.MakeVariant(true),
			"autoconnect-priority": dbus.MakeVariant(ownedPriority),
		},
		ctype:  {},
		"ipv4": {"method": dbus.MakeVariant("auto")},
		"ipv6": {"method": dbus.MakeVariant("auto")},
	}

	// Seed from the active profile so that adopting an interface does not
	// change its behaviour. Only the addressing sections are copied; identity
	// and autoconnect stay ours.
	if active, ok := m.activeSettings(devPath); ok {
		if v4, ok := active["ipv4"]; ok {
			seeded := map[string]dbus.Variant{}
			for _, k := range []string{"method", "address-data", "gateway", "dns", "dns-search", "route-metric"} {
				if val, ok := v4[k]; ok {
					seeded[k] = val
				}
			}
			if len(seeded) > 0 {
				s["ipv4"] = seeded
			}
		}
		if w, ok := active["802-11-wireless"]; ok && ctype == "802-11-wireless" {
			s[ctype] = w
			if sec, ok := active["802-11-wireless-security"]; ok {
				s["802-11-wireless-security"] = sec
			}
		}
	}
	sanitize(s)

	var path dbus.ObjectPath
	err = m.settings().Call(nmSettings+".AddConnection", 0, s).Store(&path)
	if err != nil {
		return "", nil, fmt.Errorf("AddConnection for %s (sections %s): %w", iface, sectionKeys(s), err)
	}
	return path, s, nil
}

// activeSettings returns the settings of the profile currently applied to a device.
func (m *Manager) activeSettings(devPath dbus.ObjectPath) (Settings, bool) {
	ac, ok := prop[dbus.ObjectPath](m.object(devPath), nmDevIface+".ActiveConnection")
	if !ok || ac == "/" || ac == "" {
		return nil, false
	}
	connPath, ok := prop[dbus.ObjectPath](m.object(ac), nmActive+".Connection")
	if !ok || connPath == "/" || connPath == "" {
		return nil, false
	}
	s, err := m.getSettings(connPath)
	if err != nil {
		return nil, false
	}
	return s, true
}

// update writes a settings dictionary back to a profile and reactivates it on
// the device so the change takes effect immediately.
func (m *Manager) update(iface string, path dbus.ObjectPath, s Settings) error {
	sanitize(s)
	if err := m.object(path).Call(nmConnIface+".Update", 0, s).Err; err != nil {
		return fmt.Errorf("update profile %s: %w", ProfileID(iface), err)
	}
	devPath, err := m.findDevice(iface)
	if err != nil {
		return err
	}
	err = m.nm().Call(nmIface+".ActivateConnection", 0, path, devPath, dbus.ObjectPath("/")).Err
	if err != nil {
		return fmt.Errorf("activate profile %s: %w", ProfileID(iface), err)
	}
	return nil
}

// Snapshot captures an interface's owned-profile settings so a change can be
// rolled back. Used by the web console's apply-then-confirm flow, which guards
// against reconfiguring the very interface the operator is browsing through.
type Snapshot struct {
	Iface    string
	Existed  bool
	Settings Settings
}

// Snapshot records the current owned-profile state for an interface.
func (m *Manager) Snapshot(iface string) Snapshot {
	path, s, ok := m.findOwned(iface)
	_ = path
	if !ok {
		return Snapshot{Iface: iface, Existed: false}
	}
	return Snapshot{Iface: iface, Existed: true, Settings: copySettings(s)}
}

// Restore reverts an interface to a previously captured snapshot. If netcfgd
// did not own the interface when the snapshot was taken, the profile it created
// is deleted, which hands the interface back to whatever profile was there
// before.
func (m *Manager) Restore(snap Snapshot) error {
	path, _, ok := m.findOwned(snap.Iface)
	if !ok {
		return nil
	}
	if !snap.Existed {
		if err := m.object(path).Call(nmConnIface+".Delete", 0).Err; err != nil {
			return fmt.Errorf("delete profile %s: %w", ProfileID(snap.Iface), err)
		}
		// Let NetworkManager fall back to whatever else matches the device.
		devPath, err := m.findDevice(snap.Iface)
		if err != nil {
			return err
		}
		m.nm().Call(nmIface+".ActivateConnection", 0, dbus.ObjectPath("/"), devPath, dbus.ObjectPath("/"))
		return nil
	}
	return m.update(snap.Iface, path, copySettings(snap.Settings))
}

func copySettings(s Settings) Settings {
	out := make(Settings, len(s))
	for section, kv := range s {
		c := make(map[string]dbus.Variant, len(kv))
		for k, v := range kv {
			c[k] = v
		}
		out[section] = c
	}
	return out
}
