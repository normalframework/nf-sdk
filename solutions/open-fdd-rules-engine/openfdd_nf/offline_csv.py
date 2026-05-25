"""Load HVAC CSV exports, map columns to Brick names, run Open-FDD RuleRunner (no NF)."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import pandas as pd
import yaml
from open_fdd.engine import RuleRunner

COOKBOOK_URL = "https://bbartling.github.io/open-fdd/expression_rule_cookbook.html"


def load_column_map(path: str | Path) -> dict[str, Any]:
    raw = yaml.safe_load(Path(path).read_text(encoding="utf-8"))
    if not isinstance(raw, dict):
        raise ValueError("column map must be a YAML mapping")
    if "timestamp_column" not in raw or "rename" not in raw:
        raise ValueError("column map needs timestamp_column and rename")
    return raw


def dataframe_from_mapped_csv(
    csv_path: str | Path,
    map_spec: dict[str, Any],
    max_rows: int | None = None,
) -> pd.DataFrame:
    """Read CSV, parse time, rename to Brick columns, apply optional scale factors."""
    tcol = str(map_spec["timestamp_column"])
    df = pd.read_csv(csv_path, encoding="utf-8")
    if max_rows is not None and max_rows > 0:
        df = df.head(int(max_rows)).copy()

    if tcol not in df.columns:
        raise ValueError(f"timestamp column {tcol!r} not in CSV columns: {list(df.columns)}")

    df = df.copy()
    # Strip common US tz suffixes; pandas often rejects "EST" in the string.
    ts_raw = df[tcol].astype(str).str.replace(
        r"\s+(EST|EDT|CST|CDT|MST|MDT|PST|PDT)\s*$",
        "",
        regex=True,
    )
    df["timestamp"] = pd.to_datetime(ts_raw, format="%d-%b-%y %I:%M:%S %p", utc=True)
    df = df.drop(columns=[tcol])

    rename = map_spec.get("rename") or {}
    if not isinstance(rename, dict):
        raise ValueError("rename must be a dict")
    for src, dst in rename.items():
        if src in df.columns:
            df[dst] = pd.to_numeric(df[src], errors="coerce")
    # Drop original vendor columns that were renamed (keep unmapped out)
    drop_src = [c for c in rename if c in df.columns]
    if drop_src:
        df = df.drop(columns=drop_src, errors="ignore")

    scale = map_spec.get("scale") or {}
    if isinstance(scale, dict):
        for col, factor in scale.items():
            if col in df.columns:
                df[col] = df[col] * float(factor)

    df = df.sort_values("timestamp").reset_index(drop=True)
    return df


def summarize_faults(df: pd.DataFrame) -> dict[str, Any]:
    out: dict[str, Any] = {"rows": int(len(df)), "flags": {}}
    if df.empty:
        return out
    for c in df.columns:
        if not c.endswith("_flag"):
            continue
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


def run_offline(
    csv_path: str | Path,
    map_path: str | Path,
    rules_dir: str | Path,
    max_rows: int | None = None,
) -> tuple[pd.DataFrame, dict[str, Any]]:
    spec = load_column_map(map_path)
    df = dataframe_from_mapped_csv(csv_path, spec, max_rows=max_rows)
    runner = RuleRunner(rules_path=rules_dir)
    result = runner.run(df, timestamp_col="timestamp", skip_missing_columns=True)
    summary = summarize_faults(result)
    if "timestamp" in result.columns and len(result):
        summary["timestamp_min"] = str(result["timestamp"].min())
        summary["timestamp_max"] = str(result["timestamp"].max())
    else:
        summary["timestamp_min"] = None
        summary["timestamp_max"] = None
    meta = {
        "csv": str(Path(csv_path).resolve()),
        "column_map": str(Path(map_path).resolve()),
        "rules_dir": str(Path(rules_dir).resolve()),
        "cookbook_url": COOKBOOK_URL,
    }
    return result, {"summary": summary, "meta": meta}


def build_agent_brief(summary: dict[str, Any], meta: dict[str, Any]) -> str:
    lines = [
        "# Offline Open-FDD run (CSV → Brick columns → RuleRunner)",
        "",
        f"- **Cookbook:** {COOKBOOK_URL}",
        f"- **Rows:** {summary.get('rows', 0)}",
        f"- **Time range:** {summary.get('timestamp_min')} → {summary.get('timestamp_max')}",
        "",
        "## Fault flags (last sample + count of True)",
        "",
    ]
    for name, info in sorted((summary.get("flags") or {}).items()):
        lines.append(
            f"- **{name}** — last={info.get('last')}, true_count={info.get('true_count')}"
        )
    lines.extend(
        [
            "",
            "## For AI agents",
            "- Same engine as the NF sidecar: YAML rules + `skip_missing_columns=True`.",
            "- Production: replace CSV load with NF `GET /api/v1/point/data` via `mapping.yaml`.",
            f"- Data files: `{meta.get('csv')}`",
            "",
        ]
    )
    return "\n".join(lines)


def main_json_output(
    csv_path: Path,
    map_path: Path,
    rules_dir: Path,
    max_rows: int | None,
    agent_brief: bool,
    sample_tail: int,
) -> dict[str, Any]:
    _result, payload = run_offline(csv_path, map_path, rules_dir, max_rows=max_rows)
    if agent_brief:
        payload["agent_brief_markdown"] = build_agent_brief(
            payload["summary"], payload["meta"]
        )
    if sample_tail > 0 and _result is not None and not _result.empty:
        tail = _result.tail(min(sample_tail, 200))
        flag_cols = [c for c in tail.columns if c.endswith("_flag")]
        keep = ["timestamp"] + flag_cols
        for c in tail.columns:
            if c not in keep and c != "timestamp" and len(keep) < 14:
                keep.append(c)
        keep = [c for c in keep if c in tail.columns]
        payload["sample_tail"] = (
            tail[keep]
            .astype(object)
            .where(tail[keep].notna(), None)
            .to_dict(orient="records")
        )
    return payload
