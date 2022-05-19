#!/usr/bin/env python3
"""Normal Framework CLI utils

Copyright (c) 2022 Normal Software Inc.

The goal of this utility is to interact with a NF instance from the
command line, to allow scripting and automation without the need to
call the API directly.

Commands
--------

 * Backup: create a zip archive of all "hard" state in an NF instance, 
   including points database; templates; BACnet settings; BACnet objects;
 * Restore: update a new NF instance with a backup
 * Find points in an NF instance and apply bulk operations like deleting 
   or changing the polling rate of those points
 * Examine recent errors

Installation
------------

The only dependency is a working Python 3 installation.  nfcli
intentionally only uses the python standard library, and is just one
file.  Copy it onto a machine with python and you should be able to
use it.

"""
import os
import sys
import logging
import argparse
import urllib.request
import urllib.parse
import time
import json
import collections
import zipfile
import re
import csv

# page size to get points from points api
CHUNKSZ=500
# how many times to retrie GET requests
RETRIES=3

log = logging.getLogger()

class Subcommand(object):
    # override with command name
    name = None

    # override to add flags just for
    def add_arguments(self, parser):
        return

    def post(self, api, data):
        req = urllib.request.Request(self.base + api, json.dumps(data).encode('utf-8'), headers={
            "Content-type": "application/json"})
        with urllib.request.urlopen(req) as response:
            if response.code == 200:
                log.debug("POST code=%d url=%s", response.code, api)
            else:
                log.warning("POST code=%d url=%s", response.code, api)

    def delete(self, api, data):
        req = urllib.request.Request(self.base + api, json.dumps(data).encode('utf-8'), headers={
            'Content-type': 'application/json'}, method='DELETE')
        with urllib.request.urlopen(req) as response:
            if response.code == 200:
                log.debug("DELETE code=%d url=%s", response.code, api)
            else:
                log.warning("DELETE code=%d url=%s", response.code, api)
                                     
    
    def get(self, api, **params):
        qs = urllib.parse.urlencode(params)
        req = urllib.request.Request(self.base + api + "?" + qs)

        for i in range(0, RETRIES):
            with urllib.request.urlopen(req) as response:
                log.debug("GET code=%d url=%s", response.code,  api + "?" + qs)
                if response.code == 200:
                    return json.load(response)
                else:
                    break

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

class ErrorsCommand(Subcommand):
    name = "errors"
    
    def run(self, args):
        self.layer = args.layer
        self.verbose = args.verbose
        
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

            if not args.verbose:
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

class BackupCommand(Subcommand):
    """Download and save the state of a NF instance.  This means

 * The Points database
 * BACnet settings
 * Templates

Backups are only guaranteed to work with the same minor version.  To
restore an older backup, you must reinstall the matching version,
restore, and then upgrade again.
    """
    name = "backup"

    def add_arguments(self, parser):
        parser.add_argument("--output", default="nf.backup",
                            help="filename to output to")

    def save_endpoint(self, endpoint, filename, archive, **kwargs):
        try:
            with archive.open(filename, mode="w") as fp:
                data = self.get(endpoint, **kwargs)
                fp.write(json.dumps(data).encode('utf-8'))
                return data
        except Exception as e:
            log.warning("Error fetching resource " + endpoint + ": " + str(e))
            return {}
        

    def run(self, args):

        # all of the archived endpoints go into one zip archive
        with zipfile.ZipFile(args.output, mode="w",
                             compression=zipfile.ZIP_DEFLATED) as archive:
            # save the info -- these aren't restored but for information
            info = self.save_endpoint("/api/v1/platform/info", "info.jsonl", archive)
            log.info("backing up %s at %s", info.get("siteName"), self.base)

            # save the layers -- these aren't restored
            layers = self.save_endpoint("/api/v1/point/layers", "layers.jsonl", archive)
            # save the bacnet settings
            self.save_endpoint("/api/v1/bacnet/configuration", "bacnet.jsonl", archive)
            log.info("archived BACnet datalink configuration")
            # save the templates
            templates = self.save_endpoint("/api/v1/templates", "templates.jsonl",
                               archive, content=True)
            log.info("archived up %d templates", len(templates['templates']))

            config = self.save_endpoint("/api/v1/point/configkeys", "configkeys.jsonl", archive)
            log.info("archived %d configuration keys", len(config["values"]))

            objects = self.save_endpoint("/api/v1/bacnet/local/", "bacnet-objects.jsonl", archive)
            log.info("archived %d local bacnet objects", len(objects.get("objects", [])))

            # download the point database for each base layer, and add
            # all of the points
            with archive.open("points.jsonl", mode="w") as fp:
                pointcount = 0
                for l in layers['layers']:
                    if l["kind"] != "LAYER_BASE":
                        continue
                    log.debug("backing up layer %s", l.get("name"))
                    offset = 0
                    while True:
                        data = self.get("/api/v1/point/points",
                                        layer=l["name"],
                                        query="*",
                                        page_size=CHUNKSZ,
                                        page_offset=offset)
                        if len(data["points"]) == 0:
                            break
                        else:
                            offset += CHUNKSZ
                        for p in data["points"]:
                            fp.write(json.dumps(p).encode("utf-8"))
                            fp.write("\n".encode("utf-8"))
                            pointcount += 1
                    log.info("archived count=%d layer=%s", pointcount, l["name"])
            log.info("archive written to %s", args.output)

