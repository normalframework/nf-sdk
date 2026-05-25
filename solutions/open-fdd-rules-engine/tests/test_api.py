"""FastAPI sidecar: health, capabilities, /rules, /run (mocked)."""

from pathlib import Path
from unittest.mock import patch

import pytest

pytest.importorskip("fastapi")
from fastapi.testclient import TestClient

from openfdd_nf.api_app import app
from openfdd_nf.config import AppConfig, FetchConfig, NFConfig, RunConfig

client = TestClient(app)
_RULES_DIR = Path(__file__).resolve().parent.parent / "rules"


def _minimal_cfg() -> AppConfig:
    return AppConfig(
        nf=NFConfig(base_url="http://x"),
        fetch=FetchConfig(),
        run=RunConfig(rules_dir=str(_RULES_DIR)),
        point_uuids={"Outside_Air_Temperature_Sensor": "u1"},
        fault_outputs={},
    )


def test_health():
    r = client.get("/health")
    assert r.status_code == 200
    assert r.json()["status"] == "ok"
    assert "open_fdd" in r.json()


def test_capabilities():
    r = client.get("/capabilities")
    assert r.status_code == 200
    data = r.json()
    assert data["service"] == "openfdd-nf-sidecar"
    assert "POST /run" in data["endpoints"]


@patch("openfdd_nf.api_app.load_config", return_value=_minimal_cfg())
def test_rules_lists_yaml(_mock):
    r = client.get("/rules")
    assert r.status_code == 200
    data = r.json()
    files = [x["file"] for x in data["rules"]]
    assert "sensor_bounds.yaml" in files or "sensor_flatline.yaml" in files


@patch("openfdd_nf.api_app.run_fault_pipeline")
def test_run_returns_summary(mock_run):
    mock_run.return_value = {
        "run_id": "rid",
        "rows": 10,
        "flags": {"flatline_flag": {"true_count": 0, "last": 0}},
    }
    r = client.post("/run", json={})
    assert r.status_code == 200
    assert r.json()["rows"] == 10


@patch("openfdd_nf.api_app.run_fault_pipeline")
def test_run_accepts_agent_options(mock_run):
    mock_run.return_value = {"run_id": "x"}
    r = client.post(
        "/run",
        json={
            "lookback_hours": 48,
            "only_rule_files": ["sensor_bounds"],
            "include_timeseries_stats": True,
        },
    )
    assert r.status_code == 200
    mock_run.assert_called_once()
    call_args = mock_run.call_args[0]
    assert len(call_args) >= 2
    opts = call_args[1]
    assert opts.lookback_hours == 48
    assert opts.only_rule_files == ["sensor_bounds"]
    assert opts.include_timeseries_stats is True


@patch.dict("os.environ", {"OPENFDD_API_KEY": "secret"}, clear=False)
@patch("openfdd_nf.api_app.load_config", return_value=_minimal_cfg())
def test_bearer_required_when_configured(_mock_load):
    c = TestClient(app)
    assert c.get("/rules").status_code == 401
    r = c.get("/rules", headers={"Authorization": "Bearer secret"})
    assert r.status_code == 200


def test_validate_rule_ok():
    yaml = """
name: t
type: expression
flag: f
inputs:
  Outside_Air_Temperature_Sensor:
    brick: Outside_Air_Temperature_Sensor
expression: "Outside_Air_Temperature_Sensor > 0"
"""
    r = client.post("/validate/rule", json={"yaml": yaml})
    assert r.status_code == 200
    body = r.json()
    assert body["ok"] is True
    assert body["issues"] == []


def test_validate_rule_bad_yaml():
    r = client.post("/validate/rule", json={"yaml": ":\n\tnot yaml"})
    assert r.status_code == 400


@patch("openfdd_nf.api_app.load_config", return_value=_minimal_cfg())
def test_get_rule_yaml_file(_mock):
    r = client.get("/rules/sensor_bounds.yaml")
    assert r.status_code == 200
    assert "type: bounds" in r.json()["yaml"]
