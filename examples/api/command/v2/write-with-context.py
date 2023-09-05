"""Write BACnet using the command service.

Using the Command service instead of the BACnet service means that
there is no need to manually construct the BACnet requests, or to
split requests up by device.

This example also uses a Command context for writes.  When a command
context is used, any writes made are relinquished automatically at the
end of the context.  The context ends when its lease time expires; or
if manually cancelled.

After running this program, get-commands will return the running
command with writes; after 30 seconds, the writes will be
relinquished.

"""
import os
import sys
import json
import requests

nfurl = os.getenv("NFURL", "http://localhost:8080")

# we just use this to get some example points
# this requires nf >= 2.1.1 fo rthe structured query interface
res = requests.post(nfurl + "/api/v1/point/query", data=json.dumps({
    "structured_query": {
        "and": [
            {
                "field": {
                    "property": "device_id",
                    "numeric": { "min_value": 260001, "max_value": 260001 }
                }
            },
            {
                "field" : {
                    "property": "type", "text": "OBJECT_ANALOG_VALUE",
                },
            },
        ],
    },
    "layer": "hpl:bacnet:1",
    "page_size": "15",
}))
print ("{}: {}".format(res.status_code, res.headers.get("grpc-message")))
uuids = list(map(lambda x: x["uuid"], res.json()["points"]))

# get a command context id for this command. if the name is in use,
# this will return an error instead of allowing you to create two
# conflicting command contexts.
cmd = requests.post(nfurl + "/api/v2/command", json={
    "name": "test context",
    "duration": "30s",
})
command_context_id = (cmd.json())["id"]

res = requests.post(nfurl + "/api/v2/command/write", json={
    "command_id": command_context_id,
    "writes": [ {
        "point": {
            "uuid": u,
            "layer": "hpl:bacnet:1",
        },
        "value": {
            # the value needs to be an ApplicationDataValue.  However,
            # unlike the direct BACnet API, the command service
            # performs type conversion.  Therefore (for instance) even
            # though we are writing to the present-value of an Analog
            # Value in this example (which has type real), you may
            # also send a double or unsigned here,
            "real": "50" if len(sys.argv) < 2 else float(sys.argv[1]),
        },
    } for u in uuids ],
})
        
json.dump(res.json(), sys.stdout, indent=2)
