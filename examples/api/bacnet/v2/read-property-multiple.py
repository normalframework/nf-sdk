"""ReadProperty-Multiple example of reading three properties from a single object.
"""
import sys
sys.path.append("../..")

from helpers import NfClient, print_response
client = NfClient()

# device_adr = [8, 27,220, 10, 0xba, 0xc1]

res = client.post("/api/v2/bacnet/confirmed-service", json={
    "device_address": {
        "deviceId": 260001,
    },
    "request": {
        "read_property_multiple": {
            "list_of_read_access_specifications": [
                {
                    "object_identifier": {
                        "object_type": "OBJECT_TYPE_TREND_LOG",
                        "instance":1,
                    },
                    "list_of_property_references": [
                        {
                            "property_identifier": "PROPERTY_IDENTIFIER_UNITS",
                            "property_array_index": 4294967295,
                        },
                        {
                            "property_identifier": "PROPERTY_IDENTIFIER_TOTAL_RECORD_COUNT",
                            "property_array_index": 4294967295,
                        },
                        {
                            "property_identifier": "PROPERTY_IDENTIFIER_BUFFER_SIZE",
                            "property_array_index": 4294967295,
                        }
                    ],
                }
            ]
        },
    }
},)
print_response(res)

