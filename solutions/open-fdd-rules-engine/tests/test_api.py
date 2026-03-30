"""FastAPI sidecar: health, /rules, /run (mocked)."""

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
    )


def test_health():
    r = client.get("/health")
    assert r.status_code == 200
    assert r.json()["status"] == "ok"


@patch("openfdd_nf.api_app.load_config", return_value=_minimal_cfg())
def test_rules_lists_yaml(_mock):
    r = client.get("/rules")
    assert r.status_code == 200
    data = r.json()
    names = data["files"]
    assert "sensor_bounds.yaml" in names or "sensor_flatline.yaml" in names


@patch("openfdd_nf.api_app.run_once")
def test_run_returns_summary(mock_run):
    mock_run.return_value = {
        "rows": 10,
        "flags": {"flatline_flag": {"true_count": 0, "last": 0}},
    }
    r = client.post("/run")
    assert r.status_code == 200
    assert r.json()["rows"] == 10


@patch.dict("os.environ", {"OPENFDD_API_KEY": "secret"}, clear=False)
@patch("openfdd_nf.api_app.load_config", return_value=_minimal_cfg())
def test_bearer_required_when_configured(_mock_load):
    c = TestClient(app)
    assert c.get("/rules").status_code == 401
    r = c.get("/rules", headers={"Authorization": "Bearer secret"})
    assert r.status_code == 200
