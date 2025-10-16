#!/usr/bin/env python3
import argparse
import gzip
import json
import os
import ssl
import sys
import time
import uuid
from datetime import datetime, timezone

# --- MQTT (paho-mqtt) ---
import paho.mqtt.client as mqtt
from sparkplug_b.sparkplug_b_pb2 import Payload


# ---------- Helpers ----------
def parse_recovery_time(s: str) -> datetime:
    """
    Accepts:
      - ISO 8601 (e.g., '2025-10-16T14:30:00Z' or '2025-10-16 14:30:00-04:00')
      - 'now'
      - integer/float epoch seconds (e.g., '1697467800.123')
      - integer epoch milliseconds (13 digits)
    Returns timezone-aware UTC datetime.
    """
    if s.lower() == "now":
        return datetime.now(timezone.utc)

    # Epoch ms
    if s.isdigit() and len(s) >= 13:
        ms = int(s)
        return datetime.fromtimestamp(ms / 1000.0, tz=timezone.utc)

    # Epoch seconds
    try:
        if s.replace(".", "", 1).isdigit():
            sec = float(s)
            return datetime.fromtimestamp(sec, tz=timezone.utc)
    except Exception:
        pass

    # ISO 8601
    try:
        # Allow trailing 'Z'
        if s.endswith("Z"):
            s = s[:-1] + "+00:00"
        dt = datetime.fromisoformat(s)
        if dt.tzinfo is None:
            dt = dt.replace(tzinfo=timezone.utc)
        return dt.astimezone(timezone.utc)
    except Exception as e:
        raise ValueError(f"Could not parse recovery_time '{s}': {e}")


def ms_since_epoch(dt: datetime) -> int:
    return int(dt.timestamp() * 1000)


def build_log_position(dt: datetime) -> int:
    """
    Log Position = (unix_ms << 10)
    """
    return ms_since_epoch(dt) << 10


def build_payload_json(log_position: int, payload_uuid: str, timestamp_ms: int) -> bytes:
    """
    A simple, Sparkplug-like JSON shape. Your custom implementation can map this.
    We include an explicit datatype so a receiver can validate/convert consistently.
    """
    payload = {
        "timestamp": timestamp_ms,
        "uuid": payload_uuid,
        "seq": 0,  # optional; receivers may ignore
        "metrics": [
            {
                "name": "Log Position",
                "datatype": "Int64",
                "value": log_position,
            }
        ],
    }
    return json.dumps(payload, separators=(",", ":"), ensure_ascii=False).encode("utf-8")


def build_payload_proto(log_position: int, payload_uuid: str, timestamp_ms: int) -> bytes:
    """
    Build an org.eclipse.tahu Sparkplug Payload protobuf.
    Requires a compiled Python module for the Payload proto to be importable.
    """
    msg = Payload()
    # standard fields
    msg.timestamp = timestamp_ms
    msg.uuid = payload_uuid
    # You can set 'seq' if your receiver expects it; leaving 0 is fine here.
    msg.seq = 0

    m = msg.metrics.add()
    m.name = "Log Position"

    # We prefer Int64 / Long type for the left-shifted timestamp.
    # Most generated modules expose it as Metric.Datatype.Int64 or Long; try Int64 first.
    int64_val = None
    m.datatype = 4
    int64_val = "long_value"

    setattr(m, int64_val, int(log_position))
    return msg.SerializeToString()


def maybe_gzip(data: bytes, use_gzip: bool) -> bytes:
    return gzip.compress(data) if use_gzip else data


def connect_mqtt(
    broker: str,
    port: int,
    client_id: str | None,
    username: str | None,
    password: str | None,
    certfile: str | None,
    keyfile: str | None,
) -> mqtt.Client:
    client = mqtt.Client(client_id=client_id or mqtt._base62(uuid.uuid4().int, padding=False), clean_session=True)

    if username:
        client.username_pw_set(username, password or None)

    # TLS if cert/key provided
    if certfile or keyfile:
        ssl_ctx = ssl.create_default_context()
        # If using mutual TLS:
        if certfile and keyfile:
            ssl_ctx.load_cert_chain(certfile=certfile, keyfile=keyfile)
        client.tls_set_context(ssl_ctx)

    # Simple callbacks (optional logging)
    def on_connect(c, userdata, flags, rc, properties=None):
        if rc == 0:
            print("Connected to MQTT broker.")
        else:
            print(f"MQTT connection failed with code {rc}.")

    client.on_connect = on_connect
    client.connect(broker, port, keepalive=30)
    client.loop_start()
    # wait briefly for connect (optional)
    for _ in range(20):
        if client.is_connected():
            break
        time.sleep(0.05)
    return client


