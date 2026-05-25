"""Fetch NF /api/v1/point/data and build a wide pandas DataFrame (Brick-named columns)."""

from __future__ import annotations

from datetime import datetime, timedelta, timezone
from typing import Any
from urllib.parse import urlencode

import pandas as pd

from openfdd_nf.nf_client import NFClient


def nf_response_to_dataframe(
    json_data: dict[str, Any],
    uuid_to_brick: dict[str, str],
) -> pd.DataFrame:
    """
    Parse NF HPL point/data JSON into a DataFrame with columns named by Brick class.

    Expected shape (per nf-sdk examples/api/hpl/v1/download-csv.py):
    { "data": [ { "uuid": "...", "values": [ {"ts": "...", "double": 1.2 }, ... ] }, ... ] }
    """
    rows: dict[str, dict[str, float | None]] = {}
    for series in json_data.get("data", []):
        uid = series.get("uuid")
        if not uid or uid not in uuid_to_brick:
            continue
        col = uuid_to_brick[uid]
        for entry in series.get("values", []):
            ts = entry.get("ts")
            if ts is None:
                continue
            val = entry.get("double")
            rows.setdefault(ts, {})[col] = val

    if not rows:
        return pd.DataFrame()

    df = pd.DataFrame.from_dict(rows, orient="index")
    df.index = pd.to_datetime(df.index, utc=True)
    df.index.name = "timestamp"
    df = df.sort_index().reset_index()
    return df


def fetch_point_data_chunk(
    client: NFClient,
    uuids: list[str],
    start: datetime,
    end: datetime,
    method: str,
    window: str,
) -> dict[str, Any]:
    if start.tzinfo is None:
        start = start.replace(tzinfo=timezone.utc)
    else:
        start = start.astimezone(timezone.utc)
    if end.tzinfo is None:
        end = end.replace(tzinfo=timezone.utc)
    else:
        end = end.astimezone(timezone.utc)

    params: dict[str, Any] = {
        "uuids": uuids,
        "from": start.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "to": end.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "method": method,
        "window": window,
    }
    query = urlencode(params, doseq=True)
    res = client.get(f"/api/v1/point/data?{query}")
    res.raise_for_status()
    return res.json()


def fetch_timeseries(
    client: NFClient,
    brick_to_uuid: dict[str, str],
    lookback_hours: int,
    method: str,
    window: str,
) -> pd.DataFrame:
    """Chunked GET /api/v1/point/data (1-day steps), same merge strategy as download-csv.py."""
    uuids = list(brick_to_uuid.values())
    uuid_to_brick = {v: k for k, v in brick_to_uuid.items()}
    end = datetime.now(timezone.utc)
    start = end - timedelta(hours=lookback_hours)

    all_data: dict[str, dict[str, float | None]] = {}
    current = start
    while current < end:
        chunk_end = min(current + timedelta(days=1), end)
        payload = fetch_point_data_chunk(
            client, uuids, current, chunk_end, method, window
        )
        for series in payload.get("data", []):
            uid = series.get("uuid")
            if not uid or uid not in uuid_to_brick:
                continue
            col = uuid_to_brick[uid]
            for entry in series.get("values", []):
                ts = entry.get("ts")
                if ts is None:
                    continue
                val = entry.get("double")
                all_data.setdefault(ts, {})[col] = val
        current = chunk_end

    if not all_data:
        return pd.DataFrame()

    df = pd.DataFrame.from_dict(all_data, orient="index")
    df.index = pd.to_datetime(df.index, utc=True)
    df.index.name = "timestamp"
    df = df.sort_index().reset_index()
    for col in brick_to_uuid:
        if col not in df.columns:
            df[col] = pd.NA
    return df
