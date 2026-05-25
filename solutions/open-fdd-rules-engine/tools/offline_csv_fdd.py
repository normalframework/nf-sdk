#!/usr/bin/env python3
"""
Offline fault detection demo: CSV + column map → Open-FDD RuleRunner.

Use cases:
  - **Humans** — learn rules against real trend exports before NF wiring.
  - **AI agents** — deterministic JSON + optional `agent_brief_markdown` for tool output.
  - **CI** — regression on pinned CSV + rules (see tests).

Examples:
  python tools/offline_csv_fdd.py
  python tools/offline_csv_fdd.py --max-rows 500 --sample-tail 5
  python tools/offline_csv_fdd.py --agent-brief --pretty
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

_ROOT = Path(__file__).resolve().parents[1]
if str(_ROOT) not in sys.path:
    sys.path.insert(0, str(_ROOT))

from openfdd_nf.offline_csv import main_json_output  # noqa: E402


def main() -> None:
    p = argparse.ArgumentParser(
        description="Run Open-FDD rules on a mapped HVAC CSV (no Normal Framework)."
    )
    p.add_argument(
        "--csv",
        type=Path,
        default=_ROOT / "examples" / "AHU" / "RTU11.csv",
        help="Trend export CSV",
    )
    p.add_argument(
        "--column-map",
        type=Path,
        default=_ROOT / "examples" / "AHU" / "rtu11_column_map.yaml",
        help="YAML: timestamp_column, rename, optional scale",
    )
    p.add_argument(
        "--rules-dir",
        type=Path,
        default=_ROOT / "examples" / "AHU" / "rules_demo",
        help="Directory of Open-FDD rule YAML files",
    )
    p.add_argument(
        "--max-rows",
        type=int,
        default=None,
        metavar="N",
        help="Only load first N rows (quick tests)",
    )
    p.add_argument(
        "--sample-tail",
        type=int,
        default=0,
        help="Include last N rows (flags + timestamp + some points) in JSON",
    )
    p.add_argument(
        "--agent-brief",
        action="store_true",
        help="Add agent_brief_markdown field for LLM-oriented summaries",
    )
    p.add_argument(
        "--pretty",
        action="store_true",
        help="Indent JSON",
    )
    args = p.parse_args()

    for path, label in (
        (args.csv, "CSV"),
        (args.column_map, "column map"),
        (args.rules_dir, "rules dir"),
    ):
        if not path.exists():
            print(f"Error: {label} not found: {path}", file=sys.stderr)
            sys.exit(1)
    if not args.rules_dir.is_dir():
        print(f"Error: rules dir is not a directory: {args.rules_dir}", file=sys.stderr)
        sys.exit(1)

    out = main_json_output(
        args.csv,
        args.column_map,
        args.rules_dir,
        max_rows=args.max_rows,
        agent_brief=args.agent_brief,
        sample_tail=args.sample_tail,
    )
    indent = 2 if args.pretty else None
    print(json.dumps(out, indent=indent))


if __name__ == "__main__":
    main()
