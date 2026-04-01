"""Offline CSV → RuleRunner path (no NF)."""

from pathlib import Path

import pandas as pd
import pytest

from openfdd_nf.offline_csv import (
    dataframe_from_mapped_csv,
    load_column_map,
    run_offline,
    summarize_faults,
)

_ROOT = Path(__file__).resolve().parent.parent
RTU_CSV = _ROOT / "examples" / "AHU" / "RTU11.csv"
RTU_MAP = _ROOT / "examples" / "AHU" / "rtu11_column_map.yaml"
RULES = _ROOT / "examples" / "AHU" / "rules_demo"


@pytest.mark.skipif(not RTU_CSV.is_file(), reason="RTU11.csv not present")
def test_run_offline_rtu11_subset():
    _df, payload = run_offline(RTU_CSV, RTU_MAP, RULES, max_rows=120)
    s = payload["summary"]
    assert s["rows"] == 120
    assert "bad_sensor_flag" in s["flags"]
    assert "meta" in payload
    assert "cookbook_url" in payload["meta"]


def test_dataframe_from_mapped_csv_minimal(tmp_path: Path):
    p = tmp_path / "t.csv"
    p.write_text(
        "Timestamp,RTU_11_OA_T(°F),RTU_11_DA_T(°F)\n"
        "01-Jan-26 12:00:00 AM EST,40.0,70.0\n"
        "01-Jan-26 1:00:00 AM EST,41.0,71.0\n",
        encoding="utf-8",
    )
    m = tmp_path / "m.yaml"
    m.write_text(
        """
timestamp_column: Timestamp
rename:
  "RTU_11_OA_T(°F)": Outside_Air_Temperature_Sensor
  "RTU_11_DA_T(°F)": Supply_Air_Temperature_Sensor
""",
        encoding="utf-8",
    )
    spec = load_column_map(m)
    df = dataframe_from_mapped_csv(p, spec)
    assert "timestamp" in df.columns
    assert "Outside_Air_Temperature_Sensor" in df.columns
    assert len(df) == 2


def test_summarize_faults_empty():
    df = pd.DataFrame({"timestamp": pd.date_range("2025-01-01", periods=2, freq="h", tz="UTC")})
    assert summarize_faults(df)["rows"] == 2
    assert summarize_faults(df)["flags"] == {}
