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

So the contribution is **complementary**: operators who want the **stock Open-FDD rule engine** on PyPI can run it beside NF without rewriting rules in JavaScript. A natural next step is a **managed NF Application** whose hooks run on a [schedule](https://docs.normal.dev/applications/overview/) and **HTTP-call** this sidecar’s API (`POST /run`) or parse its logs—hooks stay in the **JavaScript sandbox** with token scopes; the **Python + YAML + RuleRunner** stack stays in your container (same split as today, but wired for automation).

### Applications SDK (read this)

The [Applications SDK overview](https://docs.normal.dev/applications/overview/) describes **hooks**: managed **Node.js** functions, **schedules**, point queries, **NPM** deps, and **sandboxing** (chroot, API-only access to Normal, optional **token scopes**). That is Normal’s first-class extensibility model.

**This sidecar is not a replacement for hooks.** It is for teams that want **Open-FDD’s YAML rules and `RuleRunner`** without porting them to JavaScript. Integration patterns:

| Pattern | When |
| --- | --- |
| **A. Loop / stdout** (default `python -m openfdd_nf`) | Cron-friendly JSON to logs |
| **B. FastAPI** (`openfdd_nf.api_app`) | Same engine; **OpenAPI `/docs`**, **GET /agent/context**, **POST /run** with JSON options; optional `OPENFDD_API_KEY` (Bearer); **YAML hot reload every run** |
| **C. NF hook → HTTP** | Schedule in NF calls `POST /run` on the sidecar; hook writes summaries back via Normal APIs you already use (see *Persistence* below) |
| **D. NF hook → subprocess** | Less ideal in sandbox; prefer HTTP to the sidecar |

### HTTP API (operators, cron, AI agents)

The **FastAPI** app is the integration surface for **humans** (Swagger UI), **cron / Kubernetes jobs** (`curl` + JSON), and **LLM agents** (structured OpenAPI + stable field names).

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/health` | Liveness (no auth even when `OPENFDD_API_KEY` is set) |
| GET | `/capabilities` | Sidecar + PyPI `open-fdd` version, cookbook URL, endpoint list |
| GET | `/agent/context` | **Agent bundle:** Brick↔UUID mapping, rule inventory, markdown **brief** aligned with the [expression rule cookbook](https://bbartling.github.io/open-fdd/expression_rule_cookbook.html) |
| GET | `/mapping` | Same mapping as JSON (`?redact_uuids=true` for logs) |
| GET | `/rules` | Each `*.yaml` with `name`, `type`, `flag`, `brick_inputs` |
| GET | `/rules/{file}.yaml` | **Raw YAML** for review or prompt context |
| POST | `/run` | Run pipeline; optional JSON body (see below) |
| POST | `/validate/rule` | Check a YAML snippet before you write it to disk |

**`POST /run` body (all optional)** — omit `{}` for legacy behavior.

- `lookback_hours` — override mapping default for this run
- `only_rule_files` — e.g. `["sensor_bounds.yaml","sensor_flatline"]` (stems allowed)
- `include_timeseries_stats` — per-brick non-null % on ingested data
- `include_columns_present` — column list on the rule output frame
- `sample_tail_rows` — last N rows (timestamp + flags + sampled Brick columns), max 200
- `persist` — `true`/`false`/`null`; if `null`, use env `OPENFDD_PERSIST_DEFAULT`
- `dry_run` — with `persist`, show `would_write` without calling NF

**Response highlights:** `run_id`, `flags` (summary), `flags_detail` (per-flag ratios), `agent_brief` (markdown), `persistence` (NF write result or skipped).

Run locally: `uvicorn openfdd_nf.api_app:app --host 0.0.0.0 --port 8090`. Docker: `docker compose --profile api up --build open-fdd-rules-api` → `http://localhost:8090/docs`.

### Agentic workflow (cookbook → NF → faults)

1. **Authoring** — Rules follow the [Expression rule cookbook](https://bbartling.github.io/open-fdd/expression_rule_cookbook.html): Brick class names in `inputs`, `expression` for logic, `flag` for the output column.
2. **Bridging** — `mapping.yaml` maps each Brick class to an NF point UUID; the sidecar builds the same wide DataFrame Open-FDD expects.
3. **Validate** — `POST /validate/rule` with draft YAML before saving to `rules/`.
4. **Execute** — `POST /run` from an agent or scheduler; use `include_timeseries_stats` to diagnose missing data.
5. **Persist** — Either parse JSON in an **NF hook** and write via Normal APIs, or configure **`fault_outputs`** in `mapping.yaml` so the sidecar calls **`POST /api/v2/command/write`** (same pattern as [`examples/api/command/v2/write.py`](https://github.com/normalframework/nf-sdk/blob/master/examples/api/command/v2/write.py)). You need **write**-capable credentials and Analog Values (or equivalent) for each flag.

### Parity with Open-FDD AFDD stack (and what is intentionally different)

**Same:** YAML rules on disk, `open_fdd.engine.RuleRunner`, fault `*_flag` columns, `skip_missing_columns`, Brick column names (via `mapping.yaml`). **Hot reload:** each **`POST /run`** reloads all `*.yaml` from `rules_dir` (no stale in-memory rule cache for that request).

**Different / NF-specific:** **Ingress** is NF **HPL** (`/api/v1/point/data`), not Open-FDD’s Timescale `timeseries_readings`. **Persistence** is not copied automatically: Open-FDD AFDD writes **`fault_results`** / **`fault_state`** in Postgres; this sidecar does **not** assume your NF host exposes the same schema.

**Persistence options (choose what fits Normal ops):**

1. **`fault_outputs` + Command API** — Map each `*_flag` column to `{uuid, layer}` in `mapping.yaml`; call `POST /run` with `"persist": true` (or set `OPENFDD_PERSIST_DEFAULT=true`). Last sample of each flag is written as **real** `0.0` / `1.0`.
2. **NF hook** — Scheduled [hook](https://docs.normal.dev/applications/hooks/) calls `POST /run`, then uses `sdk.http` or command APIs to store summaries ([token scopes](https://docs.normal.dev/applications/overview/)).
3. **Sidecar database** — Add SQLite/Postgres next to this container if you want Open-FDD–shaped history without NF writes.
4. **Open-FDD platform** — Only if you also run full Open-FDD (usually not the “NF only” goal).

Future extensions (job ids, async webhooks) can sit on top of the same `run_id` without changing the rule engine.

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

**API mode (same image):** `docker compose --profile api up --build open-fdd-rules-api` then open `http://localhost:8090/docs`. Trigger a run with `POST /run`.

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

**API (local):** `uvicorn openfdd_nf.api_app:app --reload --port 8090` — then `POST http://127.0.0.1:8090/run`.

## Mapping NF points to Open-FDD

NF exposes opaque point UUIDs. Open-FDD rules refer to **Brick** class names (e.g. `Outside_Air_Temperature_Sensor`). You bridge them in `point_uuids` exactly as you would wire columns in an exploratory notebook: discover UUIDs via the NF UI or `POST /api/v1/point/query`, then list them next to the Brick tag your YAML rules use.

If your DataFrame column names differ from `brick:` in the rules, Open-FDD supports a `column_map` in the engine API; this sidecar keeps columns equal to Brick names so the default rules work without extra mapping.

## Adding rules

Drop more `.yaml` files into `rules/` (or your mounted rules directory). Start from the examples here or from the upstream **[AHU rules examples](https://github.com/bbartling/open-fdd/tree/main/examples/AHU/rules)**. For patterns and snippets, use the **[Expression rule cookbook](https://bbartling.github.io/open-fdd/expression_rule_cookbook)** and the Open-FDD documentation.

## Offline CSV demo (RTU11 — no NF, no Docker)

Shipped under **`examples/AHU/`**: **`RTU11.csv`** (same family as upstream [open-fdd AHU examples](https://github.com/bbartling/open-fdd/tree/main/examples/AHU)), **`rtu11_column_map.yaml`** (vendor columns → Brick names), and **`rules_demo/`** (bounds, flatline, one cookbook blend expression).

```bash
cd solutions/open-fdd-rules-engine
python tools/offline_csv_fdd.py --agent-brief --pretty
python tools/offline_csv_fdd.py --max-rows 500 --sample-tail 3
```

**Humans** — tweak rules under `examples/AHU/rules_demo/` and re-run. **AI agents** — consume JSON + `agent_brief_markdown`; same `RuleRunner` as production, only the DataFrame source changes. **CI** — pin `--max-rows` and assert on `summary.flags`.

**Hybrid ML** — see [`examples/AHU/RTU11_hybrid_rules_and_ml.ipynb`](examples/AHU/RTU11_hybrid_rules_and_ml.ipynb) (rules + residual ML, analogous to [open-fdd RTU7_machine_learning.ipynb](https://github.com/bbartling/open-fdd/blob/main/examples/AHU/RTU7_machine_learning.ipynb)); `pip install -r examples/AHU/requirements-ml.txt`.

## Tests

```bash
pip install -r requirements-dev.txt
pytest
```

Tests do not require a live NF instance.

## Contributing upstream

To propose this as part of **nf-sdk**, open a PR against [normalframework/nf-sdk](https://github.com/normalframework/nf-sdk) that adds or updates `solutions/open-fdd-rules-engine/` with this tree, and reference this README for operators.
