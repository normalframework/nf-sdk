"""SubscribeCOV Example

Subscribe to Change-of-Value (COV) notifications for a BACnet object.
This sends a SubscribeCOV confirmed service request to a device, which
instructs the device to send COV notifications when the object's value
changes.  After subscribing, the script listens for incoming
unconfirmed COV notification messages via the gateway's streaming
UnconfirmedHandler endpoint.

To use:
  1. Edit device_id and object below to match your site
  2. Run the script -- it will subscribe and then print notifications
  3. Ctrl-C to stop

Notes:
  - The 'lifetime' field (in seconds) controls how long the subscription
    stays active on the device.  The device will stop sending notifications
    after this time unless the subscription is renewed.
  - Setting issue_confirmed_notifications=False means the device sends
    unconfirmed COV notifications, which is the most common mode.
  - The UnconfirmedHandler stream returns ALL unconfirmed requests
    received by the gateway, so we filter for COV notifications.

Example subscribe response (SimpleAck on success):

{
    "SimpleAck": true
}

Example COV notification received via UnconfirmedHandler:

{
    "deviceAddress": {
        "deviceId": 260001
    },
    "request": {
        "unconfirmedCovNotification": {
            "subscriberProcessIdentifier": 1234,
            "initiatingDeviceIdentifier": {
                "objectType": "OBJECT_TYPE_DEVICE",
                "instance": 260001
            },
            "monitoredObjectIdentifier": {
                "objectType": "OBJECT_TYPE_ANALOG_INPUT",
                "instance": 1
            },
            "timeRemaining": 290,
            "listOfValues": [
                {
                    "propertyIdentifier": "PROPERTY_IDENTIFIER_PRESENT_VALUE",
                    "propertyArrayIndex": 4294967295,
                    "value": {
                        "@type": "type.googleapis.com/normalgw.bacnet.v2.ApplicationDataValue",
                        "real": 72.5
                    }
                },
                {
                    "propertyIdentifier": "PROPERTY_IDENTIFIER_STATUS_FLAGS",
                    ...
                }
            ]
        }
    }
}
"""
import sys
import json
import time
sys.path.append("../..")

from helpers import NfClient, print_response

client = NfClient()

# --- Configuration ---
device_id = 260001
object_type = "OBJECT_TYPE_ANALOG_INPUT"
object_instance = 1
process_id = 1234           # arbitrary identifier for this subscription
lifetime = 300              # subscription lifetime in seconds

# Step 1: Send SubscribeCOV to the device
print(f"Subscribing to COV for {object_type} {object_instance} on device {device_id}...")
res = client.post("/api/v2/bacnet/confirmed-service", json={
    "device_address": {
        "device_id": device_id,
    },
    "request": {
        "subscribe_cov": {
            "subscriber_process_identifier": process_id,
            "monitored_object_identifier": {
                "object_type": object_type,
                "instance": object_instance,
            },
            "issue_confirmed_notifications": False,
            "lifetime": lifetime,
        },
    },
})

if res.status_code != 200:
    print(f"Subscribe failed: {res.status_code}")
    print_response(res)
    sys.exit(1)

result = res.json()
if "SimpleAck" in result:
    print("Subscribed successfully.")
elif "error" in result:
    print(f"Device returned error: {json.dumps(result['error'], indent=2)}")
    sys.exit(1)
else:
    print(f"Unexpected response: {json.dumps(result, indent=2)}")
    sys.exit(1)

# Step 2: Listen for COV notifications via the UnconfirmedHandler stream
print(f"\nListening for COV notifications (lifetime={lifetime}s)...")
print("Press Ctrl-C to stop.\n")

try:
    import requests as req
    # The unconfirmed-requests endpoint is a server-sent event stream
    with req.get(
        client.base + "/api/v2/bacnet/unconfirmed-requests",
        auth=client.auth,
        stream=True,
        timeout=lifetime + 30,
    ) as stream:
        stream.raise_for_status()
        for line in stream.iter_lines(decode_unicode=True):
            if not line:
                continue
            try:
                msg = json.loads(line)
            except json.JSONDecodeError:
                continue

            # Filter for COV notifications
            request = msg.get("request", {})
            if "unconfirmedCovNotification" in request:
                cov = request["unconfirmedCovNotification"]
                obj = cov.get("monitoredObjectIdentifier", {})
                values = cov.get("listOfValues", [])
                print(f"COV from device {msg.get('deviceAddress', {}).get('deviceId')}: "
                      f"{obj.get('objectType')} {obj.get('instance')}")
                for v in values:
                    prop = v.get("propertyIdentifier", "")
                    val = v.get("value", {})
                    print(f"  {prop}: {json.dumps(val)}")
                print()
except KeyboardInterrupt:
    print("\nStopped.")
except Exception as e:
    print(f"Stream error: {e}")
