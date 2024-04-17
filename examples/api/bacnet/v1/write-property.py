"""Perform a BACnet write-property request

Note array_index=4294967295 is used for "BACNET_ARRAY_ALL"
"""

import base64
import sys
sys.path.append("../..")

from helpers import NfClient, print_response

client = NfClient()

# device_adr = [192,168,103,178,0xba, 0xc0]

res = client.post("/api/v1/bacnet/writeproperty", json={
    'device_address': {
        'device_id': 260001,
        # 'mac': base64.b64encode(bytes(device_adr)),
        # use net and addr if a routed connection
        #'net': 0,
        #'adr': 
    },
    'property':{
        'object_id': {
            'object_type': 'OBJECT_BINARY_VALUE',
            'instance': 1,
        },
        'property_id': 'PROP_PRESENT_VALUE',
        'array_index':4294967295,
    },
    'value':{
        'enumerated': 1,
    },
    'priority':12,
})
print_response(res)
