# Test proxies

Four proxies from unmodified upstream images, for exercising netcfgd against
something realistic. Only configuration is mounted in.

```sh
docker compose up -d
```

| Port | What | Why |
|---|---|---|
| 3128 | Squid, no auth, `CONNECT` to 443 only | Squid's stock policy. Reproduces the port-7844 Cloudflare Tunnel failure rather than a convenient one. |
| 3129 | Squid, basic auth `proxyuser`/`proxypass`, `CONNECT` to 443 and 7844 | Distinguishes "wrong password" from "blocked by policy". |
| 1080 | Dante SOCKS5, no auth, unrestricted | The permissive case: a failure here is the client's fault. |
| 8080 | mitmproxy, re-encrypts TLS with its own CA | The enterprise MITM case. |

## The interception case

Port 8080 is the one worth understanding, because it is what most corporate
networks actually do.

Probe through it with no CA trusted and the TLS stage fails, naming the
interceptor and showing exactly what was presented:

```
overall: FAIL at tls
tls: TLS is being intercepted by "mitmproxy": x509: certificate signed by unknown authority
intercepted by: mitmproxy
CA presented?  False
chain as presented:
  leaf   subject=balena-cloud.com   issuer=mitmproxy
```

Note `CA presented? False`. The proxy sends only a leaf, signed by a CA it does
not send — which is normal, because TLS makes sending the root optional and
clients are expected to hold it already. **The certificate you need generally
cannot be recovered from the handshake**, so netcfgd names the interceptor and
says to get the CA from the network administrator, rather than offering the
leaf as though it were trust material.

mitmproxy writes its CA to `./mitmproxy/mitmproxy-ca-cert.pem` on first run.
Trust it and the same probe passes:

```sh
cp mitmproxy/mitmproxy-ca-cert.pem <netcfgd data dir>/ca.pem
```

```
overall: PASS
tls    pass   TLS 1.3, issuer "mitmproxy"
http   pass   404 Not Found
interception: by mitmproxy, trusted=True
```

Note it still reports the interception once trusted — a privately-rooted chain
is worth surfacing even when it works, because it explains the issuer name and
tells you the CA is a dependency.

On a device, the same certificate has to go in two places: `balenaRootCA` in
`config.json` (base64 PEM, for the host trust store used by the Supervisor,
cloudflared and NF) and netcfgd's own bundle (containers do not share the host
store).
