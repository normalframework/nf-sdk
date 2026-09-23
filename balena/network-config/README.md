# netcfgd — network and proxy configuration for balenaOS

A local web console for configuring a device's network interfaces and its
outbound proxy, and for working out why a device on a customer's network cannot
reach the cloud.

Runs as one container. One static Go binary, no npm, no build step: the whole UI
is embedded in the binary.

```
http://<device-ip>:8081     admin / normal     (it forces a change on first sign-in)
```

## Why it is split across two APIs

Interfaces and the proxy are configured through completely different mechanisms,
and neither choice was free:

**Interfaces go through NetworkManager's D-Bus API.** The balena Supervisor API
has no per-interface endpoint — its only network-touching routes are
`GET`/`PATCH /v1/device/host-config`, which cover the proxy and hostname and
nothing else. The other route, writing NetworkManager keyfiles to the boot
partition, is closed because balena does not permit bind mounting `/mnt/boot`
into a container.

**The proxy goes through the Supervisor.** It owns
`/mnt/boot/system-proxy/redsocks.conf`, which a container cannot write directly.

## The two things most likely to surprise you

**1. Connection profiles only persist if we create them.** balenaOS copies
`/mnt/boot/system-connections/*` over `/etc/NetworkManager/system-connections` on
every boot. Editing an inherited profile looks like it works and then silently
reverts at the next reboot. Profiles created over D-Bus land on the state
partition and survive both reboot and OS upgrade.

So every write goes to a profile named `nf-<iface>` that netcfgd owns outright,
created at `autoconnect-priority` 100 — comfortably above the `-999` that
balena's default wired connections sit at. The console warns when another
profile bound to the same interface could win at boot.

**2. Applying a proxy change restarts every container on the device.** On
balenaOS 2.82.6 and newer, `PATCH host-config` restarts balenaEngine. That single
fact shapes the entire watchdog: it reads the host's current configuration and
compares before acting, rate-limits itself, records intent to disk *before*
acting so being killed mid-change cannot cause a loop, and trips a breaker if it
ever finds itself changing things repeatedly.

## "Online in balenaCloud" is not evidence the network works

Observed on a test device with a dead proxy applied:

```
balena cloud   IS ONLINE: true
cloudflared    registered=true, 4 connections
NetworkManager connectivity: full
new outbound   all 9 endpoints fail at tcp, including cloudlink
```

redsocks only redirects *new* connections, so VPN and tunnel sessions opened before
the proxy was applied carry on untouched. The device looks healthy on every dashboard
while being unable to open a single new connection — it cannot pull an update or
re-establish anything that drops, and it will keep looking fine until something
forces a reconnect.

Two consequences. Probe, don't trust status: the endpoint checklist exists because
every platform signal here says healthy and all of them are wrong. And this is the
argument for the auto-reboot option after applying a proxy — without it, a device
that was already connected ends up in this half-broken state indefinitely.

Note also that balenaOS host SSH validates keys against balena cloud, so a genuinely
offline device refuses SSH logins. When that happens this console, on the LAN, is the
way back in.

## Running on a production device

Set `NETCFG_READONLY=1`.

Every diagnostic still works — probes, the endpoint checklist, TLS interception
detection, ping, traceroute, the bundle — because none of them change anything.
What is disabled is every mutating endpoint, so the console cannot apply a proxy,
reconfigure an interface, or restart containers, whether by an operator's
mistake or by the watchdog.

That is the right setting wherever the network already works and a proxy is not
required: you keep the ability to find out what is happening and lose the
ability to break it.

## Networks that re-encrypt TLS

Common on corporate networks — Zscaler, Palo Alto, Fortinet and similar
terminate TLS and re-issue certificates from a private CA. A device only works
behind one if it trusts that CA.

The prober names the CA rather than reporting a bare certificate error, and
hands back the material to trust it: subject, issuer, SHA-256 fingerprint,
expiry, the PEM, and the same PEM base64-encoded.

Check the fingerprint against what the network administrator says it should be,
then trust the CA in **two** places:

1. **The device** — `balenaRootCA` in `config.json` takes the base64 form and
   installs it into the host root store. This is what makes the Supervisor,
   cloudflared and NF work.
2. **This console** — containers do not share the host trust store, so netcfgd
   needs the PEM too, at `<data>/ca.pem` (or `NETCFG_CA_BUNDLE`). Without it the
   endpoint checklist reports certificate failures everywhere, which is exactly
   where it is least useful.

## Proxy watchdog modes

| Mode | Behaviour |
|---|---|
| `off` | Ensure no proxy is configured. |
| `static` | Apply the configured proxy and keep it applied. |
| `fallback` | Apply at boot; clear it if the device still cannot reach the internet after the grace period. For sites where nobody can tell you whether a proxy is needed. |
| `dynamic` | Start with no proxy, and only apply one after it has been proven to work from userspace. Slowest to settle, least disruptive. |

