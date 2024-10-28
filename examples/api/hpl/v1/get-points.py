"""Load 10 points from the object database

The points will not have any particular order, since we don't provide a filter.
"""

import sys
sys.path.append("../..")
import requests

from helpers import NfClient, print_response

client = NfClient()
res = client.post("/api/v1/point/query", json={
    "page_size": 10
})
print_response(res)

