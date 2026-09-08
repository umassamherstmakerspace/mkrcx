# Persistent printer registry — staging rollout

Updated: 2026-09-08

## Scope and current boundary

Shira authorized building and deploying to staging first. Production deployment and changes to
live printer gating are outside this release. Staff editing is optional; management from an
authorized operator's tooling is the primary requirement.

Implementation lives on `codex/printer-registry-staging`, based on `d6a4532` of `mkrcx` main.
The local backend and frontend builds pass, along with 96 frontend unit tests, the complete Go
suite, and nine Python collector tests. Staging is **not yet deployed**: SSH to
`maker@spence.infra.mkr.cx` is refused before authentication with `Not allowed at this time`.
The GitHub build and immutable image receipts will be recorded here when available.

## Behavior

- MariaDB stores physical printer records, current saved condition/note, station observations,
  last-seen time, edit versions, and an append-only record edit history.
- The initial 16 identities are a migration seed, not a runtime allowlist. A restart never resets
  an edited or retired record. New records appear without a frontend or collector code change.
- Live readings cannot overwrite a manually maintained note. Offline/stale readings preserve
  the last known condition and explanation but remove live activity/estimates and job details.
- Testing and repair are lineup states separate from condition. Retired records remain in
  management/history but are omitted from the public list. Testing/repair printers are excluded
  from the available-idle filter.
- IDs follow physical machines. MAC identity cannot be reassigned. Clear an old printer's host
  to release its address; keep its MAC and repair record, then register the replacement separately.
- A record may have no connection information, or a known MAC with no address. Connection targets
  must be within `192.168.1.0/24`; the observer verifies MAC before reading activity.
- Notes are public. Hardware addresses, record actors, history, current file/user/material and
  progress are not in the public response. Protected live details expire after 90 seconds.
- A manual dashboard edit does **not** update the station database or local gating. The editor
  explicitly states this. Returning a record to station-reported mode is an explicit edit.
  Bidirectional station synchronization is a later, separately tested phase.

## Operator and portal access

`scripts/printer-registry.py` targets staging only. Supply `LEASH_PRINTER_TOKEN` through the
operator's credential environment, never in an argument or committed file. Commands:

```text
python scripts/printer-registry.py list
python scripts/printer-registry.py get k1c-1f44
python scripts/printer-registry.py save reviewed-printer.json
python scripts/printer-registry.py history k1c-1f44
```

Save takes a complete editable record with `id` and its current `version` (`0` for a new record).
Use `manual: true` to keep the supplied condition and note. A stale edit returns HTTP 409; fetch
and review before retrying. Setting `note` to an empty string explicitly clears it. Record
history is retained; the history endpoint returns the latest 100 edits.

`/printers/manage` offers the same functionality in the portal. Only administrators and accounts
explicitly granted `leash.printers:manage` may edit. Existing volunteer/staff job-reading access
uses `leash.printers:read`. API keys require their own matching scope as well as user permission.

## Staging deployment

1. Publish both components with the existing `Docker` workflow, using workflow dispatch on this
   branch with `component=both`. Verify each manifest digest and amd64/arm64 platform before use.
2. On Spence, use `scripts/deploy-printer-registry-staging.py --frontend <digest-image>
   --backend <digest-image>` to prepare patches. It checks the staging DB name, references only
   the existing staging ingest secret, and creates guarded rollback patches. Add `--apply` to
   roll out the staged pair. It checks production specs remained unchanged.
3. Initially keep the legacy collector bridge enabled (the default). The existing Pi upload to
   staging will seed observed conditions/notes while production continues its existing feed.
   Confirm staging `/printers/data` is fresh and notes match the current source. Existing offline
   printers whose old feed already erased their notes require an explicit operator note; do not
   invent or backfill those from guesses.
4. Install `printer-registry-collector.py` plus the unchanged `printer-fleet-collector.py` in
   `/opt/makerspace-printer-registry-staging` on the Pi. Install the separate `makerspace-printer-
   registry-staging.service` and `.timer`. Its owner-only environment file is
   `/etc/makerspace-printer-registry-staging.env` with only the staging ingest token and, if needed,
   the existing read-only station DB path. The systemd StateDirectory holds the cached roster.
   Keep production's collector, timer, credentials and config untouched.
5. After verifying the separate staging collector, rerun the staging deployment script with
   `--dedicated-collector --apply`. This acknowledges authenticated legacy staging uploads but
   does not apply them, preventing competing fixed/dynamic rosters. The old shared collector
   continues reporting to production successfully. New staging collection uses the backend's
   dedicated roster/ingest routes. An outage uses its validated cached roster.
6. Verify an authorized edit/readback, an offline testing printer, changing lineup without a
   deployment, retained repair notes after stale feed/backend restart, retired visibility,
   anonymous/invalid-auth privacy, and unchanged production. A browser test at desktop and phone
   sizes is required before considering the staged rollout complete.

Rollback: stop the separate staging timer and apply the saved, image-guarded frontend/backend
rollback patches. The additive registry tables remain for recovery. Do not delete records or
alter production's collector to roll back staging.

## Next action

Finish the published build receipt, restore authorized SSH access, and execute the staged
rollout/verification above. Do not call this deployed based on local or GitHub build success.
