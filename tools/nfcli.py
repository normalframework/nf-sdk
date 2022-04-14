"""
Normal Framework CLI utils.

Copyright (c) 2022 Normal Software Inc.
"""

import sys
import argparse
import urllib.request
import urllib.parse
import time
import json
import collections

def errorString(err):
    err = err["error"]
    if err.get("errorClass"):
        return err.get("errorClass") + " (" + err.get("errorCode") + ")"
    elif err.get("rejectReason"):
        return err.get("rejectReason")
    elif err.get("abortReason"):
        return err.get("abortReason")

def h1(title):
    print ("\n" + title + "\n" + "=" * len(title))

class NFSite(object):
    def __init__(self, base, layer):
        self.base = base.rstrip("/")
        self.layer = layer

    def get(self, api, **params):
        qs = urllib.parse.urlencode(params)
        # print(self.base + api, qs)
        req = urllib.request.Request(self.base + api + "?" + qs)
        with urllib.request.urlopen(req) as response:
            if response.code == 200:
                data = json.load(response)
                return data
    
    def summarize_errors(self, verbose=False):
        # last 15 minutes
        first_version = "{}-0".format(int(time.time() * 1000) - (900 * 1000 * 1))
        # fetch the error log from the first version and summarize it

        uuid_counts = collections.Counter()
        device_counts = collections.Counter()
        device_object_counts = collections.defaultdict(collections.Counter)
        device_error_kinds = collections.defaultdict(collections.Counter)
        device_names = {}
        object_error_kinds = collections.defaultdict(set)
        mac_counts = collections.Counter()
        uuid_meta = {}
        while True:
            chunk = self.get("/api/v1/point/updates/errors", layer=self.layer,
                             version=first_version, limit=1000, with_metadata=True)
            if chunk is None: break
            for error in chunk:
                uuid_counts[error["uuid"]] += 1
                try:
                    if error["point"]["attrs"]["type"] == "OBJECT_DEVICE":
                        continue
                    # save a copy of the metadata for printing later
                    uuid_meta[error["uuid"]] = error["point"]["attrs"]
                    device_id = error["point"]["attrs"]["device_id"] + "," + error["point"]["attrs"]["device_uuid"]
                    # one error for this device
                    device_counts[device_id] += 1
                    # one error for object in the device
                    device_object_counts[device_id][error["uuid"]] += 1

                    device_error_kinds[device_id][errorString(error["error"])] += 1
                    object_error_kinds[error["uuid"]].add(errorString(error["error"]))
                    device_names[device_id] = error["point"]["attrs"]["device_prop_object_name"]
                except KeyError:
                    pass
                try:
                    mac = error["point"]["attrs"]["bacnet_mac"]
                    mac_counts[mac] += 1
                except KeyError:
                    pass
                first_version = error["version"]
            if len(chunk) < 1000:
                break

        h1("Errors by device")
        print ("{0:<60}| {1: <13}| {2}".format("device_id", "error_count", "error types"))
        print ("-"*60 + "|-" + "-"*13 + "|" + "-" * 30)
        for d, count in device_counts.items():
            print ("{0: <60}| {1: <13}| {2}".format(
                d  + " (" + device_names.get(d, "") + ")",
                count,
                ", ".join(("{}={}".format(k, v)
                           for (k, v) in sorted(device_error_kinds[d].items())))
            ))

            if not verbose:
                continue
            for i, u in enumerate(device_object_counts[d].keys()):
                print ("\t\t" + u)
                print ("\t\t" + str(object_error_kinds[u]))
                for k, v in sorted(uuid_meta[u].items()):
                    print ("\t\t\t{}={}".format(k, v))
                if i == 10:
                    print ("\t\t...")
                    break

        h1("Errors by MAC")
        print ("{0:<47}| {1: <13}".format("bacnet_mac", "error_count"))
        print ("-"*47 + "|-" + "-"*13)
        for mac, count in sorted(mac_counts.items(), key=lambda x: x[1]):
            print ("{0: <47}| {1: <13}".format(mac, count))

if __name__ == '__main__':
    parser = argparse.ArgumentParser("Normal Framework CLI")
    parser.add_argument("command", metavar="command",
                        help="subcommand: [errors]")
    parser.add_argument("--url", default="http://localhost:8080",
                        help="base URL of NF site")
    parser.add_argument("--layer", default="hpl:bacnet:1",
                        help="layer name to query")
    parser.add_argument("--verbose", "-v", default=False, action="store_true",
                        help="layer name to query")
    args = parser.parse_args()
    site = NFSite(args.url, args.layer)

    # dispatch commands
    if args.command == 'errors':
        site.summarize_errors(args.verbose)
    else:
        if sys.stderr.isatty():
            sys.stderr.write("Invalid command: '" + args.command + "'\n\n")
            parser.print_help()
        sys.exit(1)


