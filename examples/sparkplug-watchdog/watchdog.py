#!/usr/bin/env python3
"""
Sparkplug DDATA Watchdog

Monitors the sparkplug service log stream for DDATA publish messages.
If no DDATA is observed for 5 minutes, restarts the sparkplug service
by writing a noop config change via the platform service.

All sparkplug log messages are written to a log file for diagnostics.

Uses the buf connectrpc library with gRPC-Web transport so it works
through Cloudflare tunnels. Generated types come from the buf BSR
package: buf.build/normalframework/nf

Required API scopes:
  - normalgw.platform.v1.readonly   (StreamLogs, GetEnvironmentVariables)
  - normalgw.platform.v1.readwrite  (SetEnvironmentVariables — not needed in test mode)

Environment variables:
  NFURL             Base URL of the NF site (e.g. https://site.example.com)
  NF_CLIENT_ID      OAuth client ID
  NF_CLIENT_SECRET  OAuth client secret
  DDATA_TIMEOUT     Seconds without DDATA before restart (default: 300)
  RESTART_COOLDOWN  Minimum seconds between restarts (default: 600)
  LOG_FILE          Path for sparkplug log output (default: sparkplug.log)

Install:
  pip install -r requirements.txt --extra-index-url https://buf.build/gen/python

Usage:
  python watchdog.py              # live mode — will restart on timeout
  python watchdog.py --test       # test mode — log only, no restarts
"""

import argparse
import base64
import json
import logging
import os
import sys
import time
from datetime import timezone

import httpx

from connectrpc.errors import ConnectError
from connectrpc.protocol import ProtocolType
from normalgw.platform.v1 import config_pb2, logs_pb2
from normalgw.platform.v1.config_connect import PlatformConfigClientSync
from normalgw.platform.v1.logs_connect import LogServiceClientSync

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(message)s",
)
log = logging.getLogger("sparkplug-watchdog")

DDATA_TIMEOUT = int(os.getenv("DDATA_TIMEOUT", "300"))
RESTART_COOLDOWN = int(os.getenv("RESTART_COOLDOWN", "600"))
LOG_FILE = os.getenv("LOG_FILE", "sparkplug.log")


# ---------------------------------------------------------------------------
# Token management — cache and refresh based on JWT exp claim
# ---------------------------------------------------------------------------

class TokenManager:
    """Acquires and caches an OAuth token, refreshing only when near expiry."""

    EXPIRY_MARGIN = 60  # refresh 60s before actual expiry

    def __init__(self, base_url: str, client_id: str, client_secret: str):
        self.base_url = base_url.rstrip("/")
        self.client_id = client_id
        self.client_secret = client_secret
        self._token: str | None = None
        self._expires_at: float = 0.0  # monotonic

    @staticmethod
    def _decode_jwt_exp(token: str) -> float | None:
        """Extract the 'exp' claim from a JWT without verifying the signature."""
        parts = token.split(".")
        if len(parts) != 3:
            return None
        payload_b64 = parts[1] + "=" * (-len(parts[1]) % 4)
        try:
            payload = json.loads(base64.urlsafe_b64decode(payload_b64))
            return float(payload["exp"])
        except (json.JSONDecodeError, KeyError, ValueError):
            return None

    def _token_valid(self) -> bool:
        return self._token is not None and time.monotonic() < self._expires_at

    def get_token(self) -> str:
        """Return a valid token, refreshing if expired or close to expiry."""
        if self._token_valid():
            return self._token

        log.info("Acquiring new auth token")
        resp = httpx.post(
            f"{self.base_url}/api/v1/auth/token",
            json={
                "grant_type": "client_credentials",
                "client_id": self.client_id,
                "client_secret": self.client_secret,
            },
            timeout=15,
        )
        if resp.status_code != 200:
            raise RuntimeError(f"Auth failed: HTTP {resp.status_code}: {resp.text}")

        data = resp.json()
        self._token = data.get("access_token") or data.get("accessToken")

        exp_unix = self._decode_jwt_exp(self._token)
        if exp_unix is not None:
            ttl = exp_unix - time.time()
            self._expires_at = time.monotonic() + ttl - self.EXPIRY_MARGIN
            log.info(
                "Token expires in %.0fs, will refresh in %.0fs",
                ttl,
                max(ttl - self.EXPIRY_MARGIN, 0),
            )
        else:
            self._expires_at = time.monotonic() + 300
            log.warning("Could not parse token expiry, will refresh in 300s")

        return self._token

    def auth_headers(self) -> dict[str, str]:
        """Return an Authorization header dict with a valid bearer token."""
        return {"Authorization": f"Bearer {self.get_token()}"}


# ---------------------------------------------------------------------------
# Platform helpers
# ---------------------------------------------------------------------------

def get_env_variable(
    platform: PlatformConfigClientSync,
    token_mgr: TokenManager,
    section: str,
) -> config_pb2.EnvironmentVariable:
    """Fetch the first environment variable in the given section."""
    reply = platform.get_environment_variables(
        config_pb2.GetEnvironmentVariablesRequest(section=section),
        headers=token_mgr.auth_headers(),
    )
    if not reply.variables:
        raise RuntimeError(f"No environment variables found in section '{section}'")
    return reply.variables[0]


def noop_write_env(
    platform: PlatformConfigClientSync,
    token_mgr: TokenManager,
    var: config_pb2.EnvironmentVariable,
):
    """Write an environment variable back with its current value (noop).

    This triggers the platform service to restart the services associated
    with the variable's section.
    """
    write_var = config_pb2.EnvironmentVariable(id=var.id)
    value_field = var.WhichOneof("ValueType")
    if value_field:
        setattr(write_var, value_field, getattr(var, value_field))
    else:
        write_var.string = ""

    reply = platform.set_environment_variables(
        config_pb2.SetEnvironmentVariablesRequest(variables=[write_var]),
        headers=token_mgr.auth_headers(),
    )
    if reply.errors:
        errors = [(e.id, e.error) for e in reply.errors]
        raise RuntimeError(f"SetEnvironmentVariables errors: {errors}")


