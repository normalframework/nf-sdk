"""Pydantic request/response models for agent-friendly OpenAPI."""

from __future__ import annotations

from typing import Any, Optional

from pydantic import BaseModel, Field


class RunRequest(BaseModel):
    """Optional body for POST /run — defaults preserve legacy one-click behavior."""

    lookback_hours: Optional[int] = Field(
        default=None,
        ge=1,
        le=8760,
        description="Override mapping fetch.lookback_hours for this run only.",
    )
    only_rule_files: Optional[list[str]] = Field(
        default=None,
        description='Run a subset of rules, e.g. ["sensor_bounds.yaml", "sensor_flatline"] (stem or filename).',
    )
    include_timeseries_stats: bool = Field(
        default=False,
        description="Per-brick non-null coverage on the ingested frame (before rules).",
    )
    include_columns_present: bool = Field(
        default=False,
        description="List all column names on the rule output DataFrame.",
    )
    sample_tail_rows: int = Field(
        default=0,
        ge=0,
        le=200,
        description="Include last N rows (timestamp + flags + sampled brick columns) for debugging.",
    )
    persist: Optional[bool] = Field(
        default=None,
        description="If true, write mapped fault flags to NF via command/write. If null, use OPENFDD_PERSIST_DEFAULT.",
    )
    dry_run: bool = Field(
        default=False,
        description="If true with persist, show would_write targets without calling NF.",
    )


class ValidateRuleBody(BaseModel):
    yaml: str = Field(..., min_length=1, description="Full or partial Open-FDD rule YAML to validate.")
