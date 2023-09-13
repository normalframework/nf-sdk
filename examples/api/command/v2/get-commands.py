"""List read commands"""
import os
import sys
import json
import requests

nfurl = os.getenv("NFURL", "http://localhost:8080")                        

res = requests.get(nfurl + "/api/v2/command")
json.dump(res.json(), sys.stdout, indent=2)
