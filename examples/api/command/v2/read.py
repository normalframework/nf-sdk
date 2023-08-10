"""Read BACnet using the command service.

Using the Command service instead of the BACnet service means that
there is no need to manually construct the BACnet requests, or to
split requests up by device.

Example results:
{
  "results": [
    {
      "point": {
        "uuid": "e88ad19c-718b-34ac-9911-f8186c0790aa",
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
        "uuid": "163fc6f5-caeb-3cc8-8889-db41007094b9",
        "layer": "hpl:bacnet:1"
      },
      "value": {
    },
    {
      "point": {
        "uuid": "741362e5-f54a-3841-bc54-8cda81267479",
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
        "uuid": "a7295987-af48-3998-a06d-4306989dbaff",
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
        "uuid": "0837a003-dd13-37c5-b8f8-96b0344d4d34",
        "layer": "hpl:bacnet:1"
      },
      "value": {
        "@type": "type.googleapis.com/normalgw.bacnet.v2.ApplicationDataValue",
        "unsigned": 1
      },
      "scalar": "1"
    }
  ],
  "errors": []
}

"""
import os
import sys
import json
import requests

nfurl = os.getenv("NFURL", "http://localhost:8080")

# we just use this to get some example points
# this requires nf >= 2.1.1
res = requests.post(nfurl + "/api/v1/point/query", data=json.dumps({
    "structured_query": {
        "and": [
            {
                "field": {
                    "property": "device_id",
                    "numeric": { "min_value": 260001, "max_value": 260001 }
                }
            },
            {
                "or": [
                    {
                        "field": {
                            "property": "type", "text": "OBJECT_ANALOG_INPUT",
                        }
                    }, {
                        "field" : {
                            "property": "type", "text": "OBJECT_ANALOG_VALUE",
                        },
                    },
                ],
            },
        ],
    },
    "layer": "hpl:bacnet:1",
    "page_size": "25",
}))
print ("{}: {}".format(res.status_code, res.headers.get("grpc-message")))
uuids = list(map(lambda x: x["uuid"], res.json()["points"]))

res = requests.post(nfurl + "/api/v2/command/read", json={
    "reads": [ {
        "point": {
            "uuid": u,
            "layer": "hpl:bacnet:1",
        },

        # This is optional but you can use this to read a different BACnet property on the same object.  This will generate an error if the underlying point is not really BACnet.
        
        #"bacnet_options": {
        #    "property_identifier": "PROPERTY_IDENTIFIER_UNITS",
        #}
    } for u in uuids ],
})
        
json.dump(res.json(), sys.stdout, indent=2)
