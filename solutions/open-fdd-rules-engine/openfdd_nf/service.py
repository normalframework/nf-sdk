"""Poll NF timeseries, run Open-FDD RuleRunner, emit JSON summaries."""

from __future__ import annotations

import json
import logging
import os
import sys
import time

import pandas as pd

from open_fdd.engine import RuleRunner

from openfdd_nf.config import load_config
from openfdd_nf.nf_client import NFClient
from openfdd_nf.nf_timeseries import fetch_timeseries

log = logging.getLogger("openfdd_nf")


def summarize_faults(df: pd.DataFrame) -> dict:
    out: dict = {"rows": int(len(df)), "flags": {}}
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


def run_once(mapping_path: str | None = None) -> dict:
    cfg = load_config(mapping_path)
    client = NFClient(cfg.nf.base_url)
    df = fetch_timeseries(
        client,
        cfg.point_uuids,
        cfg.fetch.lookback_hours,
        cfg.fetch.resample_method,
        cfg.fetch.resample_window,
    )
    if df.empty:
        log.warning("No timeseries rows from NF; check UUIDs and time range")
        return {"error": "empty_timeseries", "detail": "no rows from /api/v1/point/data"}

    runner = RuleRunner(rules_path=cfg.run.rules_dir)
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
    return summary


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
