package netmgr

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/godbus/dbus/v5"
)

// WifiNetwork is one visible access point.
type WifiNetwork struct {
	SSID      string `json:"ssid"`
	SignalPct int32  `json:"signalPct"`
	SignalDBm int32  `json:"signalDbm"`
	Secured   bool   `json:"secured"`
	Freq      uint32 `json:"freq,omitempty"`
}

// wifiDevice finds the first wifi device and its path.
func (m *Manager) wifiDevice() (dbus.ObjectPath, error) {
	paths, err := m.devicePaths()
	if err != nil {
		return "", err
	}
	for _, p := range paths {
		if t, _ := prop[uint32](m.object(p), nmDevIface+".DeviceType"); t == devTypeWifi {
			return p, nil
		}
	}
	return "", fmt.Errorf("no wifi device found")
}

// ScanWifi triggers a scan and returns the visible networks, strongest first.
func (m *Manager) ScanWifi(ctx context.Context) ([]WifiNetwork, error) {
	devPath, err := m.wifiDevice()
	if err != nil {
		return nil, err
	}
	dev := m.object(devPath)

	// RequestScan is rate limited by NetworkManager and fails if a scan ran
	// recently. That is not an error for us: the cached access point list is
	// still worth returning.
	dev.Call(nmWifiIface+".RequestScan", 0, map[string]dbus.Variant{})
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
	}

	var apPaths []dbus.ObjectPath
	if err := dev.Call(nmWifiIface+".GetAccessPoints", 0).Store(&apPaths); err != nil {
		return nil, fmt.Errorf("GetAccessPoints: %w", err)
	}

	best := map[string]WifiNetwork{}
	for _, apPath := range apPaths {
		ap := m.object(apPath)
		ssidBytes, ok := prop[[]byte](ap, nmAPIface+".Ssid")
		if !ok || len(ssidBytes) == 0 {
			continue
		}
		n := WifiNetwork{SSID: string(ssidBytes)}
		if s, ok := prop[uint8](ap, nmAPIface+".Strength"); ok {
			n.SignalPct = int32(s)
			n.SignalDBm = percentTodBm(int32(s))
		}
		n.Freq, _ = prop[uint32](ap, nmAPIface+".Frequency")
		wpa, _ := prop[uint32](ap, nmAPIface+".WpaFlags")
		rsn, _ := prop[uint32](ap, nmAPIface+".RsnFlags")
		n.Secured = wpa != 0 || rsn != 0

		// The same SSID appears once per band and per AP; keep the strongest.
		if prev, ok := best[n.SSID]; !ok || n.SignalPct > prev.SignalPct {
			best[n.SSID] = n
		}
	}

	out := make([]WifiNetwork, 0, len(best))
	for _, n := range best {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SignalPct > out[j].SignalPct })
	return out, nil
}

// SetWifi joins a network using the profile netcfgd owns on the wifi interface.
// It waits for the connection to reach the activated state so a wrong password
// is reported as a wrong password rather than as an apparent success.
func (m *Manager) SetWifi(ctx context.Context, iface, ssid, psk string) error {
	if ssid == "" {
		return fmt.Errorf("SSID is required")
	}

	path, s, err := m.ensureOwned(iface)
	if err != nil {
		return err
	}

	s["802-11-wireless"] = map[string]dbus.Variant{
		"ssid": dbus.MakeVariant([]byte(ssid)),
		"mode": dbus.MakeVariant("infrastructure"),
	}
	if psk != "" {
		s["802-11-wireless"]["security"] = dbus.MakeVariant("802-11-wireless-security")
		s["802-11-wireless-security"] = map[string]dbus.Variant{
			"key-mgmt": dbus.MakeVariant("wpa-psk"),
			"psk":      dbus.MakeVariant(psk),
		}
	} else {
		delete(s, "802-11-wireless-security")
	}
	sanitize(s)

	if err := m.object(path).Call(nmConnIface+".Update", 0, s).Err; err != nil {
		return fmt.Errorf("update wifi profile: %w", err)
	}

	devPath, err := m.findDevice(iface)
	if err != nil {
		return err
	}
	var activePath dbus.ObjectPath
	err = m.nm().Call(nmIface+".ActivateConnection", 0, path, devPath, dbus.ObjectPath("/")).
		Store(&activePath)
	if err != nil {
		return fmt.Errorf("activate wifi profile: %w", err)
	}

	return m.waitActivated(ctx, activePath, ssid)
}

// waitActivated polls an active connection until it activates or fails.
// NetworkManager authenticates asynchronously, and on failure it removes the
// ActiveConnection object entirely, so a property read error is the signal that
// authentication was rejected.
func (m *Manager) waitActivated(ctx context.Context, activePath dbus.ObjectPath, ssid string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	active := m.object(activePath)
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("connection to %s timed out", ssid)
		case <-time.After(time.Second):
		}

		state, ok := prop[uint32](active, nmActive+".State")
		if !ok {
			return fmt.Errorf("authentication failed for %s (wrong password?)", ssid)
		}
		switch state {
		case 2: // NM_ACTIVE_CONNECTION_STATE_ACTIVATED
			return nil
		case 4: // NM_ACTIVE_CONNECTION_STATE_DEACTIVATED
			return fmt.Errorf("failed to connect to %s (wrong password?)", ssid)
		}
	}
}
