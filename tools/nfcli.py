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
import http.client

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

    def post(self, api, data, method="POST"):
        req = urllib.request.Request(self.base + api, json.dumps(data).encode('utf-8'), headers={
            "Content-type": "application/json"}, method=method)
        try:
            with urllib.request.urlopen(req) as response:
                if response.code == 200:
                    log.debug("POST code=%d url=%s", response.code, api)
                else:
                    log.warning("POST code=%d url=%s", response.code, api)
        except urllib.error.HTTPError as e:
            print (e)
            msg = e.headers.get("grpc-message")
            log.error("POST url=%s message=%s", api, msg)#e.headers.get("gprc-message"))
                    
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
    help_text = "Display information about I/O errors encountered"

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
                except KeyError:
                    pass
                    
                # save a copy of the metadata for printing later
                uuid_meta[error["uuid"]] = error["point"]["attrs"]
                try:
                    device_id = error["point"]["attrs"]["device_id"] + "," + error["point"]["attrs"]["device_uuid"]
                except KeyError:
                    device_id = error["point"]["attrs"]["device_address"]
                    
                # one error for this device
                device_counts[device_id] += 1
                # one error for object in the device
                device_object_counts[device_id][error["uuid"]] += 1

                if error["error"].get("error") is not None:
                    device_error_kinds[device_id][errorString(error["error"])] += 1
                    object_error_kinds[error["uuid"]].add(errorString(error["error"]))
                    device_names[device_id] = error["point"]["attrs"]["device_prop_object_name"]
                else:
                    device_error_kinds[device_id][error["error"].get("message")] += 1
                    object_error_kinds[error["uuid"]].add(error["error"].get("message"))
                device_names[device_id] = error["point"]["attrs"]["device_address"]

                    
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
    help_text = "Create a backup archive of all hard state on a NF instance"

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

            profiles = self.save_endpoint("/api/v1/modbus/profiles", "modbus-profiles.jsonl", archive, content=1)
            log.info("archived %d modbus profiles", len(profiles.get("profiles", [])))

            connections = self.save_endpoint("/api/v1/modbus/connections", "modbus-connections.jsonl", archive)
            log.info("archived %d modbus connections", len(connections.get("connections", [])))

            # download the point database for each base layer, and add
            # all of the points
            with archive.open("points.jsonl", mode="w") as fp:
                for l in layers['layers']:
                    pointcount = 0
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
    help_text = "Restore data to an NF site from a saved backup"

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
                            help="don't restore UI configuration")
        parser.add_argument("--no-modbus", action="store_true", default=False,
                            help="don't restore modbus settings")

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

    def restore_modbus_profiles(self, archive):
        with archive.open("modbus-profiles.jsonl", "r") as fp:
            for l in fp.readlines():
                profiles = json.loads(l)
                for p in profiles.get("profiles", []):
                    self.post("/api/v1/modbus/profiles", {"profile": p})

    def restore_modbus_connections(self, archive):
        with archive.open("modbus-connections.jsonl", "r") as fp:
            for l in fp.readlines():
                profiles = json.loads(l)
                for c in profiles.get("connections", []):
                    self.post("/api/v1/modbus/connections", {"connection": c})

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
            if not args.no_modbus:
                self.restore_modbus_profiles(archive)
                self.restore_modbus_connections(archive)

            # layers are not restored now, only the default layers
            # will exist.

def escape_tag(v):
    return re.sub(r"([/:#_\.\-\%\&\^\ \{\}\(\)\[\\\\`\~|\]\!\@\*\,\<\>\;\'\"])", r"\\\1", v)


class VersionCommand(Subcommand):
    name = "version"
    help_text = "Display the software version of the NF instance"

    def run(self, args):
        info = self.get("/api/v1/platform/info")
        print (info.get("version", "unknown"))

class FindCommand(Subcommand):
    name = "find"
    help_text = "Search the object database for points"

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
            invert = ""
            splt = fltr.split("=")
            if len(splt) < 2:
                raise ValueError("Invalid filter: " + fltr)
            k, v = splt[0], "=".join(splt[1:])
            if k[-1] == "!":
                k = k[:-1]
                print (k)
                invert = "-" 
            if types.get(k):
                t = types.get(k)
            elif types.get("attr_" + k):
                t = types.get("attr_" + k)
                k = "attr_" + k
            else:
                raise ValueError("Invalid filter: " + k + " is not indexed")

            # add the appropiate query clause
            if t == "TAG":
                q += " " + invert + "@" + k + ":{" + escape_tag(v) + "}"
            elif t == "TEXT":
                q += " " + invert + "@" + k + ":\"" + escape_tag(v) + "\""
            elif t == "NUMERIC":
                q += " " + invert + "@" + k + ":[" + str(int(v)) + "," + str(int(v)) + "]"
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
            try:
                page = self.get("/api/v1/point/points", query=query, page_size=CHUNKSZ, page_offset=offset, layer=args.layer)
            except Exception as e:
                log.error("Error retrieving chunk: q=%s, page_size=%i, page_offset=%i, layer=%s, error=%s",
                          query, CHUNKSZ, offset, args.layer, e)
                offset += CHUNKSZ
                continue
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


class ListObjectsCommand(Subcommand):
    name = "list-bacnet-objects"

    def as_scalar(self, prop):
        kinds = ["characterString", "double", "real", "signed", "unsigned", "enumerated", "null", "octetString", "boolean"]
        for k in kinds:
            if k in prop:
                return prop[k]
        if len(prop["array"]) > 0:
            return list(map(self.as_scalar, prop["array"]))
        
    def output_object(self, obj, args):
        oid = to_objectid(obj["objectId"]["objectType"]) + "." + str(obj["objectId"]["instance"])
        output = {"object_id": oid}
        
        for prop in obj["props"]:
            output[prop["property"].lower()[5:]] = self.as_scalar(prop["value"])

        return output

    def run(self, args):
            object_list = self.get("/api/v1/bacnet/local")
            headers = set([])
            for o in object_list["objects"]:
                row = self.output_object(o, args)
                headers.update(row.keys())
            headers = list(headers)
            sorted_headers = []
            for h in ("object_id", "object_name", "description", "units"):
                if h in headers:
                    sorted_headers.append(h)
                    del headers[headers.index(h)]
            headers.sort()
            sorted_headers.extend(headers)

            writer = csv.DictWriter(sys.stdout, fieldnames=sorted_headers)
            writer.writeheader()
            for o in object_list["objects"]:
                row = self.output_object(o, args)
                writer.writerow(row)

