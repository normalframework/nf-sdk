"""Configure the Sparkplug integration via the Platform API.

This example sets environment variables to configure the Sparkplug MQTT
integration.  By default it configures a basic MQTT broker connection;
pass --transport to select a different transport (aws, azure, http).

To clear a field (e.g. remove a password), pass an empty string:
    --password ''

Usage:
    python configure-sparkplug.py --broker mqtt.example.com --port 1883
    python configure-sparkplug.py --broker mqtt.example.com --port 8883 --protocol tls
    python configure-sparkplug.py --broker mqtt.example.com --username user --password secret
    python configure-sparkplug.py --password ''  # clear the password
    python configure-sparkplug.py --transport aws --aws-domain a1b2c3-ats.iot.us-east-1.amazonaws.com --aws-thing myThing
    python configure-sparkplug.py --transport azure --azure-connection-string 'HostName=...;DeviceId=...;SharedAccessKey=...'
    python configure-sparkplug.py --transport http --http-url https://example.com/sparkplug
    python configure-sparkplug.py --transport disabled  # turn off sparkplug

Environment:
    NFURL, NF_CLIENT_ID, NF_CLIENT_SECRET -- see helpers.py
"""

import sys
sys.path.append("../..")
import argparse

from helpers import NfClient, print_response

parser = argparse.ArgumentParser(
    description="Configure the Sparkplug integration",
    formatter_class=argparse.RawDescriptionHelpFormatter,
    epilog=__doc__,
)

# Transport selection
parser.add_argument("--transport",
                    choices=["mqtt", "aws", "azure", "http", "disabled"],
                    help="Transport type")

# Sparkplug identity
parser.add_argument("--namespace", help="Sparkplug namespace (default: spBv1.0)")
parser.add_argument("--group-id", help="Sparkplug group ID (default: normalgw)")
parser.add_argument("--node-id", help="Sparkplug node ID (default: 001)")
parser.add_argument("--node-name", help="Sparkplug node display name")

# MQTT broker settings
parser.add_argument("--broker", help="MQTT broker hostname (pass '' to clear)")
parser.add_argument("--port", type=int, help="MQTT broker port (default: 1883)")
parser.add_argument("--protocol", choices=["tcp", "tls"],
                    help="MQTT protocol (default: tcp)")
parser.add_argument("--client-id", dest="mqtt_client_id",
                    help="MQTT client ID (defaults to node ID)")
parser.add_argument("--username", help="MQTT username (pass '' to clear)")
parser.add_argument("--password", help="MQTT password (pass '' to clear)")

# Message options
parser.add_argument("--payload-format",
                    choices=["proto+gzip", "proto", "json", "json+gzip"],
                    help="Payload format (default: proto+gzip)")
parser.add_argument("--max-metrics", type=int,
                    help="Max metrics per message (default: 5000)")
parser.add_argument("--use-point-name", action="store_true", default=None,
                    help="Use point name as metric name instead of UUID")
parser.add_argument("--auto-recover", action="store_true", default=None,
                    help="Automatically send recovery data on startup")

# AWS IoT settings
parser.add_argument("--aws-domain", help="AWS IoT Core domain endpoint")
parser.add_argument("--aws-thing", help="AWS IoT Things name")

# Azure IoT settings
parser.add_argument("--azure-connection-string",
                    help="Azure IoT Hub connection string (EdgeHubConnectionString)")

# HTTP transport settings
parser.add_argument("--http-url", help="HTTP endpoint URL for sparkplug payloads")
parser.add_argument("--http-authorization", help="Authorization header for HTTP transport")

args, _ = parser.parse_known_args()

# Print help if no options were provided
if all(v is None for v in vars(args).values()):
    parser.print_help()
    sys.exit(1)


def env_string(var_id, value):
    """Build an EnvironmentVariable with a string value."""
    return {"id": var_id, "string": value}

def env_integer(var_id, value):
    """Build an EnvironmentVariable with an integer value."""
    return {"id": var_id, "integer": str(value)}

def env_boolean(var_id, value):
    """Build an EnvironmentVariable with a boolean value."""
    return {"id": var_id, "boolean": value}


# -- Build the list of variables to set --
# Only includes options that were explicitly passed on the command line.
# To clear a field, pass an empty string (e.g. --password '').
variables = []

def add_string(var_id, value):
    if value is not None:
        variables.append(env_string(var_id, value))

def add_integer(var_id, value):
    if value is not None:
        variables.append(env_integer(var_id, value))

def add_boolean(var_id, value):
    if value is not None:
        variables.append(env_boolean(var_id, value))

# Transport
add_string("SPARKPLUG_TRANSPORT", args.transport)

# Sparkplug identity
add_string("SPARKPLUG_NAMESPACE", args.namespace)
add_string("SPARKPLUG_GROUP_ID", args.group_id)
add_string("SPARKPLUG_NODE_ID", args.node_id)
add_string("SPARKPLUG_NODE_NAME", args.node_name)

# Message options
add_string("SPARKPLUG_PAYLOAD_FORMAT", args.payload_format)
add_integer("SPARKPLUG_MAX_METRICS_PER_MESSAGE", args.max_metrics)
add_boolean("SPARKPLUG_USE_POINT_NAME", args.use_point_name)
add_boolean("SPARKPLUG_AUTO_RECOVER", args.auto_recover)

# MQTT broker settings (sent regardless of transport -- the server
# ignores them when the transport is not mqtt)
add_string("MQTT_BROKER", args.broker)
add_integer("MQTT_PORT", args.port)
add_string("MQTT_PROTOCOL", args.protocol)
add_string("MQTT_CLIENT_ID", args.mqtt_client_id)
add_string("MQTT_USERNAME", args.username)
add_string("MQTT_PASSWORD", args.password)

# AWS IoT
add_string("AWS_IOT_CORE_DOMAIN", args.aws_domain)
add_string("AWS_THINGS_NAME", args.aws_thing)

# Azure IoT
add_string("EdgeHubConnectionString", args.azure_connection_string)

# HTTP transport
add_string("SPARKPLUG_HTTP_URL", args.http_url)
add_string("SPARKPLUG_HTTP_AUTHORIZATION", args.http_authorization)


# -- Apply configuration --
client = NfClient()
res = client.post("/api/v1/platform/env", json={
    "variables": variables,
})
print_response(res)
