"""Inspect Open-FDD YAML rules on disk — for agents, inventory APIs, and filtering."""

from __future__ import annotations

from pathlib import Path
from typing import Any

from open_fdd.engine import load_rule


def iter_rules_with_files(rules_dir: str | Path) -> list[tuple[str, dict[str, Any]]]:
    """Return (filename, rule_dict) sorted by filename."""
    d = Path(rules_dir)
    if not d.is_dir():
        return []
    out: list[tuple[str, dict[str, Any]]] = []
    for f in sorted(d.glob("*.yaml")):
        try:
            out.append((f.name, load_rule(f)))
        except Exception:
            out.append((f.name, {"_load_error": True}))
    return out


def rule_summary(filename: str, rule: dict[str, Any]) -> dict[str, Any]:
    """Stable JSON shape for agents / GET /rules inventory."""
    if rule.get("_load_error"):
        return {"file": filename, "error": "yaml_load_failed"}
    inputs = rule.get("inputs") or {}
    bricks: list[str] = []
    if isinstance(inputs, dict):
        for _k, v in inputs.items():
            if isinstance(v, dict) and v.get("brick"):
                bricks.append(str(v["brick"]))
            elif isinstance(v, str):
                continue
    return {
        "file": filename,
        "name": rule.get("name"),
        "type": rule.get("type"),
        "flag": rule.get("flag"),
        "description": rule.get("description"),
        "equipment_type": rule.get("equipment_type"),
        "brick_inputs": bricks,
    }


def filter_rules_by_files(
    rules_dir: str | Path,
    only_files: list[str] | None,
) -> list[dict[str, Any]]:
    """Load rule dicts; if only_files set, keep stems or filenames that match."""
    loaded = iter_rules_with_files(rules_dir)
    if not only_files:
        return [r for _, r in loaded if not r.get("_load_error")]
    want = {s.strip().lower() for s in only_files if s.strip()}
    picked: list[dict[str, Any]] = []
    for fname, rule in loaded:
        if rule.get("_load_error"):
            continue
        stem = Path(fname).stem.lower()
        if fname.lower() in want or stem in want:
            picked.append(rule)
    return picked
