"""ReadProperty-Multiple example of reading three properties from a single object.
"""


import sys
import os
import base64
import requests
import json

nfurl = os.getenv("NFURL", "http://8.15.101.121:8080")
device_adr = [8, 27,220, 10, 0xba, 0xc1]

session = requests.Session()
session.auth = ("admin", "jiECJU8kvLhd7i4VhvDiy4")
res = session.post(nfurl + "/api/v2/bacnet/confirmed-service", json={
    "device_address": {
     "mac": "CBvcD7rB",
     "net": 49998,
     "adr": "wKgBebrA",
     "maxApdu": 1476,
     "deviceId": 4159533,
     "bbmd": ""
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
print(res.headers)
print ("{}: {}".format(res.status_code, res.headers.get("grpc-message")))
print (res.content)
json.dump(res.json(), sys.stdout, indent=2)

