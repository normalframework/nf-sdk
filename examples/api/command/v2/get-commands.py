"""List read commands"""
import os
import sys
import json
import requests

sys.path.append("../..")
from auth.v1.auth import get_token

nfurl = os.getenv("NFURL", "http://localhost:8080")
headers = {"Authorization": f"Bearer {get_token()}"}

res = requests.get(nfurl + "/api/v2/command", headers=headers)

json.dump(res.json(), sys.stdout, indent=2)
