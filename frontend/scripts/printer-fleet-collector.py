#!/usr/bin/env python3
"""Read the MKR3DP fleet with HTTP GETs and push one bounded dashboard snapshot."""
import concurrent.futures
import datetime
import json
import math
import os
import sqlite3
import sys
import urllib.parse
import urllib.request

FLEET = [
    ("k1-30eb", "192.168.1.159", "fc:ee:28:00:30:eb"),
    ("k1-32b8", "192.168.1.206", "fc:ee:28:00:32:b8"),
    ("k1-69ba", "192.168.1.203", "fc:ee:28:00:69:ba"),
    ("k1-791a", "192.168.1.227", "fc:ee:28:00:79:1a"),
    ("k1c-1c58", "192.168.1.187", "fc:ee:28:0b:1c:58"),
    ("k1c-1d56", "192.168.1.234", "fc:ee:28:0b:1d:56"),
    ("k1c-1f44", "192.168.1.164", "fc:ee:28:0b:1f:44"),
    ("k1c-1f65", "192.168.1.197", "fc:ee:28:0b:1f:65"),
    ("k1c-1f94", "192.168.1.244", "fc:ee:28:0b:1f:94"),
    ("k1max-2a33", "192.168.1.236", "fc:ee:28:01:2a:33"),
    ("k1max-d101", "192.168.1.149", "fc:ee:28:07:d1:01"),
    ("k1max-d103", "192.168.1.151", "fc:ee:28:07:d1:03"),
    ("k1max-d949", "192.168.1.163", "fc:ee:28:07:d9:49"),
    ("k1max-d973", "192.168.1.205", "fc:ee:28:07:d9:73"),
    ("k1max-fb47", "192.168.1.147", "fc:ee:28:01:fb:47"),
]
ALL_IDS = [item[0] for item in FLEET] + ["fdm-k1max-anderson"]


def iso_now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat().replace("+00:00", "Z")


def get_json(host, route):
    url = f"http://{host}:7125{route}"
    with urllib.request.urlopen(urllib.request.Request(url, method="GET"), timeout=1.8) as response:
        return json.load(response)


def read_runtime():
    path = os.environ.get("PRINTER_STATION_DB", "/var/lib/makerspace-print-station/station.sqlite")
    database = sqlite3.connect(f"file:{urllib.parse.quote(path)}?mode=ro", uri=True)
    database.row_factory = sqlite3.Row
    database.execute("PRAGMA query_only = ON")
    runtime = {row["printer_id"]: dict(row) for row in database.execute(
        "SELECT printer_id, condition, problem_note, reported_at, system_status, system_note FROM printer_runtime"
    )}
    requests = [dict(row) for row in database.execute(
        """SELECT printer_id, file_path, file_name, user_display, filament_type, started_at
             FROM requests WHERE state = 'started' ORDER BY started_at DESC"""
    )]
    database.close()
    active = {}
    for row in requests:
        active.setdefault((row["printer_id"], row["file_path"]), row)
    return runtime, active


def condition_fields(row):
    if not row or row.get("system_status") != "available":
        return "unknown", (row or {}).get("system_note") or ""
    condition = {"available": "working", "needs_attention": "limited", "out_of_service": "out"}.get(row.get("condition"), "unknown")
    return condition, row.get("problem_note") or ""


