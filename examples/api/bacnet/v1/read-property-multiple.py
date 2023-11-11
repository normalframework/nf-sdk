"""Perform a Read Property Multiple

Pre 2.1, the ReadProperty-Multiple endpoint only accepted GET requests
which means that all arguments must be sent as query params.

This unfortunately means that only reading one item per request is
possible.
"""
import os
import sys
import json
import requests
import base64

nfurl = os.getenv("NFURL", "http://localhost:8080")
device_adr = [192,168,103,178,0xba, 0xc0]

res = requests.get(nfurl + "/api/v1/bacnet/readpropertymultiple", params={
    'device_address.device_id': 260001,
    'device_address.mac': base64.b64encode(bytes(device_adr)),
        # use net and addr if a routed connection
        #'net': 0,
        #'adr': 

    "read_properties.object_id.object_type": "OBJECT_ANALOG_VALUE",
    "read_properties.object_id.instance": 1,
    "read_properties.property_id": "PROP_PRESENT_VALUE",
    "read_properties.array_index": 4294967295,

})

print ("{}: {}".format(res.status_code, res.headers.get("grpc-message")))
json.dump(res.json(), sys.stdout, indent=4)

