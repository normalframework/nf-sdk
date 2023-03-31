"""Perform a Read Property Multiple

This is only supported in >= 2.1
"""
import os
import sys
import json
import requests
import base64

nfurl = os.getenv("NFURL", "http://localhost:8080")
device_adr = [192,168,103,178,0xba, 0xc0]

res = requests.post(nfurl + "/api/v1/bacnet/readpropertymultiple", json={
    'device_address':{
        'device_id': 260001,
        'mac': base64.b64encode(bytes(device_adr)),
        # use net and addr if a routed connection
        #'net': 0,
        #'adr': 
    },

    'read_properties': [
        {
            'object_id': {
                'object_type': "OBJECT_ANALOG_OUTPUT",
                'instance': 1,
            },
            'property_id': 'PROP_PRESENT_VALUE',
            'array_index': 4294967295,
        },
        {
            'object_id': {
                'object_type': "OBJECT_ANALOG_OUTPUT",
                'instance': 1,
            },
            'property_id': 'PROP_PRIORITY_ARRAY',
            'array_index': 4294967295,
        }
    ]
})

print ("{}: {}".format(res.status_code, res.headers.get("grpc-message")))
json.dump(res.json(), sys.stdout, indent=4)

