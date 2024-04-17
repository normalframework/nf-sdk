"""Perform a Read Property Multiple

Pre 2.1, the ReadProperty-Multiple endpoint only accepted GET requests
which means that all arguments must be sent as query params.

This unfortunately means that only reading one item per request is
possible.
"""
import sys
sys.path.append("../..")

from helpers import NfClient, print_response

client = NfClient()

res = client.get("/api/v1/bacnet/readpropertymultiple", params={
    'device_address.device_id': 260001,
    #'device_address.mac': base64.b64encode(bytes(device_adr)),
    # use net and addr if a routed connection
    #'device_address.net': 0,
    #'device_address.adr': "",

    "read_properties.object_id.object_type": "OBJECT_ANALOG_VALUE",
    "read_properties.object_id.instance": 1,
    "read_properties.property_id": "PROP_PRESENT_VALUE",
    "read_properties.array_index": 4294967295, # BACNET_ARRAY_ALL

})
print_response(res)

