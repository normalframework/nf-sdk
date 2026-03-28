from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml


@dataclass
class NFConfig:
    base_url: str


@dataclass
class FetchConfig:
    lookback_hours: int = 24
    resample_method: str = "AVERAGE"
    resample_window: str = "900s"


@dataclass
class RunConfig:
    rules_dir: str
    poll_interval_seconds: int = 300
    once: bool = False


@dataclass
class AppConfig:
    nf: NFConfig
    fetch: FetchConfig
    run: RunConfig
    point_uuids: dict[str, str]


def load_config(path: str | Path | None = None) -> AppConfig:
    p = Path(path or os.environ.get("OPENFDD_MAPPING", "mapping.yaml"))
    if not p.is_file():
        raise FileNotFoundError(
            f"Mapping file not found: {p}. Copy mapping.example.yaml to mapping.yaml "
            "or set OPENFDD_MAPPING."
        )
    raw: dict[str, Any] = yaml.safe_load(p.read_text(encoding="utf-8")) or {}

    nf_raw = raw.get("nf") or {}
    base = (
        os.environ.get("NFURL")
        or nf_raw.get("base_url")
        or "http://localhost:8080"
    ).rstrip("/")

    fetch_raw = raw.get("fetch") or {}
    run_raw = raw.get("run") or {}

    rules_dir = os.environ.get("OPENFDD_RULES_DIR") or run_raw.get("rules_dir")
    if not rules_dir:
        raise ValueError("run.rules_dir or OPENFDD_RULES_DIR is required")

    points = raw.get("point_uuids") or {}
    if not isinstance(points, dict) or not points:
        raise ValueError("point_uuids must be a non-empty map of BrickClass -> UUID")

    return AppConfig(
        nf=NFConfig(base_url=base),
        fetch=FetchConfig(
            lookback_hours=int(fetch_raw.get("lookback_hours", 24)),
            resample_method=str(fetch_raw.get("resample_method", "AVERAGE")),
            resample_window=str(fetch_raw.get("resample_window", "900s")),
        ),
        run=RunConfig(
            rules_dir=str(rules_dir),
            poll_interval_seconds=int(
                run_raw.get("poll_interval_seconds", 300)
            ),
            once=bool(run_raw.get("once", False)),
        ),
        point_uuids={str(k): str(v) for k, v in points.items()},
    )
