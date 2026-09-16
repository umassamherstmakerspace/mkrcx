# Persistent printer registry — staging rollout

Updated: 2026-09-16

## Scope and current boundary

Shira authorized building and deploying to staging first. Production deployment and changes to
live printer gating are outside this release. Staff editing is optional; management from an
authorized operator's tooling is the primary requirement.

Implementation now lives on `codex/printer-records-history-20260915`. It combines current main
`a7893b8309e3ec74ae2edd103d63192e2204fc6c` (including the approved registration landing page)
with the earlier registry candidate `2e89e373e169a0aba1b3882b14b0f792614cb1f1`.
Shira approved the proposed scope: persistent offline conditions, editable lineup, and a bounded
staff history view, tested on staging. Production and print-gating behavior remain outside scope.

The release is deployed to [staging](https://staging.mkr.cx/printers), including the separate
roster-driven collector. Production remains on the approved registration release.
The initial September 15 release passed the complete Go suite, 11 collector tests, 96 frontend
unit tests, frontend lint/build, and local Svelte checks. Desktop and 390 px phone review used explicitly synthetic local records:
offline history, shelving, retaining the condition source during lineup edits, switching to a
manual note when edited, save/readback, readable edit history and no horizontal overflow all pass.
The phone review caught and corrected a missing history panel before deployment.
Cluster access works through `maker@armengaud.infra.mkr.cx` using its established host-key alias
and `sudo k3s kubectl`; the previous Spence access blocker is obsolete.

The live central station was read with SQLite `mode=ro` and `query_only=ON`. It retains job events,
condition edits and recovery events. The first history view imports recent starts, outcomes, and
condition edits; it never exports raw event payloads or access credentials. Older failures may
have no stored error text, which the UI states explicitly. Future observed `print_stats.message`
errors are retained with their observation time. Full historical backfill, quota/statistics work,
and recovery/authentication logs are deferred.

## Deployed release and verification

- Backend source `fc794a8c5747473a0bec109ac4849c3a8bce39d5`, [successful build](https://github.com/umassamherstmakerspace/mkrcx/actions/runs/35024149727):
  `ghcr.io/umassamherstmakerspace/mkrcx-leash@sha256:f1124bd2c821a49eb6b2da8daf2e4d350bc7e572cad8c4e26d98174ae7758c15`.
- Frontend source `74efd56f2444bc41596d2310827c98018ad14484`, [successful build](https://github.com/umassamherstmakerspace/mkrcx/actions/runs/35107612455):
  `ghcr.io/umassamherstmakerspace/mkrcx-frontend@sha256:12f55aa735ad10b8235dba6c53f51ed880b7e4b8e9787fb43eba163fc1f95286`.
  The manifest was verified as pullable for amd64 and arm64. This UI release adds the dedicated
  staff printer page and unified History. The 111 frontend unit tests, frontend lint/build and
  local Svelte checks pass. Synthetic desktop and 390 px phone checks cover navigation, mixed
  history, ten-entry expansion to fifteen entries, note save/return, and no horizontal overflow
  or nested history scrolling. No synthetic records were written to staging.
- September 16 rollout changed only the staging frontend. The staging backend and production
  deployment specs stayed unchanged, with dedicated collection still enabled. Live readback:
  16 public printers, eight saved notes, fresh feed, no private fields, and ten denied access
  checks including both HTML and JSON on the new detail route. Registration landing passed.
  The staging public fleet also rendered successfully in the browser. Signed-in live review
  remains Shira's checkpoint below.
- Shira explicitly approved installing the separate 20-second collector on the central Pi and
  sending status, saved notes and recent job/error history to `leash.staging.mkr.cx`.
  The installed collector's normalized source SHA-256 is
  `f5551e3630201f579dd752c7e8f0e1cce37a5ae807958fc988fce2937a188816`.
  Installation checked the written bytes; service exits successfully. The timer, existing
  production collector timer and print-station service remain active.
- Successful collection reported 15 configured network printers and 274 history events. The
  registry contains 16 records, including the existing unconfigured Laurie placeholder.
  The frontend uses `PRINTER_REGISTRY_DEDICATED_COLLECTOR=true`; legacy staging uploads are
  acknowledged without applying their fixed roster.
- Public feed is HTTP 200 with no-store caching, eight saved notes at final readback, and no
  job, error detail, history, actor, hardware address or progress fields. Anonymous and invalid
  credential checks for history and management are denied. Registration landing still passes.
- The staging backend was restarted. Seven saved notes had condition timestamps preceding the
  new pod's `2026-09-15T21:24:53Z` start, proving stored conditions survived. An immediate ingress
  read briefly returned 502; the subsequent readback succeeded. All four staging/production
  deployments are ready and updated at 1/1. Production deployment specs were unchanged by both
  staging rollouts and its image pins remain unchanged.
- Signed-in staging editor/history review remains Shira's checkpoint: she chose to sign in in
  her external browser, which this task's browser tools cannot inspect. Authenticated editor
  behavior was exercised with the synthetic UI fixture and backend API tests; do not describe
  that as a completed signed-in staging browser test.

The September 16 UI-only rollback is
`C:\Users\shira\AppData\Local\Temp\mkrcx-printer-registry-staging-7qehhuq_\mkrcx-frontend-staging-rollback.json`.
It restores the previous frontend digest `3dccc57aabc29b5426c73490ba46de43c9031c7e4504995cdfe1f678770364d1`
and keeps the dedicated collector enabled. This UI rollback does not require stopping the collector.

Original image-guarded rollback patches are in the workstation directory
`C:\Users\shira\AppData\Local\Temp\mkrcx-printer-registry-staging-5_3q0ubd`.
They restore frontend
`ghcr.io/umassamherstmakerspace/mkrcx-frontend@sha256:193ea13e37898b7c8213a16b08a724bd14a867cdfac25103323eef8cd4aa220b`
and backend
`ghcr.io/umassamherstmakerspace/mkrcx-leash@sha256:fffa8d662946a70c69751ac2a5350409f7c523b419a4478290356d17fec11a0f`.
The later collector-mode rollback is in `mkrcx-printer-registry-staging-mmw0_xnw` alongside it;
that later patch does not restore the original application images. For a full rollback, stop the
separate staging timer/service, then apply the original image-guarded patches. Retain the tables.

## Behavior

- Staff printer names link to `/printers/[id]` on desktop and phone. The page shows saved
  condition, current activity, lineup, last seen, notes, current print when available, and History.
  It refreshes status every 15 seconds and clears private details when access is lost.
- History merges record edits and printer events by timestamp. Notes, jobs, errors and changes
  have different layouts; ten entries appear initially, with Load more and no nested scroll area.
  Consecutive edit versions show changed fields; a truncated baseline is labeled Record saved.
  Automated printer events and staff edits retain source labels. Service-account imports do not
  imply automated authorship. Ship's-log/standup imports and source links remain a future integration;
  this UI release does not claim to import those notes.
- Edit opens the selected printer's form and returns to its detail page after saving or cancelling.
  Editing still requires `leash.printers:manage`. The condition caveat is limited to
  "Doesn't change printer controls." The note field is marked public.
- MariaDB stores physical printer records, current saved condition/note, station observations,
  last-seen time, edit versions, and an append-only record edit history.
- The initial 16 identities are a migration seed, not a runtime allowlist. A restart never resets
  an edited or retired record. New records appear without a frontend or collector code change.
- Live readings cannot overwrite a manually maintained note. Offline/stale readings preserve
  the last known condition and explanation but remove live activity/estimates and job details.
- Testing, repair, shelving and retirement are lineup states separate from condition. Shelved and
  retired records remain in staff views/history but are omitted from the public list and polling roster. Testing/repair printers are excluded
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
- Lineup-only edits preserve the selected condition source. Editing a note or condition explicitly
  selects the saved dashboard record.
- History returns the latest 100 station/observed events and 50 dashboard edits. Each collection
  reads up to 30 job events and 20 condition events per polled printer, capped at 1,000 total;
  previously imported events remain durable. Failed collection leaves stored history intact.

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
history is retained; the history endpoint returns the latest 50 edits and 100 printer events.

`/printers/manage` offers the same functionality in the portal. Only administrators and accounts
explicitly granted `leash.printers:manage` may edit. Existing volunteer/staff job-reading access
uses `leash.printers:read`. API keys require their own matching scope as well as user permission.

## Staging deployment

1. Publish both components with the existing `Docker` workflow, using workflow dispatch on this
   branch with `component=both`. Verify each manifest digest and amd64/arm64 platform before use.
   For UI-only changes, dispatch `component=frontend` and use `--frontend-only` below. The supplied
   backend image must match the running staging backend; its deployment spec is checked unchanged.
   The dedicated collector is already installed. All subsequent rollouts must include
   `--dedicated-collector`; steps 3–5 describe the initial setup, not work to repeat for UI changes.
2. From this Windows workstation, use `scripts/deploy-printer-registry-staging.py --frontend <digest-image>
   --backend <digest-image>` to prepare patches. It checks the staging DB name, references only
   the existing staging ingest secret, and creates guarded rollback patches in a fresh private directory for each run, preserving prior rollback points. Add `--apply` to
   roll out the staged pair. It sends kubectl operations through Armengaud because that host
   has no Python interpreter, retains patches locally, and checks production specs remained unchanged.
3. Initially keep the legacy collector bridge enabled (the default). The existing Pi upload to
   staging will seed observed conditions/notes while production continues its existing feed.
   Confirm staging `/printers/data` is fresh and notes match the current source. The dedicated
   collector also reads retained station conditions for offline machines; never infer a working
   condition from an unreported default.
4. Install `printer-registry-collector.py` plus its companion `printer-fleet-collector.py` in
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

Shira reviews the signed-in staging History and Manage printer records views, including the
desired actual lineup choices. Production promotion and merging this feature into main require
her subsequent instruction. The statistics page, quotas and touchscreen/gating changes are outside
this release.
