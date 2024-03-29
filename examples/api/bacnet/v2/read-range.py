"""ReadRange Example

Read a trend log.  Note for this example you may need to edit or remove
FirstSequenceNumber in order to start reading the beginning of the log

{
    "ack": {
        "readRange": {
            "objectIdentifier": {
                "objectType": "OBJECT_TYPE_TREND_LOG",
                "instance": 1
            },
            "propertyIdentifier": "PROPERTY_IDENTIFIER_LOG_BUFFER",
            "propertyArrayIndex": 0,
            "resultFlags": {
                "length": 3,
                "setBits": []
            },
            "itemCount": 5,
            "itemData": {
                "@type": "type.googleapis.com/normalgw.bacnet.v2.ListOfLogRecord",
                "listOfLogRecord": [
                    {
                        "timestamp": {
                            "date": {
                                "year": 109,
                                "month": 2,
                                "day": 1,
                                "wday": 7
                            },
                            "time": {
                                "hour": 0,
                                "minute": 15,
                                "second": 0,
                                "hundreth": 0
                            }
                        },
                        "logDatum": {
                            "realValue": 1001
                        }
                    },
                    {
                        "timestamp": {
                            "date": {
                                "year": 109,
                                "month": 2,
                                "day": 1,
                                "wday": 7
                            },
                            "time": {
                                "hour": 0,
                                "minute": 30,
                                "second": 0,
                                "hundreth": 0
                            }
                        },
                        "logDatum": {
                            "realValue": 1002
                        }
                    },
                    {
                        "timestamp": {
                            "date": {
                                "year": 109,
                                "month": 2,
                                "day": 1,
                                "wday": 7
                            },
                            "time": {
                                "hour": 0,
                                "minute": 45,
                                "second": 0,
                                "hundreth": 0
                            }
                        },
                        "logDatum": {
                            "realValue": 1003
                        }
                    },
                    {
                        "timestamp": {
                            "date": {
                            },
                            "time": {
                                "hour": 1,
                                "minute": 0,
                                "second": 0,
                                "hundreth": 0
                            }
                        },
                        "logDatum": {
                            "realValue": 1004
                        }
                    },
                    {
                        "timestamp": {
                            "date": {
                                "year": 109,
                                "month": 2,
                                "day": 1,
                                "wday": 7
                            },
                            "time": {
                                "hour": 1,
                                "minute": 15,
                                "second": 0,
                                "hundreth": 0
                            }
                        },
                        "logDatum": {
                            "realValue": 1005
                        }
                    }
                ]
            },
            "firstSequenceNumber": 9002
        }
    }
}

"""
import sys
import os
import base64
import requests
import json

nfurl = os.getenv("NFURL", "http://8.15.101.124:8080")
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
        "read_range": {
            "object_identifier": {
                "object_type": "OBJECT_TYPE_TREND_LOG",
                "instance":1,
            },
            "property_identifier": "PROPERTY_IDENTIFIER_LOG_BUFFER",
            "property_array_index": 4294967295,
            "by_position": {
                "reference_index": 1,
                "count": 5,
            }
        }
    }
},)
print ("{}: {}".format(res.status_code, res.headers.get("grpc-message")))
json.dump(res.json(), sys.stdout, indent=4)

