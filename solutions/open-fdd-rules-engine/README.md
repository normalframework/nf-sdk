# Open-FDD rules engine (Normal Framework sidecar)

This folder lives **only in the [nf-sdk](https://github.com/normalframework/nf-sdk) tree** (for example after `git clone https://github.com/normalframework/nf-sdk.git`). It does **not** belong inside the [open-fdd](https://github.com/bbartling/open-fdd) source repository. The Docker image installs **[open-fdd from PyPI](https://pypi.org/project/open-fdd/)** only (`pip install -r requirements.txt`); there is no editable install or git dependency on open-fdd.

The service runs **[open-fdd](https://pypi.org/project/open-fdd/)** against **Normal Framework** timeseries from the **HPL REST API** (`GET /api/v1/point/data`), the same surface documented under **Points & Historical Data** on [API Examples](https://docs.normal.dev/examples/) (e.g. [`get-data.py`](https://github.com/normalframework/nf-sdk/blob/master/examples/api/hpl/v1/get-data.py), [`download-csv.py`](https://github.com/normalframework/nf-sdk/blob/master/examples/api/hpl/v1/download-csv.py)). See the [Points](https://docs.normal.dev/concepts/points/) concept page for how NF models point UUIDs and history.

For NF platform context, see the [Normal Framework documentation](https://docs.normal.dev/).

## How this fits Normal Framework

**It is a legitimate pattern for NF.** The [Normal Framework home page](https://docs.normal.dev/) lists **fault detection solutions** as an intended use case: analyzing operational data to find problems. This sidecar does exactly that using **published REST APIs** (same family as the [nf-sdk Python API examples](https://docs.normal.dev/examples/)), not private internals.

**Applications SDK vs this solution.** The [Applications SDK overview](https://docs.normal.dev/applications/overview/) describes **managed JavaScript hooks** that run inside NF (schedules, point queries, sandboxing, NPM, lifecycle in the console). That is the first-class “install an app from git” model. This folder is different on purpose:

| | NF Application (hooks) | This Open-FDD sidecar |
| --- | --- | --- |
| Runtime | JavaScript in NF’s managed environment | Python in **your** container |
| Open-FDD | Would need a JS port or subprocess bridge | **`pip install open-fdd`** (PyPI) as intended |
| Data access | Normal APIs from `sdk` in the hook | `GET /api/v1/point/data` (+ auth like `examples/api/`) |
| Ops | Install/update via NF app flow | `docker compose`, CI, or k8s next to NF |

So the contribution is **complementary**: operators who want the **stock Open-FDD rule engine** on PyPI can run it beside NF without rewriting rules in JavaScript. A future step could be a **thin NF app** (hook on a schedule) that shells out or HTTP-calls this service—out of scope for the minimal sidecar.

**Security and tokens.** [Applications docs](https://docs.normal.dev/applications/overview/) note API token scopes when tokens are enabled. This sidecar uses the same **client credentials / basic** style as the SDK examples; ensure the NF client you create has permission to **read** the points you map. Credentials are supplied via environment variables (see `docker-compose.yml`); treat the container like any other service account.

**System modeling.** NF’s [system modeling](https://docs.normal.dev/) story includes Brick/Haystack-style normalization. Open-FDD YAML rules are keyed by **Brick class names**; wiring NF point UUIDs to those names in `mapping.yaml` is the integration point until a tighter model-driven mapping exists.

## What it does

1. Loads `mapping.yaml` (Brick class name → NF point UUID, plus fetch options).
2. Pulls resampled history for those UUIDs (chunked day-by-day, like `download-csv.py`).
3. Runs Open-FDD `RuleRunner` on the resulting wide table (columns = Brick names).
4. Prints a JSON summary of fault flags to stdout (useful for logs or a supervisor).

Auth matches the SDK examples: **`NF_CLIENT_ID`** / **`NF_CLIENT_SECRET`** (OAuth), or **`NF_BASIC_USER`** as `user:password`. Base URL: **`NFURL`** or `nf.base_url` in the mapping file.

## Quick start (Docker)

```bash
cd solutions/open-fdd-rules-engine
cp mapping.example.yaml mapping.yaml
# Edit mapping.yaml: set real point UUIDs from the NF console / point query API.
export NF_CLIENT_ID=... NF_CLIENT_SECRET=...
docker compose up --build
```

Mount your own rules by extending the image or bind-mounting over `/app/rules`. By default the image ships two example rules under `rules/` (sensor bounds + flatline for OAT and SAT).

## Local run (no Docker)

```bash
cd solutions/open-fdd-rules-engine
python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
cp mapping.example.yaml mapping.yaml
# Set rules_dir in mapping.yaml to the absolute path of ./rules for local use.
export NFURL=http://localhost:8080 NF_CLIENT_ID=... NF_CLIENT_SECRET=...
export OPENFDD_MAPPING=$PWD/mapping.yaml
python -m openfdd_nf
```

Set `run.once: true` in the mapping file for a single pass (e.g. cron).

## Mapping NF points to Open-FDD

NF exposes opaque point UUIDs. Open-FDD rules refer to **Brick** class names (e.g. `Outside_Air_Temperature_Sensor`). You bridge them in `point_uuids` exactly as you would wire columns in an exploratory notebook: discover UUIDs via the NF UI or `POST /api/v1/point/query`, then list them next to the Brick tag your YAML rules use.

If your DataFrame column names differ from `brick:` in the rules, Open-FDD supports a `column_map` in the engine API; this sidecar keeps columns equal to Brick names so the default rules work without extra mapping.

## Adding rules

Drop more `.yaml` files into `rules/` (or your mounted rules directory). Start from the examples here or from the upstream **[AHU rules examples](https://github.com/bbartling/open-fdd/tree/main/examples/AHU/rules)**. For patterns and snippets, use the **[Expression rule cookbook](https://bbartling.github.io/open-fdd/expression_rule_cookbook)** and the Open-FDD documentation.

## Tests

```bash
pip install -r requirements-dev.txt
pytest
```

Tests do not require a live NF instance.

## Contributing upstream

To propose this as part of **nf-sdk**, open a PR against [normalframework/nf-sdk](https://github.com/normalframework/nf-sdk) that adds or updates `solutions/open-fdd-rules-engine/` with this tree, and reference this README for operators.
