"""Write BACnet using the command service.

Using the Command service instead of the BACnet service means that
there is no need to manually construct the BACnet requests, or to
split requests up by device.

Example results:
{
  "results": [
    {
      "point": {
        "uuid": "7b79ab76-4f31-32ce-810a-78731c62b6fc",
        "layer": "hpl:bacnet:1"
      },
      "value": {
        "@type": "type.googleapis.com/normalgw.bacnet.v2.ApplicationDataValue",
        "real": 0
      },
      "scalar": "0"
    },
    {
      "point": {
        "uuid": "a70cd4bc-1579-370a-9ef2-e40f8e1784a3",
        "layer": "hpl:bacnet:1"
      },
      "value": {
        "@type": "type.googleapis.com/normalgw.bacnet.v2.ApplicationDataValue",
        "real": 0
      },
      "scalar": "0"
    },
    {
      "point": {
        "uuid": "ad20c463-3300-3274-9a1e-7349bc2d4915",
        "layer": "hpl:bacnet:1"
      },
      "value": {
        "@type": "type.googleapis.com/normalgw.bacnet.v2.ApplicationDataValue",
        "real": 0
      },
      "scalar": "0"
    },
    {
      "point": {
        "uuid": "ab8bc8ca-3b25-362e-820e-7dbce42da398",
        "layer": "hpl:bacnet:1"
      },
      "value": {
        "@type": "type.googleapis.com/normalgw.bacnet.v2.ApplicationDataValue",
        "real": 0
      },
      "scalar": "0"
    },
    {
      "point": {
        "uuid": "48a79186-8dee-300a-9f99-b1cb7da627d3",
        "layer": "hpl:bacnet:1"
      },
      "value": {
        "@type": "type.googleapis.com/normalgw.bacnet.v2.ApplicationDataValue",
        "real": 50
      },
      "scalar": "50"
    },
    {
      "point": {
        "uuid": "bd13cf0f-9aac-3117-8ddd-e174c06d2c96",
        "layer": "hpl:bacnet:1"
      },
      "value": {
        "@type": "type.googleapis.com/normalgw.bacnet.v2.ApplicationDataValue",
        "real": 50
      },
      "scalar": "50"
    },
    {
      "point": {
        "uuid": "ee157a8b-e8e2-38c6-a7d5-c991a2963f93",
        "layer": "hpl:bacnet:1"
      },
      "value": {
        "@type": "type.googleapis.com/normalgw.bacnet.v2.ApplicationDataValue",
        "real": 50
      },
      "scalar": "50"
    },
    {
      "point": {
        "uuid": "06a65dea-f8ef-3ccf-989c-2646b68d4270",
        "layer": "hpl:bacnet:1"
      },
      "value": {
        "@type": "type.googleapis.com/normalgw.bacnet.v2.ApplicationDataValue",
        "real": 50
      },
      "scalar": "50"
    }
  ],
  "errors": []
}"""
<<<<<<< HEAD
import os
import sys
import json
import requests

nfurl = os.getenv("NFURL", "http://localhost:8080")

# we just use this to get some example points
# this requires nf >= 2.1.1 fo rthe structured query interface
res = requests.post(nfurl + "/api/v1/point/query", data=json.dumps({
=======
import sys
sys.path.append("../..")

from helpers import NfClient, print_response
client = NfClient()

# we just use this to get some example points
# this requires nf >= 2.1.1 fo rthe structured query interface
res = client.post("/api/v1/point/query", json={
>>>>>>> add_auth_to_examples
    "structured_query": {
        "and": [
            {
                "field": {
                    "property": "device_id",
<<<<<<< HEAD
                    "numeric": { "min_value": 260001, "max_value": 260001 }
=======
                    "text": "260001"
>>>>>>> add_auth_to_examples
                }
            },
            {
                "field" : {
                    "property": "type", "text": "OBJECT_ANALOG_VALUE",
                },
            },
        ],
    },
    "layer": "hpl:bacnet:1",
    "page_size": "15",
<<<<<<< HEAD
}))
print ("{}: {}".format(res.status_code, res.headers.get("grpc-message")))
uuids = list(map(lambda x: x["uuid"], res.json()["points"]))

res = requests.post(nfurl + "/api/v2/command/write", json={
=======
})
print ("{}: {}".format(res.status_code, res.headers.get("grpc-message")))
uuids = list(map(lambda x: x["uuid"], res.json()["points"]))

res = client.post("/api/v2/command/write", json={
>>>>>>> add_auth_to_examples
    "writes": [ {
        "point": {
            "uuid": u,
            "layer": "hpl:bacnet:1",
        },
        "value": {
            # the value needs to be an ApplicationDataValue.  However,
            # unlike the direct BACnet API, the command service
            # performs type conversion.  Therefore (for instance) even
            # though we are writing to the present-value of an Analog
            # Value in this example (which has type real), you may
            # also send a double or unsigned here,
            "real": "50" if len(sys.argv) < 2 else float(sys.argv[1]),
        },
    } for u in uuids ],
})
        
<<<<<<< HEAD
json.dump(res.json(), sys.stdout, indent=2)
=======
print_response(res)
>>>>>>> add_auth_to_examples