class DeleteObjectCommand(Subcommand):
    name = "delete-bacnet-object"
    
    def add_arguments(self, parser):
        parser.add_argument("object_id", metavar="object_id", nargs="+", default=[],
                            help="object ids to delete.  for instance, bv.1")
    
    def run(self, args):
        for oid in args.object_id:
            try:
                otype, inst = oid.split(".")
                int(inst)
            except:
                print ("Invalid object identifier: ", oid)
                continue
            full_type = lookup_object_type(otype)
            if not full_type:
                print ("Invalid object type: ", otype)
                continue
            try:
                (self.delete("/api/v1/bacnet/local/{}/{}".format(full_type, inst), {}))
            except urllib.error.HTTPError as e:
                print ("Error deleting", oid, ":", e)

class ObjectCommand(Subcommand):
    def add_arguments(self, parser):
        parser.add_argument("object_id", metavar="object_id", nargs=1, default="",
                            help="object ids to create.  for instance, bv.1 or av")
        parser.add_argument("properties", metavar="properties", nargs="+",
                            help="property values of the form object-name=foo")

    # parse a property string to a prop_id, applicationdatavalue tuple
    def parse_property(self, prop):
        if not "=" in prop:
            raise ValueError("property value must be of the form prop=[type:]value")
        propname = prop[:prop.index("=")]
        propvalue = prop[prop.index("=")+1:]
        try:
            propname = int(propname)
        except ValueError:
            propname = propname.lower().replace("-", "_")
            for pn, val in property_ids.items():
                if "PROP_" + propname.upper() == pn:
                    propname = pn
                    break
            else:
                raise ValueError("invalid or unknown property: ", propname)
        # we have looked up the value

        if propvalue.find(":") > 0:
            typename = propvalue[:propvalue.index(":")]
            if typename in type_map:
                propvalue = {
                    typename: type_map[typename](propvalue[propvalue.index(":")+1:])
                }
            else:
                propvalue = {"character_string": propvalue } 
        else:
            propvalue = {"character_string": propvalue } 

        return (propname, propvalue)
        
    def run(self, args):
        inst = 0
        otype = args.object_id[0].split(".")
        if len(otype) > 1:
            otype, inst = otype
        else:
            otype = otype[0]
        otype = (lookup_object_type(otype))
        if otype == None:
            print ("Invalid object type:", otype)
            return

        props = []
        for val in args.properties:
            pid, pval = self.parse_property(val)
            props.append({
                "property": pid,
                "value": pval })
        res = self.post("/api/v1/bacnet/local", data={
            "object_id": {
                "object_type": otype,
                "instance": inst,
                },
            "props": props}, method=self.method)

class CreateObjectCommand(ObjectCommand):
    name = "create-bacnet-object"
    method = "POST"

class UpdateObjectCommand(ObjectCommand):
    name = "update-bacnet-object"
    require_instance = True
    method = "PATCH"

class CsvSubCommand(Subcommand):

    def run(self, args):
        writer = csv.DictWriter(sys.stdout, fieldnames=self.fieldnames, extrasaction="ignore")
        writer.writeheader()
        for p in self.getlist(args):
            writer.writerow(p)
        

class GetModbusProfiles(CsvSubCommand):
    fieldnames= ["profileName", "uuid", "endianness", "highWordFirst"]
    name = "list-modbus-profiles"
    
    def add_arguments(self, parser):
        parser.add_argument("--maps", help="ouput register maps as well as metadata", default=False, action="store_true")

    def getlist(self, args):
        return self.get("/api/v1/modbus/profiles")["profiles"]

class CreateModbusProfile(Subcommand):
    name = "create-modbus-profile"

    def add_arguments(self, parser):
        parser.add_argument("registers", metavar="registers", nargs=1,
                            help="csv register map")
        parser.add_argument("--name", "-n", help="profile name")
        parser.add_argument("--uuid", "-u", help="profile uuid")
        parser.add_argument("--dry-run", "-d", help="only parse profile file", default=False, action="store_true")

    def parse_register_type(self, t):
        try:
            t = int(t)
            if not t in [1, 2, 3, 4]:
                raise ValueError("invalid function code: " + t)
            else:
                return t
        except ValueError:
            t = t.lower()
            if t == "coil":
                return "REGISTER_COIL"
            elif t == "discrete":
                return "REGISTER_DISCRETE"
            elif t == "holding":
                return "REGISTER_HOLDING"
            elif t == "input":
                return "REGISTER_INPUT"
            else:
                raise ValueError("invalid register type: " + t)

    def parse_data_type(self, t):
        if t == "":
            raise ValueError("Must supply data type")
        return "DATATYPE_" + t.upper().strip()
        

    def parse_address(self, a):
        return int(a)

    def parse_scale(self, s):
        if not s: return 1.
        return float(s)

    def parse_offset(self, o):
        if not o:
            return 0.
        return float(o)


    def load_profile(self, name):
        profile = {}
        def parseVariables(fp):
            for row in fp:
                # skip comment rows and look for variables in them.
                # these can appear anywhere in the file
                if re.match("^\W*#", row):
                    v = row.split(":")
                    if len(v) != 2:
                        continue

                    profile[v[0].strip(" #")] = v[1].strip()
                elif row.strip() == "":
                    continue
                else:
                    yield row
        with open(name, "r") as fp:
            reader = csv.DictReader(parseVariables(fp))
            # {"registerType": "REGISTER_HOLDING", "dataType": "DATATYPE_UINT32", "address": 80, "name": "Battery Voltage", "units": "V", "scaleFactor": 0.001}

            registers = []
            for line in reader:
                reg = {
                    "registerType": self.parse_register_type(line.get("Register Type")),
                    "dataType": self.parse_data_type(line.get("Data Type")),
                    "address": self.parse_address(line.get("Address")),
                    "name": line.get("Name", ""),
                    "units": line.get("Units", "none"),
                    "scaleFactor": self.parse_scale(line.get("Scale Factor")),
                    "offset": self.parse_offset(line.get("Offset")),
                    }
                registers.append(reg)
            profile["registers"] = registers
            return profile

    def run(self, args):
        profile = self.load_profile(args.registers[0])
        if args.uuid:
            profile["uuid"] = args.uuid
        if args.name:
            profile["profileName"] = args.name
        if args.dry_run:
            json.dump(profile, sys.stdout, indent=4)
        else:
            self.post("/api/v1/modbus/profiles", {"profile": profile})

