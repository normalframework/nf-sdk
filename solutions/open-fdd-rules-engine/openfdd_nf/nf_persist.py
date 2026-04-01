"""Write fault flag scalars back to NF via Command API (POST /api/v2/command/write)."""

from __future__ import annotations

import logging
from typing import Any

import pandas as pd

from openfdd_nf.config import AppConfig, FaultOutputTarget
from openfdd_nf.nf_client import NFClient

log = logging.getLogger("openfdd_nf.persist")


def _last_flag_value(series: pd.Series) -> float | None:
    if series is None or len(series) == 0:
        return None
    last = series.iloc[-1]
    if pd.isna(last):
        return None
    return 1.0 if bool(last) else 0.0


def persist_fault_flags(
    client: NFClient,
    result_df: pd.DataFrame,
    fault_outputs: dict[str, FaultOutputTarget],
) -> dict[str, Any]:
    """
    For each configured flag column, write last value (0/1 as real) to the mapped NF point.

    Requires NF credentials with permission to POST /api/v2/command/write.
    """
    if not fault_outputs:
        return {"skipped": True, "reason": "no_fault_outputs_in_mapping"}

    writes: list[dict[str, Any]] = []
    detail: list[dict[str, Any]] = []
    for flag_col, target in fault_outputs.items():
        if flag_col not in result_df.columns:
            detail.append(
                {
                    "flag": flag_col,
                    "skipped": True,
                    "reason": "column_missing_from_result",
                }
            )
            continue
        val = _last_flag_value(result_df[flag_col])
        if val is None:
            detail.append(
                {"flag": flag_col, "skipped": True, "reason": "no_last_value"}
            )
            continue
        writes.append(
            {
                "point": {"uuid": target.uuid, "layer": target.layer},
                "value": {"real": val},
            }
        )
        detail.append(
            {
                "flag": flag_col,
                "uuid": target.uuid,
                "layer": target.layer,
                "value": val,
                "skipped": False,
            }
        )

    if not writes:
        return {"ok": False, "reason": "no_writes_built", "detail": detail}

    res = client.post("/api/v2/command/write", json={"writes": writes})
    try:
        body = res.json()
    except Exception:
        body = {"raw": res.text[:2000]}
    if not res.ok:
        log.warning("command/write failed: %s %s", res.status_code, body)
        return {
            "ok": False,
            "status_code": res.status_code,
            "nf_response": body,
            "detail": detail,
        }
    return {"ok": True, "nf_response": body, "detail": detail}


def should_persist(cfg: AppConfig, request_persist: bool | None, env_default: bool) -> bool:
    if not cfg.fault_outputs:
        return False
    if request_persist is True:
        return True
    if request_persist is False:
        return False
    return env_default
