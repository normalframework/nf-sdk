"""Create A MultiState Value Object

This example shows how to also create the State_Text property which
defines the values of the enumeration.
"""
import sys
sys.path.append("../..")

from helpers import NfClient, print_response

client = NfClient()

res = client.post("/api/v1/bacnet/local", {
    "uuid": "8735d5ca-d076-11f0-8dbc-1e262a866288",
    "object_id": {
        "object_type": "OBJECT_MULTI_STATE_VALUE",
        "instance":0, # create a new object
    },
    "props": [
        {
            "property": "PROP_OBJECT_NAME",
            "value": {
                "character_string": "Example MSV",
            },
        },
        {
            "property": "PROP_UNITS",
            "value": {
                "enumeration": "85", # look up enum value of your units
            },
        },
        {
            "property": "PROP_OUT_OF_SERVICE",
            "value": {
                "boolean": False,
            },
        },
        {
            "property": "PROP_PRESENT_VALUE",
            "value": {
                "enumerated": 2,
            },
        },
        {
            "property": "PROP_NUMBER_OF_STATES",
            "value": {
                "unsigned": 3,
            },
        },
        {
            "property": "PROP_STATE_TEXT",
            "value": {
                "array": [
                    {
                        "character_string": "State 1",
                    },
                    {
                        "character_string": "State 2",
                    },
                    {
                        "character_string": "State 3",
                    },
                ]
            },
        }
    ]
})
print_response(res)
        

