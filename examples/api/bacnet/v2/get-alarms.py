"""Get Alarms Example

Query alarm notifications from the NF gateway.  NF persists BACnet event
notifications received from devices and provides APIs to query, filter,
and acknowledge them.

This example demonstrates:
  1. Querying persisted alarms with time range and filters
  2. Getting event information directly from a remote device
  3. Acknowledging a gateway alarm

API endpoints used:
  - POST /api/v2/bacnet/alarms              (GetAlarms - query persisted alarms)
  - POST /api/v2/bacnet/device-event-information  (GetDeviceEventInformation - live query)
  - POST /api/v2/bacnet/alarms/acknowledge  (AcknowledgeAlarm)

Prerequisites:
  - At least one device must be configured with event subscriptions
    (see POST /api/v2/bacnet/event-subscriptions to create one)
  - Or: use the gateway's built-in alarm collection if enabled

Example GetAlarms response:

{
    "notifications": [
        {
            "processIdentifier": 1,
            "initiatingDeviceIdentifier": {
                "objectType": "OBJECT_TYPE_DEVICE",
                "instance": 260010
            },
            "eventObjectIdentifier": {
                "objectType": "OBJECT_TYPE_ANALOG_INPUT",
                "instance": 1
            },
            "timeStamp": { ... },
            "notificationClass": 1,
            "priority": 100,
            "eventType": "EVENT_TYPE_OUT_OF_RANGE",
            "messageText": "",
            "notifyType": "NOTIFY_TYPE_ALARM",
            "ackRequired": true,
            "fromState": "EVENT_STATE_NORMAL",
            "toState": "EVENT_STATE_HIGH_LIMIT",
            "receivedAt": "2025-01-15T10:30:00Z",
            "ackState": "UNACKED"
        }
    ],
    "nextPageToken": ""
}
"""
import sys
import json
from datetime import datetime, timedelta, timezone
sys.path.append("../..")

from helpers import NfClient, print_response

client = NfClient()


# --- 1. Query persisted alarms from the gateway ---
print("=" * 60)
print("1. Querying persisted alarms from the gateway")
print("=" * 60)

# Query alarms from the last 24 hours
now = datetime.now(timezone.utc)
one_day_ago = now - timedelta(hours=24)

res = client.post("/api/v2/bacnet/alarms", json={
    "from": one_day_ago.isoformat(),
    "to": now.isoformat(),
    # Optional filters (uncomment as needed):
    # "device_instances": [260010],
    # "event_types": ["EVENT_TYPE_OUT_OF_RANGE"],
    # "notify_types": ["NOTIFY_TYPE_ALARM"],
    # "to_states": ["EVENT_STATE_HIGH_LIMIT", "EVENT_STATE_LOW_LIMIT"],
    # "priority_min": 0,
    # "priority_max": 100,
    "limit": 20,
})

if res.status_code != 200:
    print(f"Error: {res.status_code}")
    print_response(res)
else:
    data = res.json()
    notifications = data.get("notifications", [])
    print(f"Found {len(notifications)} alarm(s)\n")

    for alarm in notifications:
        obj = alarm.get("eventObjectIdentifier", {})
        device = alarm.get("initiatingDeviceIdentifier", {})
        print(f"  Device {device.get('instance')}: "
              f"{obj.get('objectType')} {obj.get('instance')}")
        print(f"    State: {alarm.get('fromState')} -> {alarm.get('toState')}")
        print(f"    Type:  {alarm.get('eventType')}")
        print(f"    Priority: {alarm.get('priority')}  "
              f"Notify: {alarm.get('notifyType')}  "
              f"Ack: {alarm.get('ackState', 'N/A')}")
        print(f"    Received: {alarm.get('receivedAt', '')}")
        print()

    if data.get("nextPageToken"):
        print(f"  (more results available, nextPageToken: {data['nextPageToken']})")


# --- 2. Query event information directly from a device ---
print("\n" + "=" * 60)
print("2. Querying event information from remote device")
print("=" * 60)

device_id = 260010  # edit to match your site

res = client.post("/api/v2/bacnet/device-event-information", json={
    "device_address": {
        "device_id": device_id,
    },
})

if res.status_code != 200:
    print(f"Error: {res.status_code}")
    print_response(res)
else:
    data = res.json()
    summaries = data.get("eventSummaries", [])
    print(f"Device {device_id} reports {len(summaries)} event(s)\n")

    for evt in summaries:
        obj = evt.get("objectIdentifier", {})
        print(f"  {obj.get('objectType')} {obj.get('instance')}")
        print(f"    State: {evt.get('eventState')}  "
              f"Notify: {evt.get('notifyType')}")
        print(f"    Enable: {evt.get('eventEnable', {})}")
        print()


# --- 3. Acknowledge an alarm (uncomment to use) ---
# To acknowledge a persisted alarm, you need its alarm_id from the
# GetAlarms response.  Uncomment and edit the block below.
#
# print("\n" + "=" * 60)
# print("3. Acknowledging alarm")
# print("=" * 60)
#
# alarm_id = "YOUR_ALARM_ID_HERE"
# res = client.post("/api/v2/bacnet/alarms/acknowledge", json={
#     "alarm_id": alarm_id,
#     "acknowledged_by": "operator@example.com",
#     # Also send BACnet AcknowledgeAlarm to the originating device:
#     "send_to_device": True,
# })
# print_response(res)
