// Package netmgr configures network interfaces through NetworkManager's D-Bus API.
//
// The D-Bus layer is ported from gobac/src/nwconfig/network.go, with one
// deliberate behavioural change. balenaOS copies /mnt/boot/system-connections
// over /etc/NetworkManager/system-connections on every boot, so editing an
// inherited connection profile silently reverts at the next reboot. Profiles we
// create ourselves land on the state partition and persist, so every write in
// this package goes to a profile named "nf-<iface>" that we own outright. See
// profiles.go.
package netmgr

import (
	"crypto/rand"
	"fmt"
	"os"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	nmBus       = "org.freedesktop.NetworkManager"
	nmPath      = "/org/freedesktop/NetworkManager"
	nmIface     = "org.freedesktop.NetworkManager"
	nmDevIface  = "org.freedesktop.NetworkManager.Device"
	nmWifiIface = "org.freedesktop.NetworkManager.Device.Wireless"
	nmIP4Iface  = "org.freedesktop.NetworkManager.IP4Config"
	nmAPIface   = "org.freedesktop.NetworkManager.AccessPoint"
	nmConnIface = "org.freedesktop.NetworkManager.Settings.Connection"
	nmSettings  = "org.freedesktop.NetworkManager.Settings"
	nmActive    = "org.freedesktop.NetworkManager.Connection.Active"
	dbusProps   = "org.freedesktop.DBus.Properties"

	settingsPath = "/org/freedesktop/NetworkManager/Settings"

	// ownedPriority is the autoconnect-priority we stamp on profiles we own.
	// Inherited balena profiles sit at 0, so ours wins autoconnect at boot.
	ownedPriority = int32(100)
)

// Settings is NetworkManager's a{sa{sv}} connection settings dictionary.
type Settings map[string]map[string]dbus.Variant

// Manager is a connection to NetworkManager on the system bus.
type Manager struct {
	conn *dbus.Conn
}

// New connects to the system bus. On balenaOS the host bus is exposed to the
// container by the io.balena.features.dbus label; DBUS_SYSTEM_BUS_ADDRESS must
// point at /host/run/dbus/system_bus_socket or we will talk to the container's
// own bus instead of the host's.
func New() (*Manager, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("connect to system bus: %w", err)
	}
	m := &Manager{conn: conn}
	if _, err := m.Version(); err != nil {
		return nil, fmt.Errorf("NetworkManager not reachable on %s: %w", busAddress(), err)
	}
	return m, nil
}

func busAddress() string {
	if a := os.Getenv("DBUS_SYSTEM_BUS_ADDRESS"); a != "" {
		return a
	}
	return "unix:path=/var/run/dbus/system_bus_socket (default)"
}

// Close releases the bus connection.
func (m *Manager) Close() error { return m.conn.Close() }

func (m *Manager) nm() dbus.BusObject {
	return m.conn.Object(nmBus, dbus.ObjectPath(nmPath))
}

func (m *Manager) settings() dbus.BusObject {
	return m.conn.Object(nmBus, dbus.ObjectPath(settingsPath))
}

func (m *Manager) object(path dbus.ObjectPath) dbus.BusObject {
	return m.conn.Object(nmBus, path)
}

// Version returns the running NetworkManager version.
func (m *Manager) Version() (string, error) {
	v, err := m.nm().GetProperty(nmIface + ".Version")
	if err != nil {
		return "", err
	}
	s, _ := v.Value().(string)
	return s, nil
}

// devicePaths returns every device NetworkManager knows about.
func (m *Manager) devicePaths() ([]dbus.ObjectPath, error) {
	var paths []dbus.ObjectPath
	if err := m.nm().Call(nmIface+".GetDevices", 0).Store(&paths); err != nil {
		return nil, fmt.Errorf("GetDevices: %w", err)
	}
	return paths, nil
}

// findDevice resolves an interface name to its device object path.
func (m *Manager) findDevice(name string) (dbus.ObjectPath, error) {
	var path dbus.ObjectPath
	err := m.nm().Call(nmIface+".GetDeviceByIpIface", 0, name).Store(&path)
	if err != nil {
		return "", fmt.Errorf("device %q not found: %w", name, err)
	}
	return path, nil
}

// prop reads a D-Bus property, returning the zero value if it is missing or of
// an unexpected type. NetworkManager's property set varies across versions, and
// a missing property should degrade the display rather than fail the request.
func prop[T any](obj dbus.BusObject, name string) (T, bool) {
	var zero T
	v, err := obj.GetProperty(name)
	if err != nil {
		return zero, false
	}
	t, ok := v.Value().(T)
	if !ok {
		return zero, false
	}
	return t, true
}

// sanitize strips properties whose legacy D-Bus signatures (notably the
// a(ayuay) address and route arrays) do not round-trip through Go's generic
// dbus.Variant handling. Their modern "-data" equivalents carry the same
// information, so dropping them on a read-modify-write is safe. Carried over
// from gobac/src/nwconfig/network.go.
func sanitize(s Settings) {
	for _, section := range []string{"ipv4", "ipv6"} {
		if sec, ok := s[section]; ok {
			delete(sec, "addresses")
			delete(sec, "routes")
		}
	}
}

// uuid4 generates a random UUID for new connection profiles.
func uuid4() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func str(s Settings, section, key string) string {
	if sec, ok := s[section]; ok {
		if v, ok := sec[key]; ok {
			out, _ := v.Value().(string)
			return out
		}
	}
	return ""
}

func i32(s Settings, section, key string) int32 {
	if sec, ok := s[section]; ok {
		if v, ok := sec[key]; ok {
			switch n := v.Value().(type) {
			case int32:
				return n
			case uint32:
				return int32(n)
			}
		}
	}
	return 0
}

func i64(s Settings, section, key string) int64 {
	if sec, ok := s[section]; ok {
		if v, ok := sec[key]; ok {
			switch n := v.Value().(type) {
			case int64:
				return n
			case uint64:
				return int64(n)
			case int32:
				return int64(n)
			case uint32:
				return int64(n)
			}
		}
	}
	return 0
}

// sectionKeys is used in log lines when a settings update is rejected.
func sectionKeys(s Settings) string {
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	return strings.Join(keys, ",")
}
