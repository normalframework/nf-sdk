"""
FastAPI surface for the NF sidecar — humans, cron, and agentic tools.

Hot reload: each POST /run builds a fresh open_fdd.engine.RuleRunner from disk (or filtered rule list).
"""

from __future__ import annotations

import importlib.metadata
import os
import re
from pathlib import Path
from typing import Annotated, Any, Optional

from fastapi import Body, Depends, FastAPI, Header, HTTPException
from pydantic import BaseModel, Field

from openfdd_nf.config import load_config
from openfdd_nf.rule_meta import iter_rules_with_files, rule_summary
from openfdd_nf.schemas import RunRequest, ValidateRuleBody
from openfdd_nf.service import COOKBOOK_URL, RunOptions, build_agent_brief, run_fault_pipeline

try:
    _OPENFDD_VER = importlib.metadata.version("open-fdd")
except importlib.metadata.PackageNotFoundError:
    _OPENFDD_VER = "unknown"


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


_DEPS = Depends(_optional_bearer)

app = FastAPI(
    title="Open-FDD × Normal Framework (sidecar)",
    description=(
        "Runs Open-FDD YAML fault rules against NF HPL timeseries (`mapping.yaml` → Brick columns). "
        "Designed for **operators**, **scheduled jobs**, and **AI agents** (OpenAPI + structured runs). "
        "Expression rules: see the [Open-FDD expression cookbook]("
        + COOKBOOK_URL
        + "). Normal Applications SDK hooks can call `POST /run` on a schedule."
    ),
    version="0.3.0",
    openapi_tags=[
        {"name": "health", "description": "Liveness and capability discovery."},
        {"name": "rules", "description": "Rule inventory and YAML inspection."},
        {"name": "run", "description": "Execute fault detection against NF data."},
        {"name": "agent", "description": "Compact context for LLM / external agents."},
        {"name": "validate", "description": "Check rule YAML before deploy."},
    ],
)


class CapabilitiesResponse(BaseModel):
    service: str = "openfdd-nf-sidecar"
    sidecar_version: str = "0.3.0"
    open_fdd_pypi_version: str = Field(default_factory=lambda: _OPENFDD_VER)
    cookbook_url: str = COOKBOOK_URL
    endpoints: list[str] = Field(
        default_factory=lambda: [
            "GET /health",
            "GET /capabilities",
            "GET /agent/context",
            "GET /mapping",
            "GET /rules",
            "GET /rules/{filename}",
            "POST /run",
            "POST /validate/rule",
        ]
    )
    features: dict[str, bool] = Field(
        default_factory=lambda: {
            "nf_command_write_persistence": True,
            "rule_subset_filter": True,
            "yaml_hot_reload": True,
        }
    )


@app.get("/health", tags=["health"])
def health():
    return {
        "status": "ok",
        "service": "openfdd-nf-sidecar",
        "open_fdd": _OPENFDD_VER,
    }


@app.get("/capabilities", response_model=CapabilitiesResponse, tags=["health"])
def capabilities(_: Annotated[None, _DEPS]):
    return CapabilitiesResponse()


@app.get("/agent/context", tags=["agent"])
def agent_context(
    _: Annotated[None, _DEPS],
    redact_uuids: bool = False,
):
    """
    Structured + human-readable context for LLM agents (mapping, cookbook link, API hints).

    Set `redact_uuids=true` to mask UUIDs in logs (prefix only).
    """
    path = os.environ.get("OPENFDD_MAPPING", "mapping.yaml")
    try:
        cfg = load_config(path)
    except FileNotFoundError as e:
        raise HTTPException(status_code=404, detail=str(e)) from e

    pu = cfg.point_uuids
    if redact_uuids:
        pu = {k: (v[:8] + "…" if len(v) > 8 else v) for k, v in pu.items()}

    rules_dir = cfg.run.rules_dir
    inv = [rule_summary(fn, r) for fn, r in iter_rules_with_files(rules_dir)]
    files = [x["file"] for x in inv if "error" not in x]

    return {
        "cookbook_url": COOKBOOK_URL,
        "nf_base_url": cfg.nf.base_url,
        "brick_to_point_uuid": pu,
        "fetch_defaults": {
            "lookback_hours": cfg.fetch.lookback_hours,
            "resample_method": cfg.fetch.resample_method,
            "resample_window": cfg.fetch.resample_window,
        },
        "rules_inventory": inv,
        "fault_output_flags_configured": list(cfg.fault_outputs.keys()),
        "agent_brief_markdown": build_agent_brief(cfg, files),
        "openapi": "/openapi.json",
        "docs": "/docs",
    }