def read_printer(entry, runtime, active):
    printer_id, host, expected_mac = entry
    condition, note = condition_fields(runtime.get(printer_id))
    base = {"id": printer_id, "condition": condition, "activity": "unknown"}
    if note:
        base["note"] = note
    try:
        system = get_json(host, "/machine/system_info").get("result", {}).get("system_info", {})
        observed_macs = {str(value.get("mac_address", "")).lower() for value in system.get("network", {}).values()}
        if expected_mac not in observed_macs:
            raise ValueError("printer identity mismatch")
        stats = get_json(host, "/printer/objects/query?print_stats&virtual_sdcard").get("result", {}).get("status", {})
        print_stats = stats.get("print_stats", {})
        state = print_stats.get("state")
        base["activity"] = "paused" if state == "paused" else "printing" if state == "printing" else "idle"
        if base["activity"] in ("printing", "paused") and print_stats.get("filename"):
            filename = print_stats["filename"]
            metadata = get_json(host, "/server/files/metadata?filename=" + urllib.parse.quote(filename, safe="")).get("result", {})
            estimate = metadata.get("estimated_time")
            elapsed = print_stats.get("print_duration")
            if isinstance(estimate, (int, float)) and isinstance(elapsed, (int, float)):
                base["minutes"] = max(0, math.ceil((estimate - elapsed) / 60))
            progress = stats.get("virtual_sdcard", {}).get("progress")
            if isinstance(progress, (int, float)):
                base["progress"] = round(max(0, min(1, progress)) * 100)
            request = active.get((printer_id, filename))
            material = (request or {}).get("filament_type") or metadata.get("filament_type") or "Unknown"
            if isinstance(material, list):
                material = material[0] if material else "Unknown"
            base["job"] = {
                "person": (request or {}).get("user_display") or "Unknown / unassigned",
                "file": (request or {}).get("file_name") or filename.rsplit("/", 1)[-1],
                "material": str(material),
                "started": (request or {}).get("started_at") or "Unknown",
            }
        return base
    except Exception:
        return {"id": printer_id, "condition": "unknown", "activity": "unknown", "note": "Printer status unavailable."}


def collect():
    runtime, active = read_runtime()
    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as executor:
        readings = list(executor.map(lambda entry: read_printer(entry, runtime, active), FLEET))
    readings.append({"id": "fdm-k1max-anderson", "condition": "unknown", "activity": "unknown", "note": "Printer status unavailable."})
    return {"fetchedAt": iso_now(), "printers": readings}


def publish_target(snapshot, endpoint, secret):
    body = json.dumps(snapshot, separators=(",", ":")).encode()
    request = urllib.request.Request(endpoint, data=body, method="POST", headers={"Authorization": "Bearer " + secret, "Content-Type": "application/json", "User-Agent": "Makerspace-printer-fleet-collector/1"})
    with urllib.request.urlopen(request, timeout=12) as response:
        if response.status != 204:
            raise RuntimeError(f"collector publish returned HTTP {response.status}")


def publish(snapshot):
    targets = [("primary", os.environ["PRINTER_FLEET_ENDPOINT"], os.environ["PRINTER_FLEET_INGEST_SECRET"])]
    secondary_endpoint = os.environ.get("PRINTER_FLEET_SECONDARY_ENDPOINT")
    secondary_secret = os.environ.get("PRINTER_FLEET_SECONDARY_INGEST_SECRET")
    if bool(secondary_endpoint) != bool(secondary_secret):
        raise RuntimeError("secondary collector target is incomplete")
    if secondary_endpoint and secondary_secret:
        targets.append(("secondary", secondary_endpoint, secondary_secret))

    failures = []
    for name, endpoint, secret in targets:
        try:
            publish_target(snapshot, endpoint, secret)
        except Exception as error:
            failures.append(f"{name}:{type(error).__name__}")
    if failures:
        raise RuntimeError("collector publish failed for " + ",".join(failures))
    return len(targets)


if __name__ == "__main__":
    try:
        snapshot = collect()
        if "--validate-only" in sys.argv:
            available = sum(item["activity"] != "unknown" for item in snapshot["printers"])
            jobs = sum("job" in item for item in snapshot["printers"])
            print(f"validated printer fleet snapshot: {available}/{len(ALL_IDS)} activity readings, {jobs} active jobs")
            raise SystemExit(0)
        target_count = publish(snapshot)
        available = sum(item["activity"] != "unknown" for item in snapshot["printers"])
        print(f"published printer fleet snapshot to {target_count} dashboards: {available}/{len(ALL_IDS)} activity readings")
    except Exception as error:
        print(f"printer fleet collector failed: {type(error).__name__}", file=sys.stderr)
        raise SystemExit(1)
