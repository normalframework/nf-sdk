"""Device discovery

This example will initiate a device scan which does not scan any found
devices.  It will wait for the scan to complete, and print any
discovered devices.
"""
import time
import sys
import base64
sys.path.append("../..")

from helpers import NfClient, print_response

client = NfClient()

device_adr = [10, 0, 1, 5, 0xba, 0xc0]

res = client.post("/api/v1/bacnet/scan", json={
   "parentId": 1,
   "object": {
    "target": {
     "mac": base64.b64encode(bytes(device_adr)).decode("utf-8"),
     "maxApdu": 1476,
     "deviceId": 9001,
    },
    "properties": [],
    "objectTypes": [
     "OBJECT_ANALOG_INPUT",
     "OBJECT_ANALOG_OUTPUT",
     "OBJECT_ANALOG_VALUE",
     "OBJECT_BINARY_INPUT",
     "OBJECT_BINARY_OUTPUT",
     "OBJECT_BINARY_VALUE",
     "OBJECT_DEVICE",
     "OBJECT_MULTI_STATE_INPUT",
     "OBJECT_MULTI_STATE_OUTPUT",
     "OBJECT_MULTI_STATE_VALUE",
     "OBJECT_SCHEDULE",
     "OBJECT_CALENDAR",
     "OBJECT_NETWORK_PORT"
    ],
    "ifMissing": "DELETE"
   },
   "autoImport": True,
  })
              
print_response(res)
scan_id = res.json()["id"]

print("waiting for device scan to complete...")
def wait_on_job(scan_id):
    for _ in range(0, 100):
        time.sleep(.1)
        res = client.get("/api/v1/bacnet/scan", params={
            "idFilter": scan_id,
            "full": True,
        })
        scan = res.json()
        if scan["results"][0]["status"] not in ["PENDING", "RUNNING"]:
            print_response(res)
            break

wait_on_job(scan_id)

# now query on parentIdFilter=scan_id to get the object lists from any discovered devices