Guards, all configurable: a minimum dwell between changes, a maximum number of
changes per hour, update locks respected by default, and an optional reboot
after a change (the Supervisor does not close connections that were already
open, so only a reboot forces everything through a newly applied proxy).

## Diagnostics

The proxy tester runs **real handshakes** — SOCKS5 with RFC 1929 auth, SOCKS4/4a,
HTTP `CONNECT`, HTTP relay — from userspace, without touching the host. It
reports which stage failed rather than a boolean, because "the proxy is
unreachable", "your password is wrong", "no password is configured" and "this
destination is blocked by policy" are four different conversations.

Test before you save. That way a bad setting never costs you a restart of every
container on the device.

The endpoint checklist tests everything the device depends on and is meant to be
handed to a customer's network administrator. When a proxy is configured the
checks go **through it explicitly** rather than relying on redsocks to intercept
them — a transparently-proxied TCP check connects to redsocks and succeeds even
when the proxy goes on to refuse the destination.

### Cloudflare Tunnel

Remote access needs special attention, and the console calls it out separately:

- cloudflared has **no proxy support**. [PR #1514](https://github.com/cloudflare/cloudflared/pull/1514)
  is still open. It depends entirely on redsocks intercepting it.
- redsocks only redirects **TCP**. cloudflared prefers **QUIC over UDP**, which
  is never redirected — so on a network with a proxy, QUIC leaves the device
  directly, bypassing the proxy altogether. That works until someone blocks UDP,
  at which point remote access disappears with no other change on the device.
  Pin `protocol: http2` in `tunnel.yaml` to avoid depending on it.
- The edge port is **7844, not 443**, and
  [Cloudflare will not make it configurable](https://community.cloudflare.com/t/how-to-change-from-port-7844/410193).
  Plenty of proxies only allow `CONNECT` to 443. On those networks the tunnel
  cannot work through the proxy at all — that is an "ask IT to open 7844"
  conversation, and the console says so in those words.

The QUIC check is a real probe, not a guess: it sends a QUIC long-header packet
carrying a deliberately unsupported version, which RFC 9000 requires the server
to answer with a Version Negotiation packet.

## Deploying

```yaml
network-config:
  image: normal.azurecr.io/normalframework/netcfgd:0.1.8
  network_mode: host
  cap_add: [NET_ADMIN, NET_RAW]
  environment:
    - DBUS_SYSTEM_BUS_ADDRESS=unix:path=/host/run/dbus/system_bus_socket
    - NETCFG_LISTEN=:8081
  labels:
    io.balena.features.dbus: '1'
    io.balena.features.supervisor-api: '1'
  volumes:
    - netcfg-data:/data
```

Both labels are already granted to the `nf` service, so this adds nothing new at
the fleet level. `DBUS_SYSTEM_BUS_ADDRESS` is not optional — without it the
D-Bus client talks to the container's own bus and finds no interfaces. Keep the
`/data` volume: losing it makes the watchdog re-apply the proxy on next start,
restarting every container.

`docker-compose.test.yml` is the 3.10 release plus this service.

### Environment

| Variable | Default | Meaning |
|---|---|---|
| `NETCFG_LISTEN` | `:8081` | Address to serve on. 8081 avoids nf's `PORT=80` on host networking. |
| `NETCFG_DATA` | `/data` | Configuration and watchdog state. |
| `NETCFG_READONLY` | unset | Serve diagnostics but block all changes. |
| `NETCFG_FAKE_SUPERVISOR` | unset | Log Supervisor changes instead of making them. |

## Development

```sh
go test ./...
go run . -listen 127.0.0.1:8081 -data /tmp/netcfgd -fake-supervisor
```

`-fake-supervisor` logs proxy changes rather than applying them, so the logic can
be exercised against a real NetworkManager on a development machine without a
balena device.

`testproxy/` brings up three proxies from unmodified upstream images for testing
against something realistic:

```sh
cd testproxy && docker compose up -d
```

- `3128` Squid, no auth, `CONNECT` to 443 only. This is Squid's stock policy, and
  it reproduces the port-7844 tunnel failure rather than a convenient one.
- `3129` Squid, basic auth as `proxyuser` / `proxypass`, `CONNECT` to 443 and 7844.
- `1080` Dante SOCKS5, no auth, unrestricted.

## Not in this version

802.1X (wired and wireless EAP), VLAN sub-interfaces, static routes, MTU, and
cellular modem configuration. 802.1X is the one worth doing next — it is common
on the corporate networks these gateways land on, and it maps cleanly onto
NetworkManager's `802-1x` setting, which the owned-profile design already
accommodates.
