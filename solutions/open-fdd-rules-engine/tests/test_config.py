import textwrap
from pathlib import Path

import pytest

from openfdd_nf.config import load_config


def test_load_config_minimal(tmp_path: Path):
    rules = tmp_path / "r"
    rules.mkdir()
    p = tmp_path / "m.yaml"
    p.write_text(
        textwrap.dedent(
            """
            nf:
              base_url: "http://example.test"
            run:
              rules_dir: "{rules}"
            point_uuids:
              Outside_Air_Temperature_Sensor: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
              Supply_Air_Temperature_Sensor: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
            """
        )
        .strip()
        .format(rules=str(rules)),
        encoding="utf-8",
    )
    cfg = load_config(p)
    assert cfg.nf.base_url == "http://example.test"
    assert "Outside_Air_Temperature_Sensor" in cfg.point_uuids


def test_load_config_missing_file():
    with pytest.raises(FileNotFoundError):
        load_config("/nonexistent/mapping.yaml")


def test_load_config_fault_outputs(tmp_path: Path):
    rules = tmp_path / "r"
    rules.mkdir()
    p = tmp_path / "m.yaml"
    p.write_text(
        textwrap.dedent(
            """
            run:
              rules_dir: "{rules}"
            point_uuids:
              Outside_Air_Temperature_Sensor: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
            fault_outputs:
              bad_sensor_flag:
                uuid: "cccccccc-cccc-cccc-cccc-cccccccccccc"
                layer: "hpl:bacnet:1"
            """
        )
        .strip()
        .format(rules=str(rules)),
        encoding="utf-8",
    )
    cfg = load_config(p)
    assert "bad_sensor_flag" in cfg.fault_outputs
    assert cfg.fault_outputs["bad_sensor_flag"].layer == "hpl:bacnet:1"


def test_load_config_runner_column_map(tmp_path: Path):
    rules = tmp_path / "r"
    rules.mkdir()
    p = tmp_path / "m.yaml"
    p.write_text(
        textwrap.dedent(
            """
            run:
              rules_dir: "{rules}"
            point_uuids:
              Supply_Air_Temperature_Sensor: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
            runner_column_map:
              discharge_air_temp_sensor: Supply_Air_Temperature_Sensor
            """
        )
        .strip()
        .format(rules=str(rules)),
        encoding="utf-8",
    )
    cfg = load_config(p)
    assert cfg.runner_column_map == {
        "discharge_air_temp_sensor": "Supply_Air_Temperature_Sensor",
    }
