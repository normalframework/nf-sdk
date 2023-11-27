"""Load 10 points from the object database

The points will not have any particular order, since we don't provide a filter.
"""

import os
import sys
import json
import requests

nfurl = os.getenv("NFURL", "http://localhost:8080")
res = requests.post(nfurl + "/api/v1/point/query", json={
    "page_size": 10
})
json.dump(res.json(), sys.stdout, indent=2)
