"""Create A MultiState Value Object

This example shows how to also create the State_Text property which
defines the values of the enumeration.
"""
import sys
sys.path.append("../..")

from helpers import NfClient, print_response

client = NfClient()

res = client.patch("/api/v1/bacnet/local", {
    #"localDeviceInstanceOffset": 30184,
    # "uuid": "f17a5270-70a1-502b-bb6c-a16c2126c839",
    "object_id": {
        "object_type": "OBJECT_MULTI_STATE_INPUT",
        "instance": 1, # create a new object
    },    
    "props": [
        {
            "property": "PROP_PRESENT_VALUE",
            "value": {
                "enumerated": 3,
            },
        },
        {
            "property": "PROP_OUT_OF_SERVICE",
            "value": {
                "boolean": True,
            }
        },
        {
            "property": "PROP_STATUS_FLAGS",
            "value": {
                "bitString": {
                    "length": 4,
                    "setBits": [
                        1, 2
                    ]
                }
            }
        }
    ]
})
print_response(res)
        

