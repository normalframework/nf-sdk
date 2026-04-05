# Column map resolver workshop

Mirrors **[open-fdd `examples/column_map_resolver_workshop`](https://github.com/bbartling/open-fdd/tree/master/examples/column_map_resolver_workshop)** on PyPI: load a **`column_map`** from YAML, run **`RuleRunner`** on a pandas **`DataFrame`**.

Use this to learn **Haystack / DBO / 223P / Brick** naming next to the NF sidecar (which defaults to **Brick-named columns** from `mapping.yaml`).

## Run one ontology in one command

From **`solutions/open-fdd-rules-engine/`** (with `pip install -r requirements.txt`):

```bash
cd solutions/open-fdd-rules-engine

# List modes
python examples/column_map_resolver_workshop/run_ontology_demo.py --list

python examples/column_map_resolver_workshop/run_ontology_demo.py brick
python examples/column_map_resolver_workshop/run_ontology_demo.py minimal
python examples/column_map_resolver_workshop/run_ontology_demo.py haystack
python examples/column_map_resolver_workshop/run_ontology_demo.py dbo
python examples/column_map_resolver_workshop/run_ontology_demo.py 223p
```

Each mode loads a **`manifest_*.yaml`** + matching **`demo_rule*.yaml`**, builds the same sample **`DataFrame`** (`sat` hits 105 °F once), and prints rows plus whether **`demo_high_sat_flag`** fired.

## Files

| File | Purpose |
|------|---------|
| **`run_ontology_demo.py`** | CLI: **`brick`**, **`minimal`**, **`haystack`**, **`dbo`**, **`223p`** |
| `manifest_*.yaml` / `demo_rule*.yaml` | Paired maps + rules (see upstream repo for line-by-line parity) |
| `demo_one_shot.py` | Minimal manifest + `demo_rule` (no CLI) |
| `demo_multi_ontology_illustration.py` | Static dicts + loads every `manifest_*.yaml` |

## API (library)

- `load_column_map_manifest`, resolvers — `open_fdd.engine.column_map_resolver`

**Docs:** [Engine-only & IoT](https://bbartling.github.io/open-fdd/howto/engine_only_iot.html) · [Column map & resolvers](https://bbartling.github.io/open-fdd/column_map_resolvers.html)

**Security:** Do not load resolver implementations from config strings; compose in Python.

**Upstream:** To refresh this folder from **open-fdd**, copy `examples/column_map_resolver_workshop/` from [github.com/bbartling/open-fdd](https://github.com/bbartling/open-fdd).
