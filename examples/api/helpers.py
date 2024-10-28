"""Example API wrapper

This file implements helpers
"""
import sys
import json
import os
import requests
import argparse
import base64

class JsonOauth(requests.auth.AuthBase):
    def __init__(self, base, oauth=None, creds=None):
        self.token = None
        self.base = base
        self.oauth = oauth
        self.creds = creds

    def handle_401(self, r, **kwargs):
        """
        If auth is configured, we may need to acquire a token and
        retry the request.  This might not work.
        """
        r.content
        r.close()


        res = requests.post(self.base + "/api/v1/auth/token", json={
            "client_id": self.oauth[0],
            "client_secret": self.oauth[1],
            "grant_type": "client_credentials"
        })

        if res.status_code != 200:
            raise Exception("Invalid authentication: ")

        info = res.json()
        self.token = info.get("accessToken", info.get("access_token"))

        prep = r.request.copy()
        prep.headers["Authorization"] = "Bearer " + self.token
        _r = r.connection.send(prep, **kwargs)
        _r.history.append(r)
        _r.request = prep
        return _r

    def __call__(self, r):
        if self.oauth is not None:
            # if using token auth, we need to request an access token
            if self.token is None:
                r.register_hook("response", self.handle_401)
            else:
                r.headers["Authorization"] = "Bearer " + self.token
        elif self.creds is not None:
            # if using basic, just send the credentials
            r.headers["Authorization"] = "Basic " + base64.b64encode(self.creds.encode("utf-8")).decode("utf-8")

        return r

class NfClient(object):
    """A simple wrapper for requests wihch adds authentication tokens"""

    def __init__(self):
        # pull the connection URL and so on from the environment
        parser = argparse.ArgumentParser("nf-client")
        parser.add_argument("--client-id", default=os.getenv("NF_CLIENT_ID"))
        parser.add_argument("--client-secret", default=os.getenv("NF_CLIENT_SECRET"))
        parser.add_argument("-u", "--user")
        parser.add_argument("--url", default=os.getenv("NFURL", "http://localhost:8080"))
        args = parser.parse_args()


        self.base = args.url.rstrip("/")
        self.auth = JsonOauth(self.base, oauth=(args.client_id, args.client_secret) if
                              args.client_id and args.client_secret else None,
                              creds=args.user)
    
    def get(self, path, *args, **kwargs):
        return requests.get(self.base + path, *args, auth=self.auth, **kwargs)

    def post(self, path, json={}, *args, **kwargs):
        return requests.post(self.base + path, auth=self.auth, json=json)
    

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
