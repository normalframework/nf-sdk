"""Load 10 points from the object database

The points will not have any particular order, since we don't provide a filter.
"""

import sys
sys.path.append("../..")
import requests

from helpers import NfClient, print_response

client = NfClient()
res = client.get(
    "/api/v1/point/data",
    params={
        # ISO8601 timestamps
        #  or set from.seconds = unix timestamp
        "from": "2025-10-28T12:13:17.743Z",
        "to": "2025-11-04T12:13:17.743Z",
        "uuids": [
            "04461995-99fe-3161-9f80-3efc5e6ec95b",
            "6e4f9012-82ba-3a0e-a26c-bb84a5785a45",
            "8631d969-700f-3961-9ff8-6c697055322b",
            "a614ab9d-57ed-3e95-ab4e-ec6c2d232aec",
        ],
        # resample by time buckets -- MAX, MIN, AVERAGE, SUM, FIRST, LAST, COUNT
        #  see https://buf.build/normalframework/nf/docs/main:normalgw.hpl.v1#normalgw.hpl.v1.ResampleOptions.ResampleMethod
        "method": "MAX",

        # window size for resampling. ignored if not resampling
        "window": "900s",
    },
)
print_response(res)

