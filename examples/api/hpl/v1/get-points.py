"""Load 10 points from the object database

The points will not have any particular order, since we don't provide a filter.
"""

import sys
sys.path.append("../..")
import requests

from helpers import NfClient, print_response

client = NfClient()
res = client.post("/api/v1/point/query", json={
    "layer": "default",
    # you can export a structured query from Object Explorer by
    # clicking on the "..." in the top right, and using the "Export
    # cURL" option.  Writing structured queries by hand is not very easy.
    "structuredQuery": {
        "field": {
            "property": "point_type",
            "text": "POINT"
        }
    },
    # this returns the attributes from different layers separately
    #  if you don't send this, it returns them all in one attrs dictionary; which is easier to use
    #  but will be incorrect if there are duplicate attribute names in different layers
    "responseFormat": "LAYERS_SPLIT",
    "pageSize": "100"
})
print_response(res)

