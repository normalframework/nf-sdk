"""Read a schedule object's Weekly Schedule property

Example output:
{
    "ack": {
        "readProperty": {
            "objectIdentifier": {
                "objectType": "OBJECT_TYPE_SCHEDULE",
                "instance": 1
            },
            "propertyIdentifier": "PROPERTY_IDENTIFIER_WEEKLY_SCHEDULE",
            "propertyArrayIndex": 4294967295,
            "propertyValue": {
                "@type": "type.googleapis.com/normalgw.bacnet.v2.ListOfDailySchedule",
                        ]
                    },
                    {
                        "daySchedule": [
                            {}
                        ]
                    },
                    {
                        "daySchedule": [
                            {}
                        ]
                    },
                    {
                        "daySchedule": [
                            {}
                        ]
                    },
                    {
                        "daySchedule": [
                            {}
                        ]
                    },
                    {
                        "daySchedule": [
                            {}
                        ]
                    },
                    {
                        "daySchedule": [
                            {}
                        ]
                    }
                ]
            }
        }
    }
}
"""
import sys
sys.path.append("../..")

from helpers import NfClient, print_response
client = NfClient()

# device_adr = [192,168,103,178,0xba, 0xc0]

res = client.post("/api/v2/bacnet/confirmed-service", json={
    "device_address": {
        "device_id": 260001,
        # "mac": base64.b64encode(bytes(device_adr)),
    },
    "request": {
        "read_property": {
            "object_identifier": {
                "object_type": "OBJECT_TYPE_SCHEDULE",
                "instance":1,
            },
            "property_identifier": "PROPERTY_IDENTIFIER_WEEKLY_SCHEDULE",
            "property_array_index": 4294967295,
        },

    }
},)
print_response(res)

