"""Load 10 points from the object database where device_id=260001
"""

import sys
sys.path.append("../..")
import requests

from helpers import NfClient, print_response

client = NfClient()
res = client.post("/api/v1/point/query", json={
    "page_size": 10,
    "structured_query": {
        "field": {
            # property is the attribute name.
            "property": "device_id",
            # device id is actually indexed as a tag field
            "text": "260001",
        }
    }
})
print_response(res)