class GetModbusConnections(CsvSubCommand):
    name = "list-modbus-connections"
    fieldnames = ["connectionUuid", "profileUuid", "address", "unitId", "name"]
    def getlist(self, args):
        return self.get("/api/v1/modbus/connections")["connections"]

class DeleteModbusConnection(Subcommand):
    name = "delete-modbus-connection"
    def add_arguments(self, parser):
        parser.add_argument("connection_uuid", metavar="connection_uuid", nargs=1, default="",
                            help="delete a connection")

    def run(self, args):
        self.delete("/api/v1/modbus/connections", {"connection_uuid": args.connection_uuid[0]})

class CreateModbusConnection(Subcommand):
    name = "create-modbus-connection"
    def add_arguments(self, parser):
        parser.add_argument("--profile", "-p", required=True, help="profile uuid to use")
        parser.add_argument("--address", "-a", required=True, help="connection string for device; eg, tcp://192.168.1.10:502")
        parser.add_argument("--unit-id", "-u", default=1, help="modbus unit id")
        parser.add_argument("--name", "-n", default="", help="connection name")

    def run(self, args):
        self.post("/api/v1/modbus/connections", {
            "connection": {
                "profile_uuid": args.profile,
                "address": args.address,
                "name": args.name,
                "unit_id": args.unit_id,
                }
            })

class WatchDataCommand(Subcommand):
    name = "watch-data"

    def add_arguments(self, parser):
        parser.add_argument("--version", default="$", help="streaming version to start with")

    # take a urllib response object and read json objects out of the
    # stream as clumsily as possible
    def read_objects(self, response):
        buf = ""
        while True:
            # to avoid needing to intro an async api we read it a byte
            # at a time, which is, of course, horribly inefficient.
            chunk = response.read(1)
            if not chunk:
                break
            buf += chunk.decode('utf-8')
            while True:
                # remove pesky object separators
                buf = buf.lstrip(",\n [")
                try:
                    # see if the chunk boundary magically is a valid
                    # json object
                    yield json.loads(buf)
                    buf = ""
                    continue
                except json.JSONDecodeError as err:
                    if err.pos == len(buf):
                        break
                    try:
                        # more likely there is a valid object at the
                        # beginning of the buffer now, so parse it and
                        # keep going
                        yield json.loads(buf[:err.pos])
                        buf = buf[err.pos:]
                    except:
                        break
        
    
    def run(self, args):
        version = args.version
        while True:
            req = urllib.request.Request(self.base + "/api/v1/point/updates/data?wait=1&version=" + version)
            # we get a streaming reply.  we need to fish the objects out
            # of the stream so we can print them.
            last_version = ""
            with urllib.request.urlopen(req) as response:
                # use our iterator reader to read the response objects
                try:
                    for o in self.read_objects(response):
                        if o["version"] == last_version:
                            continue
                        else:
                            last_version = o["version"]
                        val = o["value"]
                        ts = val.pop("ts")
                        value = val.pop(list(val.keys())[0])
                        version = o["version"]
                        print (o["uuid"], ts, value, o["version"], o["layer"])
                except http.client.IncompleteRead:
                    # this is what you get when envoy times out
                    pass


def lookup_object_type(otype):
    for name, e in object_types.items():
        if to_objectid(name) == otype:
            return name
        

def to_objectid(oid):
    return ''.join(re.findall("\_([a-z])", oid.lower()))

def parse_bool(x):
    if isinstance(x, str):
        if x.lower() in ["false", "0"]:
            return False
        else:
            return True
    else:
        return bool(x)

type_map = {
    "real": float,
    "double": float,
    "character_string": str,
    "octet_string": str,
    "signed": int,
    "unsigned": int,
    "boolean": parse_bool,
    "null": lambda x: True,
    "enumerated": str,
}

object_types = {
    "OBJECT_ANALOG_INPUT" : 0,
    "OBJECT_ANALOG_OUTPUT" : 1,
    "OBJECT_ANALOG_VALUE" : 2,
    "OBJECT_BINARY_INPUT" : 3,
    "OBJECT_BINARY_OUTPUT" : 4,
    "OBJECT_BINARY_VALUE" : 5,
    "OBJECT_CALENDAR" : 6,
    "OBJECT_COMMAND" : 7,
    "OBJECT_DEVICE" : 8,
    "OBJECT_EVENT_ENROLLMENT" : 9,
    "OBJECT_FILE" : 10,
    "OBJECT_GROUP" : 11,
    "OBJECT_LOOP" : 12,
    "OBJECT_MULTI_STATE_INPUT" : 13,
    "OBJECT_MULTI_STATE_OUTPUT" : 14,
    "OBJECT_NOTIFICATION_CLASS" : 15,
    "OBJECT_PROGRAM" : 16,
    "OBJECT_SCHEDULE" : 17,
    "OBJECT_AVERAGING" : 18,
    "OBJECT_MULTI_STATE_VALUE" : 19,
    "OBJECT_TRENDLOG" : 20,
    "OBJECT_LIFE_SAFETY_POINT" : 21,
    "OBJECT_LIFE_SAFETY_ZONE" : 22,
    "OBJECT_ACCUMULATOR" : 23,
    "OBJECT_PULSE_CONVERTER" : 24,
    "OBJECT_EVENT_LOG" : 25,
    "OBJECT_GLOBAL_GROUP" : 26,
    "OBJECT_TREND_LOG_MULTIPLE" : 27,
    "OBJECT_LOAD_CONTROL" : 28,
    "OBJECT_STRUCTURED_VIEW" : 29,
    "OBJECT_ACCESS_DOOR" : 30,
    "OBJECT_TIMER" : 31,
    "OBJECT_ACCESS_CREDENTIAL" : 32,      
    "OBJECT_ACCESS_POINT" : 33,
    "OBJECT_ACCESS_RIGHTS" : 34,
    "OBJECT_ACCESS_USER" : 35,
    "OBJECT_ACCESS_ZONE" : 36,
    "OBJECT_CREDENTIAL_DATA_INPUT" : 37,  
    "OBJECT_NETWORK_SECURITY" : 38,       
    "OBJECT_BITSTRING_VALUE" : 39,        
    "OBJECT_CHARACTERSTRING_VALUE" : 40,  
    "OBJECT_DATE_PATTERN_VALUE" : 41,     
    "OBJECT_DATE_VALUE" : 42,     
    "OBJECT_DATETIME_PATTERN_VALUE" : 43, 
    "OBJECT_DATETIME_VALUE" : 44, 
    "OBJECT_INTEGER_VALUE" : 45,  
    "OBJECT_LARGE_ANALOG_VALUE" : 46,     
    "OBJECT_OCTETSTRING_VALUE" : 47,      
    "OBJECT_POSITIVE_INTEGER_VALUE" : 48, 
    "OBJECT_TIME_PATTERN_VALUE" : 49,     
    "OBJECT_TIME_VALUE" : 50,     
    "OBJECT_NOTIFICATION_FORWARDER" : 51, 
    "OBJECT_ALERT_ENROLLMENT" : 52,       
    "OBJECT_CHANNEL" : 53,        
    "OBJECT_LIGHTING_OUTPUT" : 54,        
    "OBJECT_BINARY_LIGHTING_OUTPUT" : 55, 
    "OBJECT_NETWORK_PORT" : 56,   
    "OBJECT_ELEVATOR_GROUP" : 57,   
    "OBJECT_ESCALATOR" : 58,   
    "OBJECT_LIFT" : 59,   
    "OBJECT_STAGING" : 60,  
}

