#!/usr/bin/env python3
"""Staging-only roster-driven observer. Install separately from the production collector."""
import importlib.util
import ipaddress
import json
import os
import re
import sqlite3
import urllib.parse
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
    # The saved station condition remains meaningful even when its machine is off.
    for reading in readings:
        row = runtime.get(reading['id'], {})
        condition = {'available':'working','needs_attention':'limited','out_of_service':'out'}.get(row.get('condition'))
        if condition and (row.get('reported_at') or row.get('problem_note') or row.get('system_status') == 'available'):
            reading['condition'] = condition
            reading['note'] = row.get('problem_note') or ''
    snapshot = {"fetchedAt": OBSERVER.iso_now(), "printers": readings}
    try:
        snapshot['history'] = read_history([entry[0] for entry in roster])
    except (sqlite3.Error, OSError, ValueError):
        # A missing history source must not wipe stored history or stop status updates.
        print('Staging history source unavailable; retaining previously collected records')
    return snapshot


def clean_text(value, limit):
    if not isinstance(value, str): return ''
    return ''.join(c for c in value if ord(c) >= 32 or c in '\n\t')[:limit]


def read_history(ids, path=None, *, job_limit=30, condition_limit=20, include_identity=None):
    if include_identity is None:
        include_identity = os.environ.get("PRINTER_HISTORY_IDENTITY_ENABLED") == "true"
    # Normal polling stays small; a one-time backfill can read a larger retained window.
    if any(type(limit) is not int or not 1 <= limit <= 1000 for limit in (job_limit, condition_limit)):
        raise ValueError('History limits must be integers from 1 to 1000')
    path = path or os.environ.get('PRINTER_STATION_DB', '/var/lib/makerspace-print-station/station.sqlite')
    db = sqlite3.connect(f'file:{urllib.parse.quote(str(path))}?mode=ro', uri=True)
    db.row_factory = sqlite3.Row
    db.execute('PRAGMA query_only=ON')
    result = []
    try:
        for printer_id in ids:
            jobs = db.execute('''SELECT e.id,e.event_type,e.created_at,e.payload_json,r.file_name,r.filament_type,r.user_display
                FROM events e JOIN requests r ON r.id=e.request_id
                WHERE r.printer_id=? AND e.event_type IN ('printer_completed','printer_cancelled','printer_failed')
                ORDER BY e.id DESC LIMIT ?''', (printer_id, job_limit)).fetchall()
            changes = db.execute('''SELECT id,event_type,created_at,payload_json FROM printer_runtime_events
                WHERE printer_id=? AND event_type IN ('staff_runtime_changed','printer_runtime_changed','system_runtime_changed')
                ORDER BY id DESC LIMIT ?''', (printer_id, condition_limit)).fetchall()
            for source, rows in [('job', jobs), ('condition', changes)]:
                for row in rows:
                    payload = json.loads(row['payload_json'])
                    if not isinstance(payload, dict): continue
                    details = []
                    if source == 'condition':
                        for key, label in [('newCondition','Condition'),('newNote','Note')]:
                            if key in payload: details.append(label + ': ' + clean_text(payload[key], 2000))
                    else:
                        for key in ('message','error','detail'):
                            value = clean_text(payload.get(key), 2000)
                            if value: details.append(value)
                        if row['event_type'] == 'printer_failed' and not any(clean_text(payload.get(k),2000) for k in ('message','error','detail')):
                            details.append('The station recorded a failed print without an error message.')
                    event = {'sourceId':f'station:{source}:{row["id"]}', 'printerId':printer_id,
                        'recordedAt':row['created_at'],'eventType':row['event_type'],'detail':clean_text('\n'.join(details),4000)}
                    if source == 'condition':
                        for field in ('Note', 'Condition'):
                            before, after = payload.get('old' + field), payload.get('new' + field)
                            if isinstance(before, str) and isinstance(after, str):
                                event[field.lower() + 'Changed'] = before != after
                                if field == 'Condition': event['previousCondition'] = clean_text(before, 80)
                        method = payload.get('method')
                        if include_identity and method in ('ucard', 'local_pin', 'printer_api'):
                            event['actorMethod'] = method
                            if method == 'ucard':
                                event['actorName'] = clean_text(payload.get('displayIdentity'),200)
                    if source == 'job':
                        event.update(file=clean_text(row['file_name'],1000),material=clean_text(row['filament_type'],120))
                        if include_identity: event['person'] = clean_text(row['user_display'],200)
                        duration = payload.get('printDurationSeconds')
                        if isinstance(duration, (int,float)) and not isinstance(duration,bool) and 0 <= duration <= 31536000:
                            event['durationSeconds'] = duration
                    result.append(event)
    finally:
        db.close()
    return sorted(result, key=lambda event: event['recordedAt'], reverse=True)[:1000]


def main():
    secret = os.environ["PRINTER_FLEET_INGEST_SECRET"]
    cache = Path(os.environ.get("PRINTER_REGISTRY_CACHE", "/var/lib/makerspace-printer-registry-staging/roster.json"))
    opener = urllib.request.build_opener(NoRedirect)
    snapshot = collect(load_roster(opener, secret, cache))
    req = urllib.request.Request(INGEST, data=json.dumps(snapshot).encode(), method="POST",
        headers={"Authorization": "Bearer " + secret, "Content-Type": "application/json"})
    with opener.open(req, timeout=15) as response:
        if response.status != 204: raise RuntimeError("Staging ingest failed")
    history_count = len(snapshot['history']) if 'history' in snapshot else 'unavailable'
    print(f"Staging printer registry: {len(snapshot['printers'])} observations; history events: {history_count}")


if __name__ == "__main__":
    try: main()
    except Exception as error:
        print("Staging collector failed: " + type(error).__name__)
        raise SystemExit(1)
