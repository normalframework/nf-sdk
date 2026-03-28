import textwrap
from pathlib import Path

import pandas as pd
import pytest

from open_fdd.engine import RuleRunner


@pytest.fixture
def rules_dir(tmp_path: Path) -> Path:
    d = tmp_path / "rules"
    d.mkdir()
    (d / "bounds.yaml").write_text(
        textwrap.dedent(
            """
            name: test_bounds
            type: bounds
            flag: test_bounds_flag
            inputs:
              Supply_Air_Temperature_Sensor:
                brick: Supply_Air_Temperature_Sensor
                bounds:
                  imperial: [50, 90]
            params:
              units: imperial
              rolling_window: 1
            """
        ).strip(),
        encoding="utf-8",
    )
    return d


def synthetic_df() -> pd.DataFrame:
    ts = pd.date_range("2025-01-01", periods=5, freq="15min", tz="UTC")
    return pd.DataFrame(
        {
            "timestamp": ts,
            "Supply_Air_Temperature_Sensor": [55.0, 55.0, 120.0, 55.0, 55.0],
        }
    )


def test_rule_runner_flags_out_of_bounds(rules_dir: Path):
    runner = RuleRunner(rules_path=rules_dir)
    out = runner.run(
        synthetic_df(),
        timestamp_col="timestamp",
        skip_missing_columns=True,
    )
    assert "test_bounds_flag" in out.columns
    assert out["test_bounds_flag"].sum() >= 1
