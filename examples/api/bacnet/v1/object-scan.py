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
   "parentId": 1244,
   "object": {
    "target": {
     "mac": "CmUoGbrA",
     "net": 30000,
     "adr": "Ag==",
     "maxApdu": 480,
     "deviceId": 300002,
     "bbmd": "",
     "portId": 0
    },
    "properties": ["PROP_OBJECT_NAME", "PROP_UNITS"],
    "objectTypes": [
     "OBJECT_BINARY_VALUE",
     "OBJECT_DEVICE"
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
