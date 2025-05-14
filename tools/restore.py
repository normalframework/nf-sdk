"""Restor a backup"""

import json
import zipfile
import os
import sys
sys.path.append(os.path.join(os.path.dirname(__file__), "..", "examples", "api"))

from helpers import NfClient, print_response

def load_points(client, ar, create_layers=False):
    if create_layers:
        lfp = ar.open("layers.jsonl")
        for l in lfp.readlines():
            res = json.loads(l)
            for l in res["layers"]:
                print("Creating layer", l["name"])
                res = client.post("/api/v1/point/layers", json={
                    "layer": l})
                print_response(res)
        lfp.close()

    fp = ar.open("points.jsonl")
    points = []
    i = 0
    for l in fp.readlines():
        i += 1
        rec = json.loads(l)
        points.append(rec)
        if len(points) == 500:
            res = client.post("/api/v1/point/points", json={
                "points": points,
                "is_async": True,
            })
            print_response(res)
            points = []
    if len(points) > 0:
        res = client.post("/api/v1/point/points", json={
            "points": points,
            "is_async": True,
        })
        print_response(res)

if __name__ == '__main__':
    client = NfClient()

    if len(sys.argv) < 2:
        print("usage:\n\trestore.py <backup.zip>\n")
        sys.exit(1)
    ar = zipfile.ZipFile(sys.argv[1])
    print(ar.namelist())
    load_points(client, ar)
