"""Create A MultiState Value Object

This example shows how to also create the State_Text property which
defines the values of the enumeration.
"""
import sys
sys.path.append("../..")

from helpers import NfClient, print_response

client = NfClient()

res = client.patch("/api/v1/bacnet/local", {
    "object_id": {
        "object_type": "OBJECT_MULTI_STATE_VALUE",
        "instance": 1, # create a new object
    },
    "props": [
        {
            "property": "PROP_PRESENT_VALUE",
            "value": {
                "enumerated": 1,
            },
        },
    ]
})
print_response(res)
        

