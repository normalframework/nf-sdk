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

nfurl = os.getenv("NFURL", "http://localhost:8080")
device_adr = [192,168,103,178,0xba, 0xc0]

res = requests.post(nfurl + "/api/v2/bacnet/confirmed-service", json={
    "device_address": {
        "device_id": 260001,
        "mac": base64.b64encode(bytes(device_adr)),
    },
    "request": {
        "read_range": {
            "object_identifier": {
                "object_type": "OBJECT_TYPE_TREND_LOG",
                "instance":1,
            },
            "property_identifier": "PROPERTY_IDENTIFIER_LOG_BUFFER",
            "by_sequence_number": {
                "reference_sequence_number": 9002,
                "count": 5,
            }
        },

    }
},)
print ("{}: {}".format(res.status_code, res.headers.get("grpc-message")))
json.dump(res.json(), sys.stdout, indent=4)