def restart_services(
    platform: PlatformConfigClientSync,
    token_mgr: TokenManager,
    test_mode: bool = False,
):
    """Restart sparkplug service via a noop env write."""
    section = "Sparkplug"
    try:
        var = get_env_variable(platform, token_mgr, section)
        if test_mode:
            log.info(
                "TEST MODE: would restart %s (id=%s) — skipping", section, var.id
            )
            return
        log.info("Noop write: section=%s id=%s", section, var.id)
        noop_write_env(platform, token_mgr, var)
        log.info("Restart triggered for section: %s", section)
    except Exception:
        log.exception("Failed to restart section %s", section)


# ---------------------------------------------------------------------------
# Log stream watcher
# ---------------------------------------------------------------------------

def is_ddata_message(log_msg: logs_pb2.LogMessage) -> bool:
    """Check if a log message indicates a DDATA publish."""
    return "DDATA" in log_msg.message or "DDATA" in log_msg.metadata


def _format_log_entry(entry: logs_pb2.LogMessage) -> str:
    """Format a LogMessage as a single line for the log file."""
    ts = entry.timestamp.ToDatetime(tzinfo=timezone.utc).isoformat()
    level = logs_pb2.LogLevel.Name(entry.level)
    line = f"{ts} [{level}] {entry.service}: {entry.message}"
    if entry.metadata:
        line += f"  {entry.metadata}"
    return line


def watch_logs(
    log_svc: LogServiceClientSync,
    platform: PlatformConfigClientSync,
    token_mgr: TokenManager,
    test_mode: bool = False,
):
    """Subscribe to sparkplug service logs and watch for DDATA messages.

    Returns when the stream ends (caller should reconnect).
    Triggers a restart if no DDATA is seen within DDATA_TIMEOUT seconds.
    All sparkplug log messages are appended to LOG_FILE.
    """
    last_ddata = time.monotonic()
    last_restart = 0.0

    req = logs_pb2.StreamLogsRequest(
        service_filter=["sparkplug"],
        min_level=logs_pb2.DEBUG,
    )

    mode_label = "TEST MODE" if test_mode else "LIVE"
    log.info(
        "Subscribing to sparkplug logs [%s] (DDATA timeout: %ds, cooldown: %ds, log file: %s)",
        mode_label,
        DDATA_TIMEOUT,
        RESTART_COOLDOWN,
        LOG_FILE,
    )

    with open(LOG_FILE, "a", buffering=1) as logf:
        try:
            for resp in log_svc.stream_logs(req, headers=token_mgr.auth_headers()):
                for entry in resp.logs:
                    logf.write(_format_log_entry(entry) + "\n")

                    if is_ddata_message(entry):
                        elapsed = time.monotonic() - last_ddata
                        last_ddata = time.monotonic()
                        log.info("DDATA seen (%.1fs since last)", elapsed)

                # Check timeout
                gap = time.monotonic() - last_ddata
                if gap > DDATA_TIMEOUT:
                    since_restart = time.monotonic() - last_restart
                    if since_restart > RESTART_COOLDOWN:
                        log.warning(
                            "No DDATA for %.0fs (threshold %ds) — restarting services",
                            gap,
                            DDATA_TIMEOUT,
                        )
                        restart_services(platform, token_mgr, test_mode=test_mode)
                        last_restart = time.monotonic()
                        last_ddata = time.monotonic()
                    else:
                        log.info(
                            "No DDATA for %.0fs but cooldown active (%.0fs remaining)",
                            gap,
                            RESTART_COOLDOWN - since_restart,
                        )
        except ConnectError as e:
            if "missing grpc-status trailer" in str(e):
                # Expected when envoy/cloudflare times out the stream — just reconnect
                log.info("Stream closed by proxy (missing trailer), will reconnect")
            else:
                raise


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(description="Sparkplug DDATA watchdog")
    parser.add_argument(
        "--test",
        action="store_true",
        help="Test mode: log everything but do not actually restart services",
    )
    args = parser.parse_args()

    base_url = os.environ.get("NFURL")
    client_id = os.environ.get("NF_CLIENT_ID")
    client_secret = os.environ.get("NF_CLIENT_SECRET")

    if not base_url:
        sys.exit("NFURL environment variable is required")
    if not client_id or not client_secret:
        sys.exit("NF_CLIENT_ID and NF_CLIENT_SECRET are required")

    if args.test:
        log.info("Starting in TEST MODE — restarts will be skipped")

    log.info("Connecting to %s", base_url)

    token_mgr = TokenManager(base_url, client_id, client_secret)

    log_svc = LogServiceClientSync(
        base_url, protocol=ProtocolType.GRPC_WEB, send_compression=None
    )
    platform = PlatformConfigClientSync(
        base_url, protocol=ProtocolType.GRPC_WEB, send_compression=None
    )

    while True:
        try:
            token_mgr.get_token()
            log.info("Token valid, starting log stream")
            watch_logs(log_svc, platform, token_mgr, test_mode=args.test)
        except KeyboardInterrupt:
            log.info("Shutting down")
            break
        except Exception:
            log.exception("Stream error — reconnecting in 10s")
            time.sleep(10)
        finally:
            pass

    log_svc.close()
    platform.close()


if __name__ == "__main__":
    main()
