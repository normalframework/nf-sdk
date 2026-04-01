# AHU example data (offline)

- **`RTU11.csv`** — Copy of [open-fdd `examples/AHU/RTU11.csv`](https://github.com/bbartling/open-fdd/blob/main/examples/AHU/RTU11.csv); hourly RTU trend export for **offline** demos (no Normal Framework).
- **`rtu11_column_map.yaml`** — Renames vendor columns to **Brick** names expected by Open-FDD rules ([expression cookbook](https://bbartling.github.io/open-fdd/expression_rule_cookbook.html)).
- **`rules_demo/`** — Small rule set: bounds, flatline, and one GL36-style blend expression.

Run the offline tool from the solution root:

```bash
cd solutions/open-fdd-rules-engine
python tools/offline_csv_fdd.py --agent-brief
python tools/offline_csv_fdd.py --json | jq .summary.flags
```

Use this path when teaching **AI agents** or **operators** the same `RuleRunner` path the NF sidecar uses, but with a file instead of `GET /api/v1/point/data`.

### Hybrid rules + ML notebook

- **[`RTU11_hybrid_rules_and_ml.ipynb`](RTU11_hybrid_rules_and_ml.ipynb)** — Same pattern as upstream [RTU7_machine_learning.ipynb](https://github.com/bbartling/open-fdd/blob/main/examples/AHU/RTU7_machine_learning.ipynb): Open-FDD YAML rules, then **scikit-learn** regression on fan-on rows, **residual quantile** fault flag, and comparison plots. Install extras: `pip install -r examples/AHU/requirements-ml.txt`.
