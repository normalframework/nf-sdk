"""Load 10 points from the object database

The points will not have any particular order, since we don't provide a filter.
"""

import sys
sys.path.append("../..")
import requests

from helpers import NfClient, print_response

import requests
import pandas as pd
from datetime import datetime, timedelta, timezone
from urllib.parse import urlencode
import sys

def download_csv_from_api(
        client,
    uuids: list,
    headers: list,
    start: datetime,
    end: datetime
):
    if len(uuids) != len(headers):
        raise ValueError("Length of uuids and headers must match")

    uuid_to_header = dict(zip(uuids, headers))
    all_data = {}

    # Ensure times are UTC with Z suffix
    if start.tzinfo is None:
        start = start.replace(tzinfo=timezone.utc)
    else:
        start = start.astimezone(timezone.utc)

    if end.tzinfo is None:
        end = end.replace(tzinfo=timezone.utc)
    else:
        end = end.astimezone(timezone.utc)

    current = start
    while current < end:
        chunk_end = min(current + timedelta(days=1), end)

        params = {
            "uuids": uuids,
            "from": current.strftime('%Y-%m-%dT%H:%M:%SZ'),
            "to": chunk_end.strftime('%Y-%m-%dT%H:%M:%SZ'),
            "method": "FIRST",
            "window": '900s',
        }

        query = urlencode(params, doseq=True)
        url = f"/api/v1/point/data?{query}"
        resp = client.get(url)

        if resp.status_code != 200:
            raise RuntimeError(f"Failed to fetch data: {resp.status_code} - {resp.text}")

        json_data = resp.json()
        for series in json_data.get("data", []):
            uuid = series["uuid"]
            col_name = uuid_to_header.get(uuid, uuid)
            for entry in series.get("values", []):
                ts = entry["ts"]
                val = entry.get("double", None)
                all_data.setdefault(ts, {})[col_name] = val

        current = chunk_end

    # Create dataframe
    df = pd.DataFrame.from_dict(all_data, orient="index")
    df.index.name = "timestamp"
    df.sort_index(inplace=True)

    # Ensure all headers are present even if data was missing
    df = df.reindex(columns=headers)

    df.to_csv(sys.stdout, na_rep="")


    
client = NfClient()
res = client.post("/api/v1/point/query", json={
    "page_size": 500,
    "structuredQuery": {
        "and": [
            {
                "field": {
                    "property": "equipRef",
                    "text": "Diggs_RTU7"
                }
            },
            {
                "or": [
                    {
                        "field": {
                            "property": "period",
                            "numeric": {
                                "minValue": 30,
                                "maxValue": 30
                            }
                        }
                    },
                    {
                        "field": {
                            "property": "period",
                            "numeric": {
                                "minValue": 300,
                                "maxValue": 300
                            }
                        }
                    },
                    {
                        "field": {
                            "property": "period",
                            "numeric": {
                                "minValue": 60,
                                "maxValue": 60
                            }
                        }
                    },
                    {
                        "field": {
                            "property": "period",
                            "numeric": {
                                "minValue": 900,
                                "maxValue": 900
                            }
                        }
                    }
                ]
            }
        ]
    }})


points = res.json()
headers = [p["name"] for p in points["points"]]
uuids = [p["uuid"] for p in points["points"]]

download_csv_from_api(client, uuids, headers, start=datetime(2025, 5, 10), end=datetime(2025, 5, 16))
