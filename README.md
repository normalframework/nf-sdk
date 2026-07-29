<img src="logo_nf.png" width="50%"/>

Welcome to the NF SDK. This repository contains an installer script, `docker-compose` files, and examples for using the REST API.

[Normal Framework](https://www.normal.dev) | [🔗 Portal](https://portal.normal-online.net) | [🔗 Developer Docs](https://docs2.normal.dev)

## Quick Install

Run this on any Linux machine (Ubuntu 22.04+, Fedora 42+, or any system with Docker/Podman already installed):

```sh
curl -fsSL https://raw.githubusercontent.com/normalframework/nf-sdk/master/install.sh | sh
```

No credentials or config needed. The installer pulls the images, starts Normal Framework,
and prints a **sign-in link** to license the device:

```
Open this link on any device and sign in to finish:
    https://portal.normal-online.net/activate?code=ABCD-EFGH
Waiting for you to activate ....
```

Open the link (on that machine or any other), sign in, choose **Free demo** (a 30-day trial
with online services) or an existing license, name the site, and click **Activate**. The
installer detects it automatically and finishes. The management console is then at the
address it printed (**http://localhost:8080** by default).

### What the installer does

1. Installs Docker CE if not already present (Ubuntu/Debian via apt, Fedora/RHEL via dnf)
2. Pulls the `nf-full` and `redis` containers (GA images pull anonymously — no login)
3. Writes `docker-compose.yml` + `.env` to `/opt/nf` (or `~/nf` for rootless runtimes)
4. Starts Normal Framework and waits for the console
5. Prints a browser sign-in link and licenses the device once you approve

Re-running the installer **upgrades in place** — it reuses the existing ports, re-pulls the
latest images, and skips activation if the box is already licensed.

### How licensing works (device-authorization grant)

The installer never handles your portal credentials. It asks the portal to start a
device-authorization grant, prints a short code + link, and long-polls until you approve in
the browser; the portal then hands back a license the installer applies locally. One
**free** license is available per user and is **portable** — activating a new box moves it
and deactivates the old one.

Prefer not to activate now? Answer `n` at the prompt (or set `NF_ACTIVATE=no`); the box runs
unlicensed and you can activate later by re-running the installer.

### Enterprise

Enterprise installs pull from a **private** registry, so they still authenticate with a
`docker login` command from the portal (or `NF_USERNAME` / `NF_PASSWORD`). Set
`NF_RELEASE=enterprise` (auto-detected when you supply credentials or an enterprise
registry). Everything else — the browser activation flow — is the same.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `NF_ACTIVATE` | *(prompts)* | `yes` / `no` — license now (browser sign-in) or leave unlicensed |
| `NF_RELEASE` | `ga` | `ga` (anonymous pull) or `enterprise` (requires login) |
| `NF_TAG` | `3.10` | Container image tag to install |
| `NF_PORT` | `8080` | Console port (auto-remapped if in use) |
| `NF_DATA_DIR` | `/var/nf` | NF data directory (`~/nf/data` for rootless) |
| `NF_REDIS_DIR` | `/var/nf-redis` | Redis data directory (`~/nf/redis` for rootless) |
| `INSTALL_DIR` | `/opt/nf` | Where `docker-compose.yml` is written (`~/nf` for rootless) |
| `NF_USERNAME` | *(enterprise)* | Registry username from the portal |
| `NF_PASSWORD` | *(enterprise)* | Registry token from the portal |

### After install

```sh
# View logs
cd /opt/nf && sudo docker compose logs -f

# Stop / start
cd /opt/nf && sudo docker compose down
cd /opt/nf && sudo docker compose up -d
```

---

## Do More

Normal offers several pre-built integrations with other systems under permissive licenses. These can be quickly installed using our Application SDK.

| Integration | Description | Read Data | Write Data | System Model | UX |
| --- | --- | --- | --- | --- | --- |
| [Application Template](https://github.com/normalframework/applications-template) | Starting point for new apps. Includes example hooks for testing point writeability and Postgres import | ✔️ | | | |
| [Desigo CC](https://github.com/normalframework/app-desigocc) | Retrieve data from a Desigo CC NORIS API | ✔️ | | | |
| [Archilogic](https://github.com/normalframework/app-archilogic) | Display data on a floor plan | | | | ✔️ |
| [Guideline 36](https://github.com/normalframework/gl36-demo/tree/master) | Implement certain [Guideline 36](https://www.ashrae.org/news/ashraejournal/guideline-36-2021-what-s-new-and-why-it-s-important) sequences | | ✔️ | | ✔️ |
| [Avuity](https://github.com/normalframework/avuity-integration) | Expose data from [Avuity](https://www.avuity.com) occupancy sensors as BACnet objects | ✔️ | | ✔️ | |
| [ALC](https://github.com/normalframework/alc-plugin) | Import data from WebCTRL | | | ✔️ | |
| [OPC](https://github.com/normalframework/opc-integration) | Connect to OPC-UA Servers | ✔️ | | | |

## Release Types

As of version 3.8, two release types are available:

- **GA** (general availability) releases are hosted in the `normal.azurecr.io` registry. These releases require a valid license to be entered before they can be used.
- **Enterprise** releases are hosted in `normalframework.azurecr.io`. Access requires a master service agreement. These releases do not require activation to be used.
