#!/usr/bin/env python3
"""
snappy_normalframework_decoder_print.py

Decodes the snappy-stream file (20B header + payload), parses payloads with the installed
normalframework protobuf package, and optionally dumps the full proto as JSON.

Usage examples:
  # pretty JSON per message (multi-line, readable)
  python snappy_normalframework_decoder_print.py -f data.snappy --proto-module normalframework_nf_protocolbuffers_python.my_pb2 --message MyEvent --dump-json

  # NDJSON (1 JSON message per line, good for downstream tools)
  python snappy_normalframework_decoder_print.py -f data.snappy --proto-module ... --message ... --ndjson --out out.ndjson

  # Raw protobuf text format:
  python snappy_normalframework_decoder_print.py -f data.snappy --proto-module ... --message ... --text
"""

import argparse
import struct
import snappy
import importlib
import inspect
import pkgutil
from google.protobuf.json_format import MessageToJson
from google.protobuf.message import Message as PBMessage

HEADER_SIZE = 8 + 8 + 4  # 20

# --- snappy stream reader (same as earlier) ---
def decode_snappy_stream_file(path: str, chunk_size: int = 64 * 1024):
    decompressor = snappy.StreamDecompressor()
    buffer = bytearray()
    with open(path, "rb") as fh:
        while True:
            chunk = fh.read(chunk_size)
            if not chunk:
                break
            decompressed = decompressor.decompress(chunk)
            if decompressed:
                buffer.extend(decompressed)
            while True:
                if len(buffer) < HEADER_SIZE:
                    break
                header = bytes(buffer[:HEADER_SIZE])
                timestamp_raw, sequence, length = struct.unpack(">QQI", header)
                total_needed = HEADER_SIZE + length
                if len(buffer) < total_needed:
                    break
                payload = bytes(buffer[HEADER_SIZE:total_needed])
                del buffer[:total_needed]
                yield {"timestamp_raw": timestamp_raw, "sequence": sequence, "length": length, "data": payload}
        # flush
        try:
            tail = decompressor.decompress(b"")
        except Exception:
            tail = b""
        if tail:
            buffer.extend(tail)
        while True:
            if len(buffer) < HEADER_SIZE:
                break
            header = bytes(buffer[:HEADER_SIZE])
            timestamp_raw, sequence, length = struct.unpack(">QQI", header)
            total_needed = HEADER_SIZE + length
            if len(buffer) < total_needed:
                raise EOFError("truncated record at end of file")
            payload = bytes(buffer[HEADER_SIZE:total_needed])
            del buffer[:total_needed]
            yield {"timestamp_raw": timestamp_raw, "sequence": sequence, "length": length, "data": payload}

# --- small helper to find message classes in module (used when scanning) ---
def find_message_classes_in_module(mod):
    out = {}
    for attr_name in dir(mod):
        try:
            attr = getattr(mod, attr_name)
        except Exception:
            continue
        if inspect.isclass(attr) and issubclass(attr, PBMessage):
            out[f"{mod.__name__}.{attr_name}"] = attr
    if hasattr(mod, "__path__"):
        for finder, subname, ispkg in pkgutil.walk_packages(mod.__path__, prefix=mod.__name__ + "."):
            try:
                sm = importlib.import_module(subname)
            except Exception:
                continue
            for attr_name in dir(sm):
                try:
                    attr = getattr(sm, attr_name)
                except Exception:
                    continue
                if inspect.isclass(attr) and issubclass(attr, PBMessage):
                    out[f"{sm.__name__}.{attr_name}"] = attr
    return out

