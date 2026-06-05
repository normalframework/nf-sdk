<img src="logo_nf.png" width="50%"/>

Welcome to the NF SDK. This repository contains an installer script, `docker-compose` files, and examples for using the REST API.

[Normal Framework](https://www.normal.dev) | [🔗 Portal](https://portal.normal-online.net) | [🔗 Developer Docs](https://docs2.normal.dev)

## Quick Install

Run this on any Linux machine (Ubuntu 22.04+, Fedora 42+, or any system with Docker/Podman already installed):

```sh
curl -fsSL https://raw.githubusercontent.com/normalframework/nf-sdk/master/install.sh | sh
```

The installer will ask you to paste your `docker login` command from the [Normal Portal](https://portal.normal-online.net) (**Settings → API Keys**).

For non-interactive installs (CI/CD, provisioning scripts), pass credentials as environment variables instead:

```sh
curl -fsSL https://raw.githubusercontent.com/normalframework/nf-sdk/master/install.sh | \
  NF_USERNAME=<username> NF_PASSWORD=<token> sh
```

When it finishes, the management console is at **http://localhost:8080**.

### What the installer does

1. Installs Docker CE if not already present (Ubuntu/Debian via apt, Fedora/RHEL via dnf)
2. Logs in to the NF container registry
3. Downloads the appropriate `docker-compose.yml` to `/opt/nf` (or `~/nf` for rootless runtimes)
4. Pulls the `nf-full` and `redis` containers
5. Starts Normal Framework and waits for the console to respond

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `NF_USERNAME` | *(prompted)* | Registry username from the portal |
| `NF_PASSWORD` | *(prompted)* | Registry token from the portal |
| `NF_RELEASE` | `ga` | `ga` or `enterprise` (auto-detected from registry hostname) |
| `NF_TAG` | `3.10` | Container image tag to install |
| `NF_PORT` | `8080` | Console port |
| `NF_DATA_DIR` | `/var/nf` | NF data directory (`~/nf/data` for rootless) |
| `NF_REDIS_DIR` | `/var/nf-redis` | Redis data directory (`~/nf/redis` for rootless) |
| `INSTALL_DIR` | `/opt/nf` | Where `docker-compose.yml` is written (`~/nf` for rootless) |

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
