"""Device discovery

This example will initiate a device scan which does not scan any found
devices.  It will wait for the scan to complete, and print any
discovered devices.
"""
import time
import sys
sys.path.append("../..")

from helpers import NfClient, print_response

client = NfClient()

res = client.post("/api/v1/bacnet/scan", json={
    "autoImport": False,    # set to true to update object database
    "device": {
        "autoScan": True,       # force a new scan of the device even if we've previously scanned it
        "targets": [
            {
                # set to 0 for global scan
                "lowLimit": 260001,
                "highLimit": 260001,
            },
        ],
    },
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