class RestoreCommand(Subcommand):
    name = "restore"

    def add_arguments(self, parser):
        parser.add_argument("backup", metavar="backup", 
                            help="backup archive")
        parser.add_argument("--force", "-f", action="store_true", default=False,
                            help="override version check"
                            )
        parser.add_argument("--no-templates", action="store_true", default=False,
                            help="don't restore saved templates")
        parser.add_argument("--no-points", action="store_true", default=False,
                            help="don't restore saved point database")
        parser.add_argument("--no-bacnet-settings", action="store_true", default=False,
                            help="don't restore bacnet settings")
        parser.add_argument("--no-config", action="store_true", default=False,
                            help="don't resture UI configuration")

    def restore_bacnet_settings(self, archive):
        """Restore the BACnet settings (datalink, BBMD, etc)."""
        log.info("restoring BACnet datalink settings")
        with archive.open("bacnet.jsonl", "r") as fp:
            settings = json.load(fp)
        self.post("/api/v1/bacnet/configuration", settings)

    def restore_bacnet_objects(self, archive):
        """Restore all local BACnet objects"""
        log.info("restoring local BACnet objects")
        with archive.open("bacnet-objects.jsonl", "r") as fp:
            objects = json.load(fp)
        for obj in objects.get("objects", []):
            self.post("/api/v1/bacnet/local", obj)

    def restore_config(self, archive):
        with archive.open("configkeys.jsonl", "r") as fp:
            config = json.load(fp)
        log.info("restoring %d configuration settings", len(config["values"]))
        for k, v in config["values"].items():
            self.post("/api/v1/point/configkeys", {
                "key": k,
                "value": v })

    def restore_templates(self, archive):
        """Restore all of the templates, and reenable any that were running
        """

        log.info("restoring templates")
        enable_count, count = 0, 0
        with archive.open("templates.jsonl", "r") as fp:
            templates = json.load(fp)

            for t in templates["templates"]:
                del t["path"]
                count += 1
                self.post("/api/v1/templates", {"template": t})
                try:
                    enabled = t["status"]["enabled"]
                except KeyError:
                    enabled = False
                if enabled:
                    enable_count += 1
                    # enable any previously enabled templates
                    log.info("enabling template %s", t["name"])
                    self.post("/api/v1/templates/" + t["name"], {
                        "name": t["name"],
                        "enable": True,
                        "inputLayer": t["status"]["inputLayer"],
                        "outputLayer": t["status"]["outputLayer"],
                        })
        log.info("restored %d templates (%d enabled)", count, enable_count)

    def restore_points(self, archive):
        """Restore the object database"""
        count = 0
        log.info("starting to reload point database")
        with archive.open("points.jsonl", "r") as fp:
            points = {"points": []}
            for l in fp.readlines():
                count += 1
                points["points"].append(json.loads(l))
                if len(points["points"]) == CHUNKSZ:
                       self.post("/api/v1/point/points", points)
                       points["points"] = []
            self.post("/api/v1/point/points", points)
        log.info("restored %d points", count)

    def run(self, args):
        with zipfile.ZipFile(args.backup, mode="r") as archive:
            info = json.load(archive.open("info.jsonl"))
            version = info.get("version", None)
            site_version = self.get("/api/v1/platform/info")
            if version != site_version.get("version", None):
                log.warning("site is running a different version from the backup (site=%s, backup=%s)",
                         site_version.get("version", None), version)
                if not args.force:
                    return
            
            log.info("restoring site \"%s\" at \"%s\"", info.get("siteName", ""), self.base)

            if not args.no_bacnet_settings:
                self.restore_bacnet_settings(archive)
                try:
                    self.restore_bacnet_objects(archive)
                except Exception as e:
                    log.warning("error restoring BACnet local objects: " + str(e))
            if not args.no_templates:
                self.restore_templates(archive)
            if not args.no_config:
                self.restore_config(archive)
            if not args.no_points:
                self.restore_points(archive)

            # layers are not restored now, only the default layers
            # will exist.

def escape_tag(v):
    return re.sub(r"([/:#_\.\-\%\&\^\ \{\}\(\)\[\\\\`\~|\]\!\@\*\,\<\>\;\'\"])", r"\\\1", v)


