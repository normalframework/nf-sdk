# Examples (Open-FDD × Normal Framework sidecar)

This **`solutions/open-fdd-rules-engine/examples/`** tree ships **next to** the NF sidecar so you can try the engine without wiring **NF** first.

## Ontology naming (Brick, Haystack, DBO, 223P)

Same mini-demos as upstream **open-fdd** — logical names in YAML ↔ short columns (`sat` / `oat`) via **`column_map`**:

```bash
cd solutions/open-fdd-rules-engine
pip install -r requirements.txt

python examples/column_map_resolver_workshop/run_ontology_demo.py --list
python examples/column_map_resolver_workshop/run_ontology_demo.py haystack
```

Details: **[`column_map_resolver_workshop/README.md`](column_map_resolver_workshop/README.md)**.

## RTU11 offline CSV (Brick columns in the frame)

Vendor CSV → Brick names → **`rules_demo/`** (no NF):

```bash
python tools/offline_csv_fdd.py --agent-brief --pretty
```

See **[`AHU/README.md`](AHU/README.md)**.

## Production sidecar (NF)

Brick class names in **`mapping.yaml`** → NF point UUIDs; the service builds a DataFrame whose **columns are those Brick names**, so **`column_map`** is usually **not** needed. If your rules use non-Brick logical names, build a **`column_map`** in code (same as the workshop) and pass it to **`RuleRunner.run(...)`** — see upstream **[Getting started](https://bbartling.github.io/open-fdd/getting_started.html)**.

## Upstream

- [open-fdd examples](https://github.com/bbartling/open-fdd/tree/master/examples)
- [Engine docs](https://bbartling.github.io/open-fdd/)
