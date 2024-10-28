"""Load 10 points from the object database where device_id=260001,
using a field mask to only return certain fields
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
            # use a numeric query.  For a numeric query to work, the
            # field has to be indexed as NUMERIC in the layer
            # definition.
            "numeric": {
                "min_value": 260001,
                "max_value": 260001
            },
        }
    },
    # 
    "masks": {
        # fields are attributes on the point object
        "field_mask": ["uuid", "latest_value"],
        # attrs_include_mask whitelists certain attributes which
        # appear in layers
        "attr_include_mask": ["type", "instance"]
    }
})
print_response(res)
