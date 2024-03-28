"""Get Auth Token"""
import os
import sys
import json
import requests

nfurl = os.getenv("NFURL", "http://localhost:8080")
client_id = os.getenv("NF_CLIENT_ID")
client_secret = os.getenv("NF_CLIENT_SECRET")

def get_token():
    url = nfurl + "/api/v1/auth/token"

    data = json.dumps(
        {
            "client_id": client_id,
            "client_secret": client_secret,
            "grant_type": "client_credentials",
        }
    )

    response = requests.request("POST", url=url, data=data)
    return response.json()["accessToken"]
