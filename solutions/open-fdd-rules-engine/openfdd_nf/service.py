"""Poll NF timeseries, run Open-FDD RuleRunner, emit JSON summaries."""

from __future__ import annotations

import json
import logging
import os
import sys
import time
import uuid
from dataclasses import asdict, dataclass
from typing import Any

import pandas as pd

from open_fdd.engine import RuleRunner

from openfdd_nf.config import AppConfig, load_config
from openfdd_nf.nf_client import NFClient
from openfdd_nf.nf_persist import persist_fault_flags, should_persist
from openfdd_nf.nf_timeseries import fetch_timeseries
from openfdd_nf.rule_meta import filter_rules_by_files, iter_rules_with_files, rule_summary

log = logging.getLogger("openfdd_nf")

COOKBOOK_URL = "https://bbartling.github.io/open-fdd/expression_rule_cookbook.html"


@dataclass
class RunOptions:
    lookback_hours: int | None = None
    only_rule_files: list[str] | None = None
    include_timeseries_stats: bool = False
    include_columns_present: bool = False
    sample_tail_rows: int = 0
    persist: bool | None = None
    dry_run: bool = False


def summarize_faults(df: pd.DataFrame) -> dict[str, Any]:
    out: dict[str, Any] = {"rows": int(len(df)), "flags": {}}
    if df.empty:
        return out
    flag_cols = [c for c in df.columns if c.endswith("_flag")]
    for c in flag_cols:
        s = df[c]
        if not isinstance(s, pd.Series):
            continue
        last = 0
        if len(s) and pd.notna(s.iloc[-1]):
            last = int(s.iloc[-1] == 1)
        out["flags"][c] = {
            "true_count": int(s.fillna(0).astype(bool).sum()),
            "last": last,
        }
    return out


def flags_expanded(df: pd.DataFrame) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    if df.empty:
        return rows
    for c in df.columns:
        if not c.endswith("_flag"):
            continue
        s = df[c]
        if not isinstance(s, pd.Series):
            continue
        n = len(s)
        true_ct = int(s.fillna(0).astype(bool).sum()) if n else 0
        last_v: int | None = None
        if n and pd.notna(s.iloc[-1]):
            last_v = int(s.iloc[-1] == 1)
        rows.append(
            {
                "flag": c,
                "true_count": true_ct,
                "last": last_v,
                "true_ratio": round(true_ct / n, 6) if n else 0.0,
            }
        )
    return sorted(rows, key=lambda x: x["flag"])


def timeseries_stats(df: pd.DataFrame, brick_cols: list[str]) -> dict[str, Any]:
    if df.empty:
        return {"row_count": 0, "columns": {}}
    out: dict[str, Any] = {"row_count": int(len(df)), "columns": {}}
    for col in brick_cols:
        if col not in df.columns:
            out["columns"][col] = {"present": False, "non_null_pct": 0.0}
            continue
        s = df[col]
        nn = int(s.notna().sum())
        n = len(s)
        out["columns"][col] = {
            "present": True,
            "non_null_pct": round(100.0 * nn / n, 2) if n else 0.0,
        }
    return out


def build_agent_brief(cfg: AppConfig, rule_files: list[str]) -> str:
    """Short markdown-style brief for LLM tool context."""
    lines = [
        "# Open-FDD × NF sidecar (agent brief)",
        "",
        f"- **Expression rule cookbook:** {COOKBOOK_URL}",
        "- Rules use **Brick class names** as DataFrame columns; this sidecar maps them from "
        "`mapping.yaml` → NF point UUIDs (HPL `/api/v1/point/data`).",
        f"- **NF base URL:** {cfg.nf.base_url}",
        f"- **Brick inputs configured:** {', '.join(sorted(cfg.point_uuids.keys()))}",
        f"- **Rule files:** {', '.join(rule_files)}",
        "",
        "## API hints",
        "- `POST /run` — full fault pass; optional JSON body for filters and `include_*` flags.",
        "- `GET /rules` — inventory (name, type, flag, brick_inputs).",
        "- `GET /rules/{{file}}` — raw YAML for authoring or review.",
        "- `POST /validate/rule` — validate a YAML snippet before deploy.",
        "- `GET /agent/context` — machine-readable contract + this brief.",
        "",
        "## Persistence",
        "- Optional `fault_outputs` in mapping maps `*_flag` columns to NF points; "
        "`POST /run` with `\"persist\": true` writes last flag values via `/api/v2/command/write`.",
        "",
    ]
    return "\n".join(lines)


