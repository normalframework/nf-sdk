"""WriteProperty using an ApplicationDataValue type
"""
import sys
import os
import base64
import requests
import json

nfurl = os.getenv("NFURL", "http://localhost:8080")
device_adr = [192,168,103,178,0xba, 0xc0]

res = requests.post(nfurl + "/api/v2/bacnet/confirmed-service", json={
    "device_address": {
        "device_id": 260001,
        "mac": base64.b64encode(bytes(device_adr)),
    },
    "request": {
        "write_property": {
            "object_identifier": {
                "object_type": "OBJECT_TYPE_ANALOG_OUTPUT",
                "instance":1,
            },
            "property_identifier": "PROPERTY_IDENTIFIER_PRESENT_VALUE",
            "property_array_index": 4294967295,
            "property_value": {
                "@type": "type.googleapis.com/normalgw.bacnet.v2.ApplicationDataValue",
                "real": 1
            },
            "priority": 16,
        },
    }
},)
print ("{}: {}".format(res.status_code, res.headers.get("grpc-message")))
json.dump(res.json(), sys.stdout, indent=4)

