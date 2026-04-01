"""Minimal authenticated HTTP client for NF (env-driven; avoids argparse clashes)."""

from __future__ import annotations

import base64
import os
from typing import Any

import requests


class _JsonOauth(requests.auth.AuthBase):
    def __init__(self, base: str) -> None:
        self.base = base.rstrip("/")
        cid = os.environ.get("NF_CLIENT_ID")
        sec = os.environ.get("NF_CLIENT_SECRET")
        self.oauth = (cid, sec) if cid and sec else None
        user = os.environ.get("NF_BASIC_USER")
        self.creds = user.encode("utf-8") if user else None
        self.token: str | None = None

    def _token(self) -> str:
        assert self.oauth is not None
        r = requests.post(
            f"{self.base}/api/v1/auth/token",
            json={
                "client_id": self.oauth[0],
                "client_secret": self.oauth[1],
                "grant_type": "client_credentials",
            },
            timeout=60,
        )
        r.raise_for_status()
        info = r.json()
        tok = info.get("accessToken") or info.get("access_token")
        if not tok:
            raise RuntimeError("auth token response missing accessToken")
        return str(tok)

    def __call__(self, r: requests.PreparedRequest) -> requests.PreparedRequest:
        if self.oauth is not None:
            if self.token is None:
                self.token = self._token()
            r.headers["Authorization"] = "Bearer " + self.token
        elif self.creds is not None:
            b = base64.b64encode(self.creds).decode("utf-8")
            r.headers["Authorization"] = "Basic " + b
        return r


class NFClient:
    def __init__(self, base_url: str) -> None:
        self.base = base_url.rstrip("/")
        self.auth = _JsonOauth(self.base)

    def get(self, path: str, **kwargs: Any) -> requests.Response:
        url = self.base + path if path.startswith("/") else f"{self.base}/{path}"
        res = requests.get(url, auth=self.auth, timeout=120, **kwargs)
        # OAuth access tokens can expire during long poll intervals; mirror retry intent of nf-sdk helpers.
        if res.status_code == 401 and self.auth.oauth is not None:
            self.auth.token = None
            res = requests.get(url, auth=self.auth, timeout=120, **kwargs)
        return res

    def post(self, path: str, **kwargs: Any) -> requests.Response:
        url = self.base + path if path.startswith("/") else f"{self.base}/{path}"
        res = requests.post(url, auth=self.auth, timeout=120, **kwargs)
        if res.status_code == 401 and self.auth.oauth is not None:
            self.auth.token = None
            res = requests.post(url, auth=self.auth, timeout=120, **kwargs)
        return res
