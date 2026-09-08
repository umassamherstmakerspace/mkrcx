#!/usr/bin/env python3
"""Staging-only roster-driven observer. Install separately from the production collector."""
import importlib.util
import ipaddress
import json
import os
import re
import urllib.request
from pathlib import Path

SPEC = importlib.util.spec_from_file_location("fleet_observer", Path(__file__).with_name("printer-fleet-collector.py"))
OBSERVER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(OBSERVER)
ROSTER = "https://leash.staging.mkr.cx/api/printer-fleet/roster"
INGEST = "https://leash.staging.mkr.cx/api/printer-fleet/ingest"


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *_args, **_kwargs):
        raise RuntimeError("Unexpected collector redirect")


def validate_roster(value):
    printers = value.get("printers") if isinstance(value, dict) else None
    if not isinstance(printers, list) or len(printers) > 200:
        raise ValueError("Invalid roster")
    ids, hosts, macs, result = set(), set(), set(), []
    for p in printers:
        identity, host, mac = p["id"], p["host"], p["mac"]
        if not re.fullmatch(r"[a-z0-9][a-z0-9-]{0,79}", identity) or identity in ids or host in hosts or mac in macs:
            raise ValueError("Duplicate or invalid identity")
        # This collector has only the existing MKR3DP subnet as its observation scope.
        ip = ipaddress.ip_address(host)
        if ip not in ipaddress.ip_network("192.168.1.0/24") or int(ip) % 256 in (0, 255):
            raise ValueError("Target outside printer subnet")
        if not re.fullmatch(r"(?:[0-9a-f]{2}:){5}[0-9a-f]{2}", mac) or int(mac[:2], 16) & 1:
            raise ValueError("Invalid hardware identity")
        ids.add(identity); hosts.add(host); macs.add(mac)
        result.append((identity, host, mac))
    return result


def load_roster(opener, secret, cache):
    req = urllib.request.Request(ROSTER, headers={"Authorization": "Bearer " + secret})
    try:
        with opener.open(req, timeout=10) as response:
            value = json.load(response)
        roster = validate_roster(value)
    except (OSError, TimeoutError):
        # A malformed server response is not silently accepted through the cache.
        return validate_roster(json.loads(cache.read_text()))
    temporary = cache.with_suffix(".new")
    temporary.write_text(json.dumps(value))
    os.replace(temporary, cache)
    return roster


def collect(roster):
    try:
        runtime, active = OBSERVER.read_runtime()
    except Exception:
        # The station DB being unavailable must not stop read-only activity observations.
        runtime, active = {}, {}
    with OBSERVER.concurrent.futures.ThreadPoolExecutor(max_workers=8) as executor:
        readings = list(executor.map(lambda entry: OBSERVER.read_printer(entry, runtime, active), roster))
    return {"fetchedAt": OBSERVER.iso_now(), "printers": readings}


def main():
    secret = os.environ["PRINTER_FLEET_INGEST_SECRET"]
    cache = Path(os.environ.get("PRINTER_REGISTRY_CACHE", "/var/lib/makerspace-printer-registry-staging/roster.json"))
    opener = urllib.request.build_opener(NoRedirect)
    snapshot = collect(load_roster(opener, secret, cache))
    req = urllib.request.Request(INGEST, data=json.dumps(snapshot).encode(), method="POST",
        headers={"Authorization": "Bearer " + secret, "Content-Type": "application/json"})
    with opener.open(req, timeout=15) as response:
        if response.status != 204: raise RuntimeError("Staging ingest failed")
    print(f"Staging printer registry: {len(snapshot['printers'])} observations")


if __name__ == "__main__":
    try: main()
    except Exception as error:
        print("Staging collector failed: " + type(error).__name__)
        raise SystemExit(1)
