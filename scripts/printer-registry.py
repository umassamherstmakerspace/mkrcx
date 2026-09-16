#!/usr/bin/env python3
"""Read and edit staging printer records without a repository change.

Set LEASH_PRINTER_TOKEN to an authorized session/API token. Tokens are never
accepted on the command line or written to output. Saves require a reviewed
JSON record and its current version; the server records actor and history.
"""
import argparse
import json
import os
import sys
import urllib.error
import urllib.request
from pathlib import Path

BASE = "https://leash.staging.mkr.cx/api/printer-fleet/records"
FIELDS = ("name", "model", "machineId", "location", "lifecycle", "maintenance", "host", "mac", "condition", "note", "manual", "version")


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *_args, **_kwargs):
        raise RuntimeError("Unexpected redirect; refusing to forward credentials")


def request(path="", body=None):
    token = os.environ.get("LEASH_PRINTER_TOKEN")
    if not token:
        raise RuntimeError("Set LEASH_PRINTER_TOKEN to an authorized staging credential")
    req = urllib.request.Request(BASE + path, method="GET" if body is None else "PUT",
        data=None if body is None else json.dumps(body).encode(),
        headers={"Authorization": "Bearer " + token, "Content-Type": "application/json"})
    try:
        with urllib.request.build_opener(NoRedirect).open(req, timeout=15) as response:
            return json.load(response)
    except urllib.error.HTTPError as error:
        raise RuntimeError(f"Registry returned HTTP {error.code}; for 409, reload and review the current version") from None


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="action", required=True)
    sub.add_parser("list")
    get = sub.add_parser("get"); get.add_argument("id")
    save = sub.add_parser("save"); save.add_argument("file", type=Path)
    history = sub.add_parser("history"); history.add_argument("id")
    args = parser.parse_args()
    if args.action == "list":
        result = request()
    elif args.action == "get":
        result = next((r for r in request() if r["id"] == args.id), None)
        if result is None: raise RuntimeError("Printer not found")
    elif args.action == "history":
        import re
        if not re.fullmatch(r"[a-z0-9][a-z0-9-]{0,79}", args.id): raise RuntimeError("Invalid printer ID")
        result = request("/" + args.id + "/history")
    else:
        import re
        record = json.loads(args.file.read_text(encoding="utf-8-sig"))
        if not re.fullmatch(r"[a-z0-9][a-z0-9-]{0,79}", record.get("id", "")): raise RuntimeError("Invalid printer ID")
        body = {field: record[field] for field in FIELDS}
        result = request("/" + record["id"], body)
        if result.get("version") != body["version"] + 1 or any(result.get(k) != body[k] for k in ("condition", "note", "manual", "lifecycle", "maintenance", "location")):
            raise RuntimeError("Saved response did not match; read back before retrying")
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    try: main()
    except (RuntimeError, ValueError, KeyError, OSError) as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
