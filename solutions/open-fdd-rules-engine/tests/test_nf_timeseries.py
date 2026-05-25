import json
from pathlib import Path

import pandas as pd

from openfdd_nf.nf_timeseries import nf_response_to_dataframe


def test_nf_response_to_dataframe_maps_uuids_to_brick_columns():
    fixture = Path(__file__).parent / "fixtures" / "sample_point_data.json"
    payload = json.loads(fixture.read_text(encoding="utf-8"))
    uuid_to_brick = {
        "11111111-1111-1111-1111-111111111111": "Outside_Air_Temperature_Sensor",
        "22222222-2222-2222-2222-222222222222": "Supply_Air_Temperature_Sensor",
    }
    df = nf_response_to_dataframe(payload, uuid_to_brick)
    assert len(df) == 2
    assert "timestamp" in df.columns
    assert df["Outside_Air_Temperature_Sensor"].tolist() == [35.0, 36.0]
    assert df["Supply_Air_Temperature_Sensor"].tolist() == [72.0, 72.0]
    assert pd.api.types.is_datetime64_any_dtype(df["timestamp"])