def main():
    env = os.environ
    parser = argparse.ArgumentParser(description="Send a Sparkplug recovery request on an NCMD topic.")

    parser.add_argument("--recovery_time", required=True,
                        help="Recovery point time (ISO 8601, 'now', epoch seconds, or epoch ms).")
    parser.add_argument("--command_topic", required=True,
                        help="MQTT topic to publish to (e.g., spBv1.0/normalgw/NCMD/001)")

    parser.add_argument("--payload_format", choices=["proto", "proto+gzip", "json", "json+gzip"],
                        default="proto+gzip", help="Payload wire format.")

    # MQTT connection params (env fallbacks)
    parser.add_argument("--mqtt_broker", default=env.get("MQTT_BROKER", "localhost"))
    parser.add_argument("--mqtt_port", type=int, default=int(env.get("MQTT_PORT", "1883")))
    parser.add_argument("--mqtt_client_id", default=env.get("MQTT_CLIENT_ID"))
    parser.add_argument("--mqtt_username", default=env.get("MQTT_USERNAME"))
    parser.add_argument("--mqtt_password", default=env.get("MQTT_PASSWORD"))
    parser.add_argument("--mqtt_certfile", default=env.get("MQTT_CERTFILE"))
    parser.add_argument("--mqtt_keyfile", default=env.get("MQTT_KEYFILE"))

    # Some folks also export a host via a nonstandard var; accept it if present
    parser.add_argument("--mqtt_hostfile", default=env.get("MQTT_HOSTFILE"))

    parser.add_argument("--qos", type=int, default=1, choices=[0, 1, 2])
    parser.add_argument("--retain", action="store_true", help="Set retained flag on publish (usually false for NCMD).")

    args = parser.parse_args()

    # Allow MQTT_HOSTFILE to override if present & looks like a hostname:port
    if args.mqtt_hostfile and ":" in args.mqtt_hostfile and not args.mqtt_broker:
        host, p = args.mqtt_hostfile.split(":", 1)
        args.mqtt_broker = host.strip()
        try:
            args.mqtt_port = int(p.strip())
        except:
            pass

    # Parse recovery time and compute Log Position
    dt = parse_recovery_time(args.recovery_time)
    ts_ms = ms_since_epoch(dt)
    log_position = build_log_position(dt)
    payload_uuid = str(uuid.uuid4())

    # Build payload
    use_gzip = args.payload_format.endswith("+gzip")
    fmt = args.payload_format.replace("+gzip", "")

    if fmt == "json":
        wire = build_payload_json(log_position, payload_uuid, ts_ms)
    elif fmt == "proto":
        wire = build_payload_proto(log_position, payload_uuid, ts_ms)
    else:
        raise ValueError(f"Unknown payload_format '{args.payload_format}'")

    wire = maybe_gzip(wire, use_gzip)

    # Connect and publish
    client = connect_mqtt(
        broker=args.mqtt_broker,
        port=args.mqtt_port,
        client_id=args.mqtt_client_id,
        username=args.mqtt_username,
        password=args.mqtt_password,
        certfile=args.mqtt_certfile,
        keyfile=args.mqtt_keyfile,
    )

    print(f"Publishing recovery request to '{args.command_topic}' with QoS={args.qos}, retain={args.retain}")
    print(f"  UUID: {payload_uuid}")
    print(f"  Recovery time (UTC): {dt.isoformat()}")
    print(f"  Timestamp ms: {ts_ms}")
    print(f"  Log Position (ms<<10): {log_position}")
    print(f"  Payload format: {args.payload_format}")

    info = client.publish(args.command_topic, payload=wire, qos=args.qos, retain=args.retain)
    info.wait_for_publish(timeout=5.0)

    if info.rc is not None and info.rc != mqtt.MQTT_ERR_SUCCESS:
        print(f"Publish returned error code: {info.rc}", file=sys.stderr)
        sys.exit(2)

    # Give the network loop a moment to flush
    time.sleep(0.1)
    client.loop_stop()
    client.disconnect()
    print("Done.")

if __name__ == "__main__":
    main()
