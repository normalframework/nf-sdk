Docker Compose files for Balena
-------------------------------

This directory has baseline docker file for when NF is deployed using
Balena.  There are some small differences between Balena's platform
and normal docker-compose which make it desirable to have separate
configuration.

Each numbered directory is a release: push it with

```sh
balena push <fleet> --source 3.10
```

Network configuration console (3.10 and later)
----------------------------------------------

The 3.10 release includes a `network-config` service
(`normal.azurecr.io/normalframework/netcfgd`, amd64 and arm64, public).
It is a local web console for configuring the device's network
interfaces and outbound proxy, and for diagnosing why a device on a
customer network cannot reach the cloud.

* **URL:** `http://<device-ip>:8081` (host networking; nf keeps port 80)
* **Login:** `admin` / `normal` — a password change is forced on first sign-in
* **Changes are enabled by default.** To lock a device down, set the
  device variable `NETCFG_READONLY=1` on the `network-config` service in
  balenaCloud. All diagnostics keep working (proxy tester, endpoint
  checklist, TLS interception detection, ping, traceroute, support
  bundle) but nothing can be changed. This only restarts the
  `network-config` container.

Things to know before making changes:

* **Applying a proxy restarts every container on the device** (balenaOS
  restarts balenaEngine on a `host-config` change). Use the proxy tester
  before saving.
* Interface changes are written to NetworkManager profiles named
  `nf-<iface>`, which persist across reboots and OS upgrades. Profiles
  from `/mnt/boot/system-connections` are not edited, because balenaOS
  overwrites them on boot.
* A device can show "online" in balenaCloud with a broken proxy — existing
  VPN / tunnel sessions survive while every new connection fails. Trust
  the endpoint checklist, not the dashboard.
* Do not delete the `netcfg-data` volume; the proxy watchdog would
  re-apply its configuration and restart all containers.

The service uses the same `dbus` and `supervisor-api` labels as `nf`, so
it grants no new privileges at the fleet level. To leave it out, delete
the `network-config` service and the `netcfg-data` volume from the
compose file.

Full documentation, proxy watchdog modes, and development notes are in
[network-config/README.md](network-config/README.md).