@app.get("/mapping", tags=["agent"])
def get_mapping(
    _: Annotated[None, _DEPS],
    redact_uuids: bool = False,
):
    """Return current mapping-derived config (no secrets)."""
    path = os.environ.get("OPENFDD_MAPPING", "mapping.yaml")
    try:
        cfg = load_config(path)
    except FileNotFoundError as e:
        raise HTTPException(status_code=404, detail=str(e)) from e
    pu = cfg.point_uuids
    if redact_uuids:
        pu = {k: (v[:8] + "…" if len(v) > 8 else v) for k, v in pu.items()}
    outs = {
        k: {"uuid": (v.uuid[:8] + "…" if redact_uuids and len(v.uuid) > 8 else v.uuid), "layer": v.layer}
        for k, v in cfg.fault_outputs.items()
    }
    return {
        "nf_base_url": cfg.nf.base_url,
        "point_uuids": pu,
        "fault_outputs": outs,
        "fetch": {
            "lookback_hours": cfg.fetch.lookback_hours,
            "resample_method": cfg.fetch.resample_method,
            "resample_window": cfg.fetch.resample_window,
        },
        "rules_dir": cfg.run.rules_dir,
    }


@app.get("/rules", tags=["rules"])
def list_rules(_: Annotated[None, _DEPS]):
    """Inventory all `*.yaml` rules with parsed metadata (name, type, flag, brick_inputs)."""
    path = os.environ.get("OPENFDD_MAPPING", "mapping.yaml")
    try:
        cfg = load_config(path)
    except FileNotFoundError as e:
        raise HTTPException(status_code=404, detail=str(e)) from e
    rules_dir = Path(cfg.run.rules_dir)
    if not rules_dir.is_dir():
        return {"rules_dir": str(rules_dir), "rules": [], "count": 0}
    inv = [rule_summary(fn, r) for fn, r in iter_rules_with_files(rules_dir)]
    return {"rules_dir": str(rules_dir), "rules": inv, "count": len(inv)}


def _safe_rule_filename(name: str) -> str:
    base = Path(name).name
    if not re.match(r"^[a-zA-Z0-9][a-zA-Z0-9_.-]*\.yaml$", base):
        raise HTTPException(status_code=400, detail="invalid rule filename")
    return base


@app.get("/rules/{filename}", tags=["rules"])
def get_rule_yaml(filename: str, _: Annotated[None, _DEPS]):
    """Raw YAML text for one rule file — for agents or humans reviewing before edit."""
    safe = _safe_rule_filename(filename)
    path = os.environ.get("OPENFDD_MAPPING", "mapping.yaml")
    try:
        cfg = load_config(path)
    except FileNotFoundError as e:
        raise HTTPException(status_code=404, detail=str(e)) from e
    fpath = Path(cfg.run.rules_dir) / safe
    if not fpath.is_file():
        raise HTTPException(status_code=404, detail=f"rule file not found: {safe}")
    return {
        "file": safe,
        "yaml": fpath.read_text(encoding="utf-8"),
    }


@app.post("/run", tags=["run"])
def trigger_run(
    _: Annotated[None, _DEPS],
    body: RunRequest | None = Body(default=None),
):
    """
    Pull NF timeseries, run rules (full reload from disk), return structured summary.

    Optional JSON body for agents: subset of rules, stats, tail sample, NF write-back.
    """
    path = os.environ.get("OPENFDD_MAPPING", "mapping.yaml")
    b = body or RunRequest()
    opts = RunOptions(
        lookback_hours=b.lookback_hours,
        only_rule_files=b.only_rule_files,
        include_timeseries_stats=b.include_timeseries_stats,
        include_columns_present=b.include_columns_present,
        sample_tail_rows=b.sample_tail_rows,
        persist=b.persist,
        dry_run=b.dry_run,
    )
    try:
        return run_fault_pipeline(path, opts)
    except FileNotFoundError as e:
        raise HTTPException(status_code=404, detail=str(e)) from e
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e)) from e


@app.post("/validate/rule", tags=["validate"])
def validate_rule(
    _: Annotated[None, _DEPS],
    body: ValidateRuleBody,
):
    """Parse YAML and check minimal Open-FDD rule keys (does not execute against data)."""
    import yaml

    try:
        data = yaml.safe_load(body.yaml)
    except yaml.YAMLError as e:
        raise HTTPException(status_code=400, detail=f"yaml_error: {e}") from e
    if not isinstance(data, dict):
        raise HTTPException(status_code=400, detail="root_must_be_mapping")
    issues: list[str] = []
    if "name" not in data:
        issues.append("missing_name")
    if "type" not in data:
        issues.append("missing_type")
    if "flag" not in data:
        issues.append("missing_flag")
    rtype = data.get("type")
    if rtype == "expression":
        if "expression" not in data:
            issues.append("expression_type_missing_expression")
        if "inputs" not in data:
            issues.append("expression_type_missing_inputs")
    elif rtype in ("bounds", "flatline", "hunting", "oa_fraction", "erv_efficiency"):
        if "inputs" not in data:
            issues.append(f"{rtype}_missing_inputs")
    elif rtype:
        issues.append(f"unknown_type:{rtype}")
    ok = len(issues) == 0
    return {"ok": ok, "issues": issues, "preview": rule_summary("inline.yaml", data)}