def locate_message_class(proto_module: str = None, message_name: str = None):
    # try explicit module first
    if proto_module:
        try:
            mod = importlib.import_module(proto_module)
        except Exception as e:
            raise ImportError(f"failed to import specified proto module '{proto_module}': {e!r}")
        if message_name:
            cls_name = message_name.split(".")[-1]
            if hasattr(mod, cls_name):
                cls = getattr(mod, cls_name)
                if inspect.isclass(cls) and issubclass(cls, PBMessage):
                    return f"{mod.__name__}.{cls_name}", cls
                else:
                    raise TypeError(f"{proto_module}.{cls_name} exists but is not a protobuf Message class")
        msgs = find_message_classes_in_module(mod)
        if msgs:
            return next(iter(msgs.items()))
    # try common candidate package roots derived from the pip package name
    candidate_base_names = [
        "normalframework_nf_protocolbuffers_python",
        "normalframework_nf_protocolbuffers",
        "normalframework.nf_protocolbuffers_python",
        "normalframework.nf_protocolbuffers",
        "nf_protocolbuffers",
        "normalframework",
    ]
    for base in candidate_base_names:
        try:
            mod = importlib.import_module(base)
        except Exception:
            continue
        msgs = find_message_classes_in_module(mod)
        if msgs:
            if message_name:
                for qn, cls in msgs.items():
                    if qn.endswith("." + message_name) or qn.endswith("." + message_name.split(".")[-1]):
                        return qn, cls
            return next(iter(msgs.items()))
    raise ImportError("Could not locate a protobuf Message class. Please pass --proto-module and --message explicitly.")

# --- main decode + print loop ---
def decode_and_print(path: str, proto_module: str = None, message_name: str = None,
                     dump_json: bool = False, ndjson: bool = False, text: bool = False,
                     out_path: str = None, max_records: int = None):
    qn, msg_cls = locate_message_class(proto_module, message_name)
    print(f"[info] using message class: {qn}")

    out_fh = None
    if out_path:
        out_fh = open(out_path, "w", encoding="utf-8")

    count = 0
    for rec in decode_snappy_stream_file(path):
        payload = rec["data"]
        try:
            msg = msg_cls()
            msg.ParseFromString(payload)
        except Exception as e:
            print(f"[warn] parse failed for seq={rec['sequence']} len={rec['length']}: {e}")
            continue

        # printing options
        if text:
            s = str(msg)  # protobuf text format
            if out_fh:
                out_fh.write(s + "\n\n")
            else:
                print(s)
        elif dump_json:
            # pretty-printed JSON
            s = MessageToJson(msg, preserving_proto_field_name=True, indent=2)
            if out_fh:
                out_fh.write(s + "\n")
            else:
                print(s)
        elif ndjson:
            # compact single-line JSON for each message
            s = MessageToJson(msg, preserving_proto_field_name=True, indent=None)
            if out_fh:
                out_fh.write(s + "\n")
            else:
                print(s)
        else:
            # default short summary
            print(f"seq={rec['sequence']} ts={rec['timestamp_raw']} len={rec['length']}")

        count += 1
        if max_records and count >= max_records:
            break

    if out_fh:
        out_fh.close()

# --- CLI ---
if __name__ == "__main__":
    p = argparse.ArgumentParser(description="Decode snappy-stream and print parsed protobuf messages.")
    p.add_argument("-f", "--file", required=True, help="path to snappy-compressed file")
    p.add_argument("--proto-module", help="exact generated pb2 module, e.g. mypkg.messages_pb2")
    p.add_argument("--message", help="message class name, e.g. RealtimeEvent")
    g = p.add_mutually_exclusive_group()
    g.add_argument("--dump-json", action="store_true", help="print pretty JSON for each message")
    g.add_argument("--ndjson", action="store_true", help="print one JSON object per line (NDJSON)")
    g.add_argument("--text", action="store_true", help="print protobuf text format (print(msg))")
    p.add_argument("--out", help="write output to file instead of stdout")
    p.add_argument("--max", type=int, help="stop after N records (useful for testing)")
    args = p.parse_args()

    decode_and_print(args.file, proto_module=args.proto_module, message_name=args.message,
                     dump_json=args.dump_json, ndjson=args.ndjson, text=args.text,
                     out_path=args.out, max_records=args.max)