property_ids ={
    "PROP_ACKED_TRANSITIONS" : 0,
    "PROP_ACK_REQUIRED" : 1,
    "PROP_ACTION" : 2,
    "PROP_ACTION_TEXT" : 3,
    "PROP_ACTIVE_TEXT" : 4,
    "PROP_ACTIVE_VT_SESSIONS" : 5,
    "PROP_ALARM_VALUE" : 6,
    "PROP_ALARM_VALUES" : 7,
    "PROP_ALL" : 8,
    "PROP_ALL_WRITES_SUCCESSFUL" : 9,
    "PROP_APDU_SEGMENT_TIMEOUT" : 10,
    "PROP_APDU_TIMEOUT" : 11,
    "PROP_APPLICATION_SOFTWARE_VERSION" : 12,
    "PROP_ARCHIVE" : 13,
    "PROP_BIAS" : 14,
    "PROP_CHANGE_OF_STATE_COUNT" : 15,
    "PROP_CHANGE_OF_STATE_TIME" : 16,
    "PROP_NOTIFICATION_CLASS" : 17,
    "PROP_BLANK_1" : 18,
    "PROP_CONTROLLED_VARIABLE_REFERENCE" : 19,
    "PROP_CONTROLLED_VARIABLE_UNITS" : 20,
    "PROP_CONTROLLED_VARIABLE_VALUE" : 21,
    "PROP_COV_INCREMENT" : 22,
    "PROP_DATE_LIST" : 23,
    "PROP_DAYLIGHT_SAVINGS_STATUS" : 24,
    "PROP_DEADBAND" : 25,
    "PROP_DERIVATIVE_CONSTANT" : 26,
    "PROP_DERIVATIVE_CONSTANT_UNITS" : 27,
    "PROP_DESCRIPTION" : 28,
    "PROP_DESCRIPTION_OF_HALT" : 29,
    "PROP_DEVICE_ADDRESS_BINDING" : 30,
    "PROP_DEVICE_TYPE" : 31,
    "PROP_EFFECTIVE_PERIOD" : 32,
    "PROP_ELAPSED_ACTIVE_TIME" : 33,
    "PROP_ERROR_LIMIT" : 34,
    "PROP_EVENT_ENABLE" : 35,
    "PROP_EVENT_STATE" : 36,
    "PROP_EVENT_TYPE" : 37,
    "PROP_EXCEPTION_SCHEDULE" : 38,
    "PROP_FAULT_VALUES" : 39,
    "PROP_FEEDBACK_VALUE" : 40,
    "PROP_FILE_ACCESS_METHOD" : 41,
    "PROP_FILE_SIZE" : 42,
    "PROP_FILE_TYPE" : 43,
    "PROP_FIRMWARE_REVISION" : 44,
    "PROP_HIGH_LIMIT" : 45,
    "PROP_INACTIVE_TEXT" : 46,
    "PROP_IN_PROCESS" : 47,
    "PROP_INSTANCE_OF" : 48,
    "PROP_INTEGRAL_CONSTANT" : 49,
    "PROP_INTEGRAL_CONSTANT_UNITS" : 50,
    "PROP_ISSUE_CONFIRMED_NOTIFICATIONS" : 51,
    "PROP_LIMIT_ENABLE" : 52,
    "PROP_LIST_OF_GROUP_MEMBERS" : 53,
    "PROP_LIST_OF_OBJECT_PROPERTY_REFERENCES" : 54,
    "PROP_LIST_OF_SESSION_KEYS" : 55,
    "PROP_LOCAL_DATE" : 56,
    "PROP_LOCAL_TIME" : 57,
    "PROP_LOCATION" : 58,
    "PROP_LOW_LIMIT" : 59,
    "PROP_MANIPULATED_VARIABLE_REFERENCE" : 60,
    "PROP_MAXIMUM_OUTPUT" : 61,
    "PROP_MAX_APDU_LENGTH_ACCEPTED" : 62,
    "PROP_MAX_INFO_FRAMES" : 63,
    "PROP_MAX_MASTER" : 64,
    "PROP_MAX_PRES_VALUE" : 65,
    "PROP_MINIMUM_OFF_TIME" : 66,
    "PROP_MINIMUM_ON_TIME" : 67,
    "PROP_MINIMUM_OUTPUT" : 68,
    "PROP_MIN_PRES_VALUE" : 69,
    "PROP_MODEL_NAME" : 70,
    "PROP_MODIFICATION_DATE" : 71,
    "PROP_NOTIFY_TYPE" : 72,
    "PROP_NUMBER_OF_APDU_RETRIES" : 73,
    "PROP_NUMBER_OF_STATES" : 74,
    "PROP_OBJECT_IDENTIFIER" : 75,
    "PROP_OBJECT_LIST" : 76,
    "PROP_OBJECT_NAME" : 77,
    "PROP_OBJECT_PROPERTY_REFERENCE" : 78,
    "PROP_OBJECT_TYPE" : 79,
    "PROP_OPTIONAL" : 80,
    "PROP_OUT_OF_SERVICE" : 81,
    "PROP_OUTPUT_UNITS" : 82,
    "PROP_EVENT_PARAMETERS" : 83,
    "PROP_POLARITY" : 84,
    "PROP_PRESENT_VALUE" : 85,
    "PROP_PRIORITY" : 86,
    "PROP_PRIORITY_ARRAY" : 87,
    "PROP_PRIORITY_FOR_WRITING" : 88,
    "PROP_PROCESS_IDENTIFIER" : 89,
    "PROP_PROGRAM_CHANGE" : 90,
    "PROP_PROGRAM_LOCATION" : 91,
    "PROP_PROGRAM_STATE" : 92,
    "PROP_PROPORTIONAL_CONSTANT" : 93,
    "PROP_PROPORTIONAL_CONSTANT_UNITS" : 94,
    "PROP_PROTOCOL_CONFORMANCE_CLASS" : 95,       
    "PROP_PROTOCOL_OBJECT_TYPES_SUPPORTED" : 96,
    "PROP_PROTOCOL_SERVICES_SUPPORTED" : 97,
    "PROP_PROTOCOL_VERSION" : 98,
    "PROP_READ_ONLY" : 99,
    "PROP_REASON_FOR_HALT" : 100,
    "PROP_RECIPIENT" : 101,
    "PROP_RECIPIENT_LIST" : 102,
    "PROP_RELIABILITY" : 103,
    "PROP_RELINQUISH_DEFAULT" : 104,
    "PROP_REQUIRED" : 105,
    "PROP_RESOLUTION" : 106,
    "PROP_SEGMENTATION_SUPPORTED" : 107,
    "PROP_SETPOINT" : 108,
    "PROP_SETPOINT_REFERENCE" : 109,
    "PROP_STATE_TEXT" : 110,
    "PROP_STATUS_FLAGS" : 111,
    "PROP_SYSTEM_STATUS" : 112,
    "PROP_TIME_DELAY" : 113,
    "PROP_TIME_OF_ACTIVE_TIME_RESET" : 114,
    "PROP_TIME_OF_STATE_COUNT_RESET" : 115,
    "PROP_TIME_SYNCHRONIZATION_RECIPIENTS" : 116,
    "PROP_UNITS" : 117,
    "PROP_UPDATE_INTERVAL" : 118,
    "PROP_UTC_OFFSET" : 119,
    "PROP_VENDOR_IDENTIFIER" : 120,
    "PROP_VENDOR_NAME" : 121,
    "PROP_VT_CLASSES_SUPPORTED" : 122,
    "PROP_WEEKLY_SCHEDULE" : 123,
    "PROP_ATTEMPTED_SAMPLES" : 124,
    "PROP_AVERAGE_VALUE" : 125,
    "PROP_BUFFER_SIZE" : 126,
    "PROP_CLIENT_COV_INCREMENT" : 127,
    "PROP_COV_RESUBSCRIPTION_INTERVAL" : 128,
    "PROP_CURRENT_NOTIFY_TIME" : 129,
    "PROP_EVENT_TIME_STAMPS" : 130,
    "PROP_LOG_BUFFER" : 131,
    "PROP_LOG_DEVICE_OBJECT_PROPERTY" : 132,
    "PROP_ENABLE" : 133,
    "PROP_LOG_INTERVAL" : 134,
    "PROP_MAXIMUM_VALUE" : 135,
    "PROP_MINIMUM_VALUE" : 136,
    "PROP_NOTIFICATION_THRESHOLD" : 137,
    "PROP_PREVIOUS_NOTIFY_TIME" : 138,
    "PROP_PROTOCOL_REVISION" : 139,
    "PROP_RECORDS_SINCE_NOTIFICATION" : 140,
    "PROP_RECORD_COUNT" : 141,
    "PROP_START_TIME" : 142,
    "PROP_STOP_TIME" : 143,
    "PROP_STOP_WHEN_FULL" : 144,
    "PROP_TOTAL_RECORD_COUNT" : 145,
    "PROP_VALID_SAMPLES" : 146,
    "PROP_WINDOW_INTERVAL" : 147,
    "PROP_WINDOW_SAMPLES" : 148,
    "PROP_MAXIMUM_VALUE_TIMESTAMP" : 149,
    "PROP_MINIMUM_VALUE_TIMESTAMP" : 150,
    "PROP_VARIANCE_VALUE" : 151,
    "PROP_ACTIVE_COV_SUBSCRIPTIONS" : 152,
    "PROP_BACKUP_FAILURE_TIMEOUT" : 153,
    "PROP_CONFIGURATION_FILES" : 154,
    "PROP_DATABASE_REVISION" : 155,
    "PROP_DIRECT_READING" : 156,
    "PROP_LAST_RESTORE_TIME" : 157,
    "PROP_MAINTENANCE_REQUIRED" : 158,
    "PROP_MEMBER_OF" : 159,
    "PROP_MODE" : 160,
    "PROP_OPERATION_EXPECTED" : 161,
    "PROP_SETTING" : 162,
    "PROP_SILENCED" : 163,
    "PROP_TRACKING_VALUE" : 164,
    "PROP_ZONE_MEMBERS" : 165,
    "PROP_LIFE_SAFETY_ALARM_VALUES" : 166,
    "PROP_MAX_SEGMENTS_ACCEPTED" : 167,
    "PROP_PROFILE_NAME" : 168,
    "PROP_AUTO_SLAVE_DISCOVERY" : 169,
    "PROP_MANUAL_SLAVE_ADDRESS_BINDING" : 170,
    "PROP_SLAVE_ADDRESS_BINDING" : 171,
    "PROP_SLAVE_PROXY_ENABLE" : 172,
    "PROP_LAST_NOTIFY_RECORD" : 173,
    "PROP_SCHEDULE_DEFAULT" : 174,
    "PROP_ACCEPTED_MODES" : 175,
    "PROP_ADJUST_VALUE" : 176,
    "PROP_COUNT" : 177,
    "PROP_COUNT_BEFORE_CHANGE" : 178,
    "PROP_COUNT_CHANGE_TIME" : 179,
    "PROP_COV_PERIOD" : 180,
    "PROP_INPUT_REFERENCE" : 181,
    "PROP_LIMIT_MONITORING_INTERVAL" : 182,
    "PROP_LOGGING_OBJECT" : 183,
    "PROP_LOGGING_RECORD" : 184,
    "PROP_PRESCALE" : 185,
    "PROP_PULSE_RATE" : 186,
    "PROP_SCALE" : 187,
    "PROP_SCALE_FACTOR" : 188,
    "PROP_UPDATE_TIME" : 189,
    "PROP_VALUE_BEFORE_CHANGE" : 190,
    "PROP_VALUE_SET" : 191,
    "PROP_VALUE_CHANGE_TIME" : 192,
    "PROP_ALIGN_INTERVALS" : 193,
    "PROP_INTERVAL_OFFSET" : 195,
    "PROP_LAST_RESTART_REASON" : 196,
    "PROP_LOGGING_TYPE" : 197,
    "PROP_RESTART_NOTIFICATION_RECIPIENTS" : 202,
    "PROP_TIME_OF_DEVICE_RESTART" : 203,
    "PROP_TIME_SYNCHRONIZATION_INTERVAL" : 204,
    "PROP_TRIGGER" : 205,
    "PROP_UTC_TIME_SYNCHRONIZATION_RECIPIENTS" : 206,
    "PROP_NODE_SUBTYPE" : 207,
    "PROP_NODE_TYPE" : 208,
    "PROP_STRUCTURED_OBJECT_LIST" : 209,
    "PROP_SUBORDINATE_ANNOTATIONS" : 210,
    "PROP_SUBORDINATE_LIST" : 211,
    "PROP_ACTUAL_SHED_LEVEL" : 212,
    "PROP_DUTY_WINDOW" : 213,
    "PROP_EXPECTED_SHED_LEVEL" : 214,
    "PROP_FULL_DUTY_BASELINE" : 215,

    "PROP_REQUESTED_SHED_LEVEL" : 218,
    "PROP_SHED_DURATION" : 219,
    "PROP_SHED_LEVEL_DESCRIPTIONS" : 220,
    "PROP_SHED_LEVELS" : 221,
    "PROP_STATE_DESCRIPTION" : 222,
    "PROP_DOOR_ALARM_STATE" : 226,
    "PROP_DOOR_EXTENDED_PULSE_TIME" : 227,
    "PROP_DOOR_MEMBERS" : 228,
    "PROP_DOOR_OPEN_TOO_LONG_TIME" : 229,
    "PROP_DOOR_PULSE_TIME" : 230,
    "PROP_DOOR_STATUS" : 231,
    "PROP_DOOR_UNLOCK_DELAY_TIME" : 232,
    "PROP_LOCK_STATUS" : 233,
    "PROP_MASKED_ALARM_VALUES" : 234,
    "PROP_SECURED_STATUS" : 235,
    "PROP_ABSENTEE_LIMIT" : 244,
    "PROP_ACCESS_ALARM_EVENTS" : 245,
    "PROP_ACCESS_DOORS" : 246,
    "PROP_ACCESS_EVENT" : 247,
    "PROP_ACCESS_EVENT_AUTHENTICATION_FACTOR" : 248,
    "PROP_ACCESS_EVENT_CREDENTIAL" : 249,
    "PROP_ACCESS_EVENT_TIME" : 250,
    "PROP_ACCESS_TRANSACTION_EVENTS" : 251,
    "PROP_ACCOMPANIMENT" : 252,
    "PROP_ACCOMPANIMENT_TIME" : 253,
    "PROP_ACTIVATION_TIME" : 254,
    "PROP_ACTIVE_AUTHENTICATION_POLICY" : 255,
    "PROP_ASSIGNED_ACCESS_RIGHTS" : 256,
    "PROP_AUTHENTICATION_FACTORS" : 257,
    "PROP_AUTHENTICATION_POLICY_LIST" : 258,
    "PROP_AUTHENTICATION_POLICY_NAMES" : 259,
    "PROP_AUTHENTICATION_STATUS" : 260,
    "PROP_AUTHORIZATION_MODE" : 261,
    "PROP_BELONGS_TO" : 262,
    "PROP_CREDENTIAL_DISABLE" : 263,
    "PROP_CREDENTIAL_STATUS" : 264,
    "PROP_CREDENTIALS" : 265,
    "PROP_CREDENTIALS_IN_ZONE" : 266,
    "PROP_DAYS_REMAINING" : 267,
    "PROP_ENTRY_POINTS" : 268,
    "PROP_EXIT_POINTS" : 269,
    "PROP_EXPIRATION_TIME" : 270,
    "PROP_EXTENDED_TIME_ENABLE" : 271,
    "PROP_FAILED_ATTEMPT_EVENTS" : 272,
    "PROP_FAILED_ATTEMPTS" : 273,
    "PROP_FAILED_ATTEMPTS_TIME" : 274,
    "PROP_LAST_ACCESS_EVENT" : 275,
    "PROP_LAST_ACCESS_POINT" : 276,
    "PROP_LAST_CREDENTIAL_ADDED" : 277,
    "PROP_LAST_CREDENTIAL_ADDED_TIME" : 278,
    "PROP_LAST_CREDENTIAL_REMOVED" : 279,
    "PROP_LAST_CREDENTIAL_REMOVED_TIME" : 280,
    "PROP_LAST_USE_TIME" : 281,
    "PROP_LOCKOUT" : 282,
    "PROP_LOCKOUT_RELINQUISH_TIME" : 283,
    "PROP_MASTER_EXEMPTION" : 284,
    "PROP_MAX_FAILED_ATTEMPTS" : 285,
    "PROP_MEMBERS" : 286,
    "PROP_MUSTER_POINT" : 287,
    "PROP_NEGATIVE_ACCESS_RULES" : 288,
    "PROP_NUMBER_OF_AUTHENTICATION_POLICIES" : 289,
    "PROP_OCCUPANCY_COUNT" : 290,
    "PROP_OCCUPANCY_COUNT_ADJUST" : 291,
    "PROP_OCCUPANCY_COUNT_ENABLE" : 292,
    "PROP_OCCUPANCY_EXEMPTION" : 293,
    "PROP_OCCUPANCY_LOWER_LIMIT" : 294,
    "PROP_OCCUPANCY_LOWER_LIMIT_ENFORCED" : 295,
    "PROP_OCCUPANCY_STATE" : 296,
    "PROP_OCCUPANCY_UPPER_LIMIT" : 297,
    "PROP_OCCUPANCY_UPPER_LIMIT_ENFORCED" : 298,
    "PROP_PASSBACK_EXEMPTION" : 299,
    "PROP_PASSBACK_MODE" : 300,
    "PROP_PASSBACK_TIMEOUT" : 301,
    "PROP_POSITIVE_ACCESS_RULES" : 302,
    "PROP_REASON_FOR_DISABLE" : 303,
    "PROP_SUPPORTED_FORMATS" : 304,
    "PROP_SUPPORTED_FORMAT_CLASSES" : 305,
    "PROP_THREAT_AUTHORITY" : 306,
    "PROP_THREAT_LEVEL" : 307,
    "PROP_TRACE_FLAG" : 308,
    "PROP_TRANSACTION_NOTIFICATION_CLASS" : 309,
    "PROP_USER_EXTERNAL_IDENTIFIER" : 310,
    "PROP_USER_INFORMATION_REFERENCE" : 311,
    "PROP_USER_NAME" : 317,
    "PROP_USER_TYPE" : 318,
    "PROP_USES_REMAINING" : 319,
    "PROP_ZONE_FROM" : 320,
    "PROP_ZONE_TO" : 321,
    "PROP_ACCESS_EVENT_TAG" : 322,
    "PROP_GLOBAL_IDENTIFIER" : 323,
    "PROP_VERIFICATION_TIME" : 326,
    "PROP_BASE_DEVICE_SECURITY_POLICY" : 327,
    "PROP_DISTRIBUTION_KEY_REVISION" : 328,
    "PROP_DO_NOT_HIDE" : 329,
    "PROP_KEY_SETS" : 330,
    "PROP_LAST_KEY_SERVER" : 331,
    "PROP_NETWORK_ACCESS_SECURITY_POLICIES" : 332,
    "PROP_PACKET_REORDER_TIME" : 333,
    "PROP_SECURITY_PDU_TIMEOUT" : 334,
    "PROP_SECURITY_TIME_WINDOW" : 335,
    "PROP_SUPPORTED_SECURITY_ALGORITHM" : 336,
    "PROP_UPDATE_KEY_SET_TIMEOUT" : 337,
    "PROP_BACKUP_AND_RESTORE_STATE" : 338,
    "PROP_BACKUP_PREPARATION_TIME" : 339,
    "PROP_RESTORE_COMPLETION_TIME" : 340,
    "PROP_RESTORE_PREPARATION_TIME" : 341,
    "PROP_BIT_MASK" : 342,
    "PROP_BIT_TEXT" : 343,
    "PROP_IS_UTC" : 344,
    "PROP_GROUP_MEMBERS" : 345,
    "PROP_GROUP_MEMBER_NAMES" : 346,
    "PROP_MEMBER_STATUS_FLAGS" : 347,
    "PROP_REQUESTED_UPDATE_INTERVAL" : 348,
    "PROP_COVU_PERIOD" : 349,
    "PROP_COVU_RECIPIENTS" : 350,
    "PROP_EVENT_MESSAGE_TEXTS" : 351,
    "PROP_EVENT_MESSAGE_TEXTS_CONFIG" : 352,
    "PROP_EVENT_DETECTION_ENABLE" : 353,
    "PROP_EVENT_ALGORITHM_INHIBIT" : 354,
    "PROP_EVENT_ALGORITHM_INHIBIT_REF" : 355,
    "PROP_TIME_DELAY_NORMAL" : 356,
    "PROP_RELIABILITY_EVALUATION_INHIBIT" : 357,
    "PROP_FAULT_PARAMETERS" : 358,
    "PROP_FAULT_TYPE" : 359,
    "PROP_LOCAL_FORWARDING_ONLY" : 360,
    "PROP_PROCESS_IDENTIFIER_FILTER" : 361,
    "PROP_SUBSCRIBED_RECIPIENTS" : 362,
    "PROP_PORT_FILTER" : 363,
    "PROP_AUTHORIZATION_EXEMPTIONS" : 364,
    "PROP_ALLOW_GROUP_DELAY_INHIBIT" : 365,
    "PROP_CHANNEL_NUMBER" : 366,
    "PROP_CONTROL_GROUPS" : 367,
    "PROP_EXECUTION_DELAY" : 368,
    "PROP_LAST_PRIORITY" : 369,
    "PROP_WRITE_STATUS" : 370,
    "PROP_PROPERTY_LIST" : 371,
    "PROP_SERIAL_NUMBER" : 372,
    "PROP_BLINK_WARN_ENABLE" : 373,
    "PROP_DEFAULT_FADE_TIME" : 374,
    "PROP_DEFAULT_RAMP_RATE" : 375,
    "PROP_DEFAULT_STEP_INCREMENT" : 376,
    "PROP_EGRESS_TIME" : 377,
    "PROP_IN_PROGRESS" : 378,
    "PROP_INSTANTANEOUS_POWER" : 379,
    "PROP_LIGHTING_COMMAND" : 380,
    "PROP_LIGHTING_COMMAND_DEFAULT_PRIORITY" : 381,
    "PROP_MAX_ACTUAL_VALUE" : 382,
    "PROP_MIN_ACTUAL_VALUE" : 383,
    "PROP_POWER" : 384,
    "PROP_TRANSITION" : 385,
    "PROP_EGRESS_ACTIVE" : 386,
    "PROP_INTERFACE_VALUE" : 387,
    "PROP_FAULT_HIGH_LIMIT" : 388,
    "PROP_FAULT_LOW_LIMIT" : 389,
    "PROP_LOW_DIFF_LIMIT" : 390,
    "PROP_STRIKE_COUNT" : 391,
    "PROP_TIME_OF_STRIKE_COUNT_RESET" : 392,
    "PROP_DEFAULT_TIMEOUT" : 393,
    "PROP_INITIAL_TIMEOUT" : 394,
    "PROP_LAST_STATE_CHANGE" : 395,
    "PROP_STATE_CHANGE_VALUES" : 396,
    "PROP_TIMER_RUNNING" : 397,
    "PROP_TIMER_STATE" : 398,
    "PROP_APDU_LENGTH" : 399,
    "PROP_IP_ADDRESS" : 400,
    "PROP_IP_DEFAULT_GATEWAY" : 401,
    "PROP_IP_DHCP_ENABLE" : 402,
    "PROP_IP_DHCP_LEASE_TIME" : 403,
    "PROP_IP_DHCP_LEASE_TIME_REMAINING" : 404,
    "PROP_IP_DHCP_SERVER" : 405,
    "PROP_IP_DNS_SERVER" : 406,
    "PROP_BACNET_IP_GLOBAL_ADDRESS" : 407,
    "PROP_BACNET_IP_MODE" : 408,
    "PROP_BACNET_IP_MULTICAST_ADDRESS" : 409,
    "PROP_BACNET_IP_NAT_TRAVERSAL" : 410,
    "PROP_IP_SUBNET_MASK" : 411,
    "PROP_BACNET_IP_UDP_PORT" : 412,
    "PROP_BBMD_ACCEPT_FD_REGISTRATIONS" : 413,
    "PROP_BBMD_BROADCAST_DISTRIBUTION_TABLE" : 414,
    "PROP_BBMD_FOREIGN_DEVICE_TABLE" : 415,
    "PROP_CHANGES_PENDING" : 416,
    "PROP_COMMAND" : 417,
    "PROP_FD_BBMD_ADDRESS" : 418,
    "PROP_FD_SUBSCRIPTION_LIFETIME" : 419,
    "PROP_LINK_SPEED" : 420,
    "PROP_LINK_SPEEDS" : 421,
    "PROP_LINK_SPEED_AUTONEGOTIATE" : 422,
    "PROP_MAC_ADDRESS" : 423,
    "PROP_NETWORK_INTERFACE_NAME" : 424,
    "PROP_NETWORK_NUMBER" : 425,
    "PROP_NETWORK_NUMBER_QUALITY" : 426,
    "PROP_NETWORK_TYPE" : 427,
    "PROP_ROUTING_TABLE" : 428,
    "PROP_VIRTUAL_MAC_ADDRESS_TABLE" : 429,
    "PROP_COMMAND_TIME_ARRAY" : 430,
    "PROP_CURRENT_COMMAND_PRIORITY" : 431,
    "PROP_LAST_COMMAND_TIME" : 432,
    "PROP_VALUE_SOURCE" : 433,
    "PROP_VALUE_SOURCE_ARRAY" : 434,
    "PROP_BACNET_IPV6_MODE" : 435,
    "PROP_IPV6_ADDRESS" : 436,
    "PROP_IPV6_PREFIX_LENGTH" : 437,
    "PROP_BACNET_IPV6_UDP_PORT" : 438,
    "PROP_IPV6_DEFAULT_GATEWAY" : 439,
    "PROP_BACNET_IPV6_MULTICAST_ADDRESS" : 440,
    "PROP_IPV6_DNS_SERVER" : 441,
    "PROP_IPV6_AUTO_ADDRESSING_ENABLE" : 442,
    "PROP_IPV6_DHCP_LEASE_TIME" : 443,
    "PROP_IPV6_DHCP_LEASE_TIME_REMAINING" : 444,
    "PROP_IPV6_DHCP_SERVER" : 445,
    "PROP_IPV6_ZONE_INDEX" : 446,
    "PROP_ASSIGNED_LANDING_CALLS" : 447,
    "PROP_CAR_ASSIGNED_DIRECTION" : 448,
    "PROP_CAR_DOOR_COMMAND" : 449,
    "PROP_CAR_DOOR_STATUS" : 450,
    "PROP_CAR_DOOR_TEXT" : 451,
    "PROP_CAR_DOOR_ZONE" : 452,
    "PROP_CAR_DRIVE_STATUS" : 453,
    "PROP_CAR_LOAD" : 454,
    "PROP_CAR_LOAD_UNITS" : 455,
    "PROP_CAR_MODE" : 456,
    "PROP_CAR_MOVING_DIRECTION" : 457,
    "PROP_CAR_POSITION" : 458,
    "PROP_ELEVATOR_GROUP" : 459,
    "PROP_ENERGY_METER" : 460,
    "PROP_ENERGY_METER_REF" : 461,
    "PROP_ESCALATOR_MODE" : 462,
    "PROP_FAULT_SIGNALS" : 463,
    "PROP_FLOOR_TEXT" : 464,
    "PROP_GROUP_ID" : 465,
    "PROP_GROUP_MODE" : 467,
    "PROP_HIGHER_DECK" : 468,
    "PROP_INSTALLATION_ID" : 469,
    "PROP_LANDING_CALLS" : 470,
    "PROP_LANDING_CALL_CONTROL" : 471,
    "PROP_LANDING_DOOR_STATUS" : 472,
    "PROP_LOWER_DECK" : 473,
    "PROP_MACHINE_ROOM_ID" : 474,
    "PROP_MAKING_CAR_CALL" : 475,
    "PROP_NEXT_STOPPING_FLOOR" : 476,
    "PROP_OPERATION_DIRECTION" : 477,
    "PROP_PASSENGER_ALARM" : 478,
    "PROP_POWER_MODE" : 479,
    "PROP_REGISTERED_CAR_CALL" : 480,
    "PROP_ACTIVE_COV_MULTIPLE_SUBSCRIPTIONS" : 481,
    "PROP_PROTOCOL_LEVEL" : 482,
    "PROP_REFERENCE_PORT" : 483,
    "PROP_DEPLOYED_PROFILE_LOCATION" : 484,
    "PROP_PROFILE_LOCATION" : 485,
    "PROP_TAGS" : 486,
    "PROP_SUBORDINATE_NODE_TYPES" : 487,
    "PROP_SUBORDINATE_TAGS" : 488,
    "PROP_SUBORDINATE_RELATIONSHIPS" : 489,
    "PROP_DEFAULT_SUBORDINATE_RELATIONSHIP" : 490,
    "PROP_REPRESENTS" : 491,
    "PROP_DEFAULT_PRESENT_VALUE" : 492,
    "PROP_PRESENT_STAGE" : 493,
    "PROP_STAGES" : 494,
    "PROP_STAGE_NAMES" : 495,
}

if __name__ == '__main__':
    subcommands = [
        ErrorsCommand,
        BackupCommand,
        RestoreCommand,
        FindCommand,
        VersionCommand,
        ListObjectsCommand,
        DeleteObjectCommand,
        CreateObjectCommand,
        UpdateObjectCommand,
        GetModbusProfiles,
        CreateModbusProfile,
        GetModbusConnections,
        DeleteModbusConnection,
        CreateModbusConnection,
        WatchDataCommand,
    ]
    parser = argparse.ArgumentParser("Normal Framework CLI")
    parser.add_argument("command", metavar="command",
                        help="subcommand: [{}]".format('|'.join((
                            c.name for i, c in enumerate(subcommands)))))
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
