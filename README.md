<img src="logo_nf.png" width="50%"/>

Welcome to the NF SDK. This repository contains an installer script, `docker-compose` files, and examples for using the REST API.

[Normal Framework](https://www.normal.dev) | [🔗 Portal](https://portal.normal-online.net) | [🔗 Developer Docs](https://docs2.normal.dev)

## Quick Install

Run this on any Linux machine (Ubuntu 22.04+, Fedora 42+, or any system with Docker/Podman already installed), or on a Mac with Docker Desktop for evaluation:

```sh
curl -fsSL https://raw.githubusercontent.com/normalframework/nf-sdk/master/install.sh | sh
```

No credentials or config needed. The installer prints a **sign-in link** first — signing in
both authorizes the image download and sets up the site:

```
── Sign in to Normal ──

  Sign in to Normal to set up this site and authorize the download:

    https://portal.normal-online.net/activate?code=ABCD-EFGH

    (code: ABCD-EFGH)

Waiting for you to sign in ....
```

Open the link (on that machine or any other), sign in, choose **Free demo** (a 30-day trial
with online services) or an existing license, set the site name + map location, accept the
terms, and click **Activate**. The installer then pulls the images, starts Normal Framework,
and licenses the box automatically. The management console is at the address it printed
(**http://localhost:8080** by default).

### What the installer does

1. Installs Docker CE if not already present (Ubuntu/Debian via apt, Fedora/RHEL via dnf; on
   macOS it points you at Docker Desktop rather than installing it)
2. Prints a browser sign-in link; signing in returns short-lived registry pull credentials
   and records your site details
3. Pulls the `nf-full` and `redis` containers and writes `docker-compose.yml` + `.env` to
   `/opt/nf` (or `~/nf` for rootless runtimes and macOS)
4. Starts Normal Framework and waits for the console
5. Exchanges the sign-in for a license once the box reports its machine id — no second sign-in

Re-running the installer **upgrades in place** — it reuses the existing ports and cached
registry login, re-pulls the latest images, and skips licensing if the box is already
licensed.

### How it works (device-authorization grant)

The installer never handles your portal credentials. It asks the portal to start a
device-authorization grant, prints a short code + link, and long-polls until you approve in
the browser. Approval returns registry pull credentials so the installer can download the
images; the same approval is later exchanged for a license bound to the box's machine id.
One **free** license is available per user and is **portable** — activating a new box moves
it and deactivates the old one.

Every image pull is gated behind a Normal portal account — there is no anonymous pull. GA
and Enterprise use the same flow; the portal returns the right registry for your account.

### CI / air-gapped installs

To install without a browser, set `NF_USERNAME` + `NF_PASSWORD` (registry credentials) in
the environment. The installer pulls with those and skips the sign-in; the box stays
unlicensed until you license it from the console (or via `AUTO_PROVISION_KEY`).

### macOS / Docker Desktop

The installer supports Docker Desktop on a Mac, but **only for evaluation and development**. Docker Desktop runs containers inside a Linux VM, which means:

- **BACnet/IP broadcast does not work.** The VM is behind NAT, so Who-Is/I-Am discovery neither reaches the LAN nor arrives from it. Devices can be polled by unicast address, or reached through a BBMD using foreign-device registration. BACnet/Ethernet and MS/TP do not work at all.
- **NF does not reliably come back after a restart.** Containers only run while Docker Desktop is running, which requires a desktop login — after a reboot NF stays down until someone signs in, and sleep/resume can wedge the VM. Turn on *Settings → General → Start Docker Desktop when you sign in* and expect to `docker compose up -d` by hand sometimes.
- **Volumes are bind mounts through the VM's file sharing layer**, so I/O is slower and redis persistence does not behave the way it does on Linux.

Because host networking is unavailable, the Mac install uses [`compose/macos.yml`](compose/macos.yml): a bridge network with the console published on `${NF_PORT}` and BACnet/IP on `47808/udp`, data under `~/nf`, and the timezone passed in as `NF_TZ` (`/etc/localtime` cannot be mounted into the VM).

For production, run NF on Linux.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `NF_TAG` | `3.10` | Container image tag to install |
| `NF_PORT` | `8080` | Console port (auto-remapped if in use) |
| `NF_DATA_DIR` | `/var/nf` | NF data directory (`~/nf/data` for rootless and macOS) |
| `NF_REDIS_DIR` | `/var/nf-redis` | Redis data directory (`~/nf/redis` for rootless and macOS) |
| `INSTALL_DIR` | `/opt/nf` | Where `docker-compose.yml` is written (`~/nf` for rootless and macOS) |
| `NF_USERNAME` | *(none)* | Registry username — escape hatch, skips the browser sign-in |
| `NF_PASSWORD` | *(none)* | Registry password — escape hatch, skips the browser sign-in |
| `NF_REGISTRY` | *(from sign-in)* | Registry hostname (only with `NF_USERNAME`/`NF_PASSWORD`) |
| `NF_TZ` | *(host timezone)* | Container timezone, macOS only |
| `NF_ASSUME_YES` | *(unset)* | Set to `1` to skip the Docker Desktop confirmation prompt |
| `NF_COMPOSE_REF` | `master` | Git ref the compose templates are fetched from |

### After install

```sh
# View logs
cd /opt/nf && sudo docker compose logs -f

# Stop / start
cd /opt/nf && sudo docker compose down
cd /opt/nf && sudo docker compose up -d
```

On macOS (and rootless runtimes) the install lives in `~/nf` and needs no `sudo`:

```sh
cd ~/nf && docker compose logs -f
cd ~/nf && docker compose up -d
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
