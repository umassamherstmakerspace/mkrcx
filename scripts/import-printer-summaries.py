#!/usr/bin/env python3
"""Import reviewed printer summaries through the staff API.

Input: JSON list of {printerId, sourceId, reportDate, body}, optionally sources:[{url,label}].
Use a stable standup:<printer>:<summary> ID, independent of any message or link.
Summaries may combine several reports; links are supplementary. Identical retries are safe; corrections
need a new ID. Dates are report dates, never inferred repair-completion times.
The configured API key must have leash.printers:manage on an authorized account.
"""
import argparse
import json
import os
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *_args, **_kwargs):
        raise RuntimeError("Refusing to redirect a staff API request")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("file", type=Path)
    parser.add_argument("--endpoint", required=True)
    parser.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    endpoint = urllib.parse.urlparse(args.endpoint)
    if endpoint.scheme != "https" or not endpoint.hostname or endpoint.username or endpoint.query or endpoint.fragment:
        raise ValueError("An explicit HTTPS backend endpoint is required")
    rows = json.loads(args.file.read_text(encoding="utf-8-sig"))
    if not isinstance(rows, list) or not 1 <= len(rows) <= 100:
        raise ValueError("Expected 1–100 reviewed summaries")
    print(json.dumps({"destination": args.endpoint, "summaries": len(rows), "apply": args.apply}))
    if not args.apply:
        return
    secret = os.environ["PRINTER_STAFF_API_KEY"]
    opener = urllib.request.build_opener(NoRedirect)
    for source in rows:
        row = dict(source)
        printer = row.pop("printerId")
        url = args.endpoint.rstrip("/") + "/api/printer-fleet/history/" + urllib.parse.quote(printer, safe="") + "/summaries"
        req = urllib.request.Request(url, data=json.dumps(row).encode(), headers={"Content-Type":"application/json", "Authorization":"API-Key " + secret}, method="POST")
        with opener.open(req, timeout=20) as response:
            saved = json.load(response)
        if saved["sourceId"] != row["sourceId"] or saved["printerId"] != printer or saved["body"] != row["body"]:
            raise RuntimeError("Summary readback differs from the submitted record")
        print(json.dumps({"printerId":printer,"sourceId":saved["sourceId"],"saved":True}))


if __name__ == "__main__":
    try:
        main()
    except urllib.error.HTTPError as error:
        raise SystemExit(f"Summary import failed: HTTP {error.code}") from None