def run_fault_pipeline(
    mapping_path: str | None = None,
    options: RunOptions | None = None,
) -> dict[str, Any]:
    """
    Full pipeline: load config, fetch NF data, run rules, optional persist.

    Backward compatible: `run_once(path)` wraps this with default options.
    """
    opts = options or RunOptions()
    cfg = load_config(mapping_path)
    run_id = str(uuid.uuid4())

    lookback = opts.lookback_hours
    if lookback is None:
        lookback = cfg.fetch.lookback_hours

    client = NFClient(cfg.nf.base_url)
    df = fetch_timeseries(
        client,
        cfg.point_uuids,
        lookback,
        cfg.fetch.resample_method,
        cfg.fetch.resample_window,
    )

    rules_dir = cfg.run.rules_dir
    if opts.only_rule_files:
        rules = filter_rules_by_files(rules_dir, opts.only_rule_files)
        if not rules:
            return {
                "run_id": run_id,
                "error": "no_matching_rules",
                "detail": f"no rules matched filter {opts.only_rule_files!r}",
            }
        runner = RuleRunner(rules=rules)
    else:
        runner = RuleRunner(rules_path=rules_dir)

    if df.empty:
        log.warning("No timeseries rows from NF; check UUIDs and time range")
        return {
            "run_id": run_id,
            "error": "empty_timeseries",
            "detail": "no rows from /api/v1/point/data",
        }

    result = runner.run(
        df,
        timestamp_col="timestamp",
        skip_missing_columns=True,
    )
    summary = summarize_faults(result)
    if "timestamp" in result.columns and len(result):
        summary["timestamp_min"] = str(result["timestamp"].min())
        summary["timestamp_max"] = str(result["timestamp"].max())
    else:
        summary["timestamp_min"] = None
        summary["timestamp_max"] = None

    out: dict[str, Any] = {
        "run_id": run_id,
        **summary,
    }

    out["flags_detail"] = flags_expanded(result)

    if opts.include_timeseries_stats:
        out["timeseries_stats"] = timeseries_stats(df, list(cfg.point_uuids.keys()))

    if opts.include_columns_present:
        out["result_columns"] = [str(c) for c in result.columns]

    max_tail = min(max(0, opts.sample_tail_rows), 200)
    if max_tail > 0 and not result.empty:
        tail = result.tail(max_tail)
        # keep timestamp + flags + a few brick cols to limit size
        keep = ["timestamp"] + [
            c for c in tail.columns if c.endswith("_flag")
        ]
        for b in list(cfg.point_uuids.keys())[:12]:
            if b in tail.columns and b not in keep:
                keep.append(b)
        keep = [c for c in keep if c in tail.columns]
        out["sample_tail"] = tail[keep].astype(object).where(tail[keep].notna(), None).to_dict(
            orient="records"
        )

    rule_inventory = [
        rule_summary(fn, r) for fn, r in iter_rules_with_files(rules_dir)
    ]
    out["rules_loaded"] = [x["file"] for x in rule_inventory if "error" not in x]
    out["agent_brief"] = build_agent_brief(
        cfg, out["rules_loaded"]
    )

    persist_env = os.environ.get("OPENFDD_PERSIST_DEFAULT", "").lower() in (
        "1",
        "true",
        "yes",
    )
    do_persist = should_persist(cfg, opts.persist, persist_env)
    if do_persist and not opts.dry_run:
        out["persistence"] = persist_fault_flags(
            client, result, cfg.fault_outputs
        )
    elif do_persist and opts.dry_run:
        would: list[dict[str, Any]] = []
        for k, t in cfg.fault_outputs.items():
            if k in result.columns:
                would.append({"flag": k, **asdict(t)})
        out["persistence"] = {"dry_run": True, "would_write": would}
    else:
        out["persistence"] = {"skipped": True}

    return out


def run_once(mapping_path: str | None = None) -> dict[str, Any]:
    """CLI / simple callers: omit verbose agent fields; keep run_id and persistence."""
    full = run_fault_pipeline(mapping_path, RunOptions())
    noisy = {"flags_detail", "agent_brief", "rules_loaded"}
    return {k: v for k, v in full.items() if k not in noisy}


def main() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(message)s",
        stream=sys.stderr,
    )
    path = os.environ.get("OPENFDD_MAPPING") or (
        sys.argv[1] if len(sys.argv) > 1 else "mapping.yaml"
    )
    cfg = load_config(path)
    while True:
        try:
            summary = run_once(path)
            print(json.dumps(summary, indent=2))
        except Exception:
            log.exception("run failed")
            raise
        if cfg.run.once:
            break
        time.sleep(max(10, cfg.run.poll_interval_seconds))


if __name__ == "__main__":
    main()
