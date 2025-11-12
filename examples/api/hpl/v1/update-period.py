
"""Load 10 points from the object database

The points will not have any particular order, since we don't provide a filter.
"""

import sys
sys.path.append("../..")
import requests

from helpers import NfClient, print_response

client = NfClient()

res = client.get("/api/v1/point/points", params={
    "layer": "hpl:bacnet:1",
    "page_size": 2,
})
uids = [p["uuid"] for p in res.json()["points"]]
print_response(res)

res = client.post("/api/v1/point/points", json={
    "points": [
        {
            "uuid": u,
            "period": "30s",
        }
        for u in uids
    ]
})
print_response(res)