class FindCommand(Subcommand):
    name = "find"

    def add_arguments(self, parser):
        parser.add_argument("filters", metavar="N", nargs="*", default=['*'],
                            help="property filters to apply, in the form of key=value")
        parser.add_argument("--query", "-q", help="query to apply to find", default="")
        parser.add_argument("--json", help="output json", default=False, action="store_true")
        parser.add_argument("--csv", help="output CSV", default=False, action="store_true")
        parser.add_argument("--delete", "-d", dest="delete", action="store_true", default=False,
                            help="delete matching points")
        parser.add_argument("--period", "-p", type=int, default=None,
                            help="update polling period for matching points")

    def build_query(self, args):
        """Build a query string from the command line
        """
        self.layers = self.get("/api/v1/point/layers/" + args.layer)
        if len(self.layers.get("layers", [])) != 1:
            raise Exception("Layer Not found")
        types = (dict(zip(self.layers["layers"][0]["components"], self.layers["layers"][0]["componentTypes"])))
        types["uuid"] = "TAG"
        types["period"] = "NUMERIC"
        if "*" in args.filters and len(args.filters) == 1:
            return "*"
        q = ""
        for fltr in args.filters:
            splt = fltr.split("=")
            if len(splt) < 2:
                raise ValueError("Invalid filter: " + fltr)
            k, v = splt[0], "=".join(splt[1:])
            if types.get(k):
                t = types.get(k)
            elif types.get("attr_" + k):
                t = types.get("attr_" + k)
                k = "attr_" + k
            else:
                raise ValueError("Invalid filter: " + k + " is not indexed")
            if t == "TAG":
                q += " @" + k + ":{" + escape_tag(v) + "}"
            elif t == "TEXT":
                q += " @" + k + ":\"" + escape_tag(v) + "\""
            elif t == "NUMERIC":
                q += " @" + k + ":[" + str(int(v)) + "," + str(int(v)) + "]"
        return q

    def output_csv(self, page, out, header=False):
        cols = ["uuid", "period"] + [
                x[5:] if x.startswith("attr_") else x
                for x in 
                self.layers["layers"][0]["components"]]
        if header:
            out.writerow(cols)
        for p in page["points"]:
            row = [p["uuid"], p.get("period", "")]
            row += [p["attrs"].get(a, "") for a in cols[2:]]
            out.writerow(row)
        
    def run(self, args):
        # build the redis query
        query = args.query + self.build_query(args)
        count, offset = CHUNKSZ, 0
        if args.csv:
            out = csv.writer(sys.stdout)
        uuids = []

        # iterate over result pages and save the matching uuids
        while count == CHUNKSZ:
            page = self.get("/api/v1/point/points", query=query, page_size=CHUNKSZ, page_offset=offset)
            offset += CHUNKSZ
            count = len(page["points"])
            for p in page["points"]:
                uuids.append(p["uuid"])
            if args.csv:
                self.output_csv(page, out, header=(offset == CHUNKSZ))
            elif args.json:
                for p in page["points"]:
                    json.dump(p, sys.stdout)
                    sys.stdout.write("\n")
        if not args.json and not args.csv and not args.delete and args.period is None:
            log.info ("found " + str(len(uuids)) + " point" + ("" if len(uuids) == 1 else "s"))
        
        if args.period is not None:
            if sys.stdin.isatty():
                x = input("Really update polling rate for {} points? [y/N]: ".format(len(uuids)))
                if not x.lower().strip() == "y":
                    return
            for i in range(0, len(uuids), CHUNKSZ):
                self.post("/api/v1/point/points", {
                    "points": [{
                        "layer": args.layer,
                        "uuid": u,
                        "period": {
                            "seconds": args.period,
                            },
                        } for u in uuids[i:i+CHUNKSZ]]
                    })
            log.info("updated polling rate for {} points".format(len(uuids)))

        if args.delete:
            if sys.stdin.isatty():
                x = input("Really delete {} points? [y/N]: ".format(len(uuids)))
                if not x.lower().strip() == "y":
                    return
            for i in range(0, len(uuids), CHUNKSZ):
                self.delete("/api/v1/point/points", {
                    "layer": args.layer,
                    "uuids": uuids[i:i+CHUNKSZ]})


if __name__ == '__main__':
    subcommands = [
        ErrorsCommand,
        BackupCommand,
        RestoreCommand,
        FindCommand
    ]
    parser = argparse.ArgumentParser("Normal Framework CLI")
    parser.add_argument("command", metavar="command",
                        help="subcommand: [{}]".format('|'.join((c.name for c in subcommands))))
    parser.add_argument("--url", default=os.getenv("NFURL", "http://localhost:8080"),
                        help="base URL of NF site")
    parser.add_argument("--layer", default="hpl:bacnet:1",
                        help="layer name to query")
    parser.add_argument("--verbose", "-v", default=False, action="store_true",
                        help="print verbose logs")
    if len(sys.argv) < 2:
        parser.print_help()
        sys.exit(1)
    for command_class in subcommands:
        if command_class.name == sys.argv[1]:
            break
    else:
        sys.stderr.write("Invalid command: '" + sys.argv[1] + "'\n\n")
        parser.print_help()
        sys.exit(1)
        
    command = command_class()
    command.add_arguments(parser)

    args = parser.parse_args()
    command.base = args.url.rstrip("/")
    logging.basicConfig(
        format='%(asctime)s %(levelname)-8s %(message)s',
        level=logging.DEBUG if args.verbose else logging.INFO)

    command.run(args)

