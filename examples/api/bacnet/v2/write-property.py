"""WriteProperty using an ApplicationDataValue type
"""
import sys
sys.path.append("../..")

from helpers import NfClient, print_response
client = NfClient()

#device_adr = [192,168,103,178,0xba, 0xc0]

res = client.post("/api/v2/bacnet/confirmed-service", json={
    "device_address": {
        "device_id": 260001,
        # "mac": base64.b64encode(bytes(device_adr)),
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
print_response(res)
