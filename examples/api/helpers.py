"""Example API wrapper

This file implements helpers
"""
import sys
import json
import os
import requests

base = os.getenv("NFURL", "http://localhost:8080")
client_id = os.getenv("NF_CLIENT_ID")
client_secret = os.getenv("NF_CLIENT_SECRET")

class JsonOauth(requests.auth.AuthBase):
    def __init__(self):
        self.token = None

    def handle_401(self, r, **kwargs):
        """
        If auth is configured, we may need to acquire a token and
        retry the request.  This might not work.
        """
        r.content
        r.close()

        res = requests.post(base + "/api/v1/auth/token", json={
            "client_id": client_id,
            "client_secret": client_secret,
            "grant_type": "client_credentials"
        })

        if res.status_code != 200:
            raise Exception("Invalid authentication: ")

        info = res.json()
        self.token = info["accessToken"]
        print (self.token)

        prep = r.request.copy()
        prep.headers["Authorization"] = "Bearer " + self.token
        _r = r.connection.send(prep, **kwargs)
        _r.history.append(r)
        _r.request = prep
        return _r

    def __call__(self, r):
        if self.token is None:
            r.register_hook("response", self.handle_401)
        else:
            r.headers["Authorization"] = "Bearer " + self.token
        return r

class NfClient(object):
    """A simple wrapper for requests wihch adds authentication tokens"""

    def __init__(self):
        self.auth = JsonOauth()
    
    def get(self, path, *args, **kwargs):
        return requests.get(base + path, auth=self.auth)

    def post(self, path, json={}, *args, **kwargs):
        return requests.post(base + path, auth=self.auth, json=json)
    

def print_response(res):
    """Interpret the response from requests
    """
    if res.status_code == 200:
        json.dump(res.json(), sys.stdout, indent=2)
        print()#newline
    else:
        if "x-nf-unauthorized-reason" in res.headers:
            sys.stderr.write(res.headers.get("x-nf-unauthorized-reason") + "\n")
            sys.exit(1)
        else:
            # the headers should have more information for other error
            print (res.headers)
