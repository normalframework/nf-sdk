"""Read and write a daily schedule object.

NB: this may not work on many controllers, and successfully writing
the schedule requires understanding what valid values are possible for
each schedule event.
"""
import sys
sys.path.append("../..")

from helpers import NfClient, print_response
client = NfClient()

#device_adr = [192,168,103,178,0xba, 0xc0]

read_request = {
  "deviceAddress": {
      "deviceId": 260001
      # "mac": base64.b64encode(bytes(device_adr)),
  },
  "request": {
    "readProperty": {
      "objectIdentifier": {
        "objectType": "OBJECT_TYPE_SCHEDULE",
        "instance": 1
      },
      "propertyIdentifier": "PROPERTY_IDENTIFIER_WEEKLY_SCHEDULE",
      "propertyArrayIndex": 4294967295
    }
  }
}
res = client.post("/api/v2/bacnet/confirmed-service", json=read_request)
print_response(res)


# writeproperty
write_request = {
  "deviceAddress": {
      # base64 IP + port for BACnet/ip
      # "mac": base64.b64encode(bytes(device_adr)),
      "deviceId": 260001,
  },
  "request": {
    "writeProperty": {
      "objectIdentifier": {
        "objectType": "OBJECT_TYPE_SCHEDULE",
        "instance": 1
      },
      "propertyIdentifier": "PROPERTY_IDENTIFIER_WEEKLY_SCHEDULE",
      "propertyArrayIndex": 2,
      "propertyValue": {
        "@type": "type.googleapis.com/normalgw.bacnet.v2.ListOfDailySchedule",
        "listOfDailySchedule": [
          {
            "daySchedule": [
              {
                "time": {
                  "hour": 6
                },
                "value": {
                  "@type": "type.googleapis.com/normalgw.bacnet.v2.ApplicationDataValue",
                  "enumerated": 2
                }
              }
            ]
          }
        ]
      },
      "priority": 16
    }
  }
}

res = client.post("/api/v2/bacnet/confirmed-service", json=write_request)
print_response(res)
