"""
Optional FastAPI surface for the NF sidecar — AFDD-style trigger + rule inventory.

Hot reload: each POST /run builds a fresh open_fdd.engine.RuleRunner from disk (same idea as
Open-FDD's loop loading rules every cycle). Mount or bind-mount ./rules to change YAML without
rebuilding the image.
"""

from __future__ import annotations

import os
from pathlib import Path
from typing import Annotated, Optional

from fastapi import Depends, FastAPI, Header, HTTPException

from openfdd_nf.config import load_config
from openfdd_nf.service import run_once

app = FastAPI(
    title="Open-FDD × Normal Framework (sidecar)",
    description=(
        "Runs the same YAML-driven Open-FDD RuleRunner as the Open-FDD AFDD stack, fed by "
        "NF HPL timeseries (mapping.yaml). Complements Normal's managed JavaScript hooks "
        "(see Applications SDK); this service stays outside the NF hook sandbox."
    ),
    version="0.2.0",
)


def _optional_bearer(
    authorization: Annotated[Optional[str], Header()] = None,
) -> None:
    expected = os.environ.get("OPENFDD_API_KEY", "").strip()
    if not expected:
        return
    if not authorization or not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="Missing or invalid Authorization")
    token = authorization.removeprefix("Bearer ").strip()
    if token != expected:
        raise HTTPException(status_code=401, detail="Invalid API key")


@app.get("/health")
def health():
    return {"status": "ok", "service": "openfdd-nf-sidecar"}


@app.get("/rules")
def list_rules(
    _: Annotated[None, Depends(_optional_bearer)],
):
    """Rule file names under rules_dir (YAML hot-reload source; same semantics as Open-FDD rules_dir)."""
    path = os.environ.get("OPENFDD_MAPPING", "mapping.yaml")
    try:
        cfg = load_config(path)
    except FileNotFoundError as e:
        raise HTTPException(status_code=404, detail=str(e)) from e
    rules_dir = Path(cfg.run.rules_dir)
    if not rules_dir.is_dir():
        return {"rules_dir": str(rules_dir), "files": []}
    files = sorted(p.name for p in rules_dir.glob("*.yaml"))
    return {"rules_dir": str(rules_dir), "files": files, "count": len(files)}


@app.post("/run")
def trigger_run(
    _: Annotated[None, Depends(_optional_bearer)],
):
    """
    Pull NF timeseries, run all rules, return JSON summary (same payload shape as stdout mode).

    Re-loads rule YAML from disk on every call — equivalent to hot reload in the Open-FDD platform loop.
    """
    path = os.environ.get("OPENFDD_MAPPING", "mapping.yaml")
    try:
        return run_once(path)
    except FileNotFoundError as e:
        raise HTTPException(status_code=404, detail=str(e)) from e
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e)) from e
