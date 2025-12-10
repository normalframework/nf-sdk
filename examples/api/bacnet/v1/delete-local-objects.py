
"""Create A MultiState Value Object

This example shows how to also create the State_Text property which
defines the values of the enumeration.
"""
import sys
sys.path.append("../..")
import json

from helpers import NfClient, print_response

client = NfClient()

db = client.get("/api/v1/bacnet/local").json()

if input("really delete {} objects? [y/N] ".format(len(db["objects"]))).strip() != "y":
    sys.exit(1)

for p in db["objects"]:
    if p["objectId"]["objectType"] != "OBJECT_ANALOG_INPUT":
        continue
    print (p)

    res = client.delete("/api/v1/bacnet/local/" + p["objectId"]["objectType"] + "/" + str(p["objectId"]["instance"]))
    print(res)



