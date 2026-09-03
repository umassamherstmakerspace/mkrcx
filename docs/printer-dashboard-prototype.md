# Printer dashboard — live production

Updated: 2026-09-03. [Production](https://mkr.cx/printers) and
[staging](https://staging.mkr.cx/printers) receive the same real read-only fleet snapshot from the
central printer Pi.

## Settled product design

- `/printers` is public without login. It shows all 16 printers, working/limited/out-of-service/
  unavailable condition, activity, estimated time remaining, and public condition notes.
- All condition notes are public. Staff must write them with that audience in mind.
- Current print identity, filename, material, start time and progress are protected. Authenticated
  Makerspace staff select a printer name to expand those details. The server reports
  `Unknown / unassigned` when it cannot correlate an exact active file to a user.
- The fleet is a dense sortable list. Default order is green, yellow, red, unavailable; within
  condition, active precedes idle, then K1 Max precedes K1/K1C, then printer name.
- On screens 760 px wide or narrower, the desktop table becomes a compact stacked list with no
  horizontal scrolling. Each row keeps the printer, model, condition, activity, estimate and any
  public note visible; a compact select and direction button preserve sorting. Staff details expand
  inside the same row.
- Notes are visible but not sortable. Search is intentionally absent.
- The homepage has compact `Send us a note` and `3D Printer Status` links. The public menu has
  `3D Printers` directly below Home.
- Material is known only as a type such as PLA or PETG. Do not display an inferred color.
- Remote condition changes remain a later phase. There is no remote Start, Cancel, Pause, or
  Resume action.

## Live architecture

- `frontend/src/lib/printers/prototype-data.ts` now contains only the 16-printer identity roster
  and unavailable defaults. It has no invented runtime state or jobs. Laurie Anderson remains the
  one printer without a recorded address or hardware ID and is shown unavailable.
- `frontend/scripts/printer-fleet-collector.py` runs on the central Pi every 20 seconds. It verifies
  each addressable printer's expected MAC and uses only Moonraker GET requests plus read-only SQL
  against the station database. It sends the same complete bounded snapshot to staging and
  production over HTTPS using separate credentials. It holds no printer credential and cannot send
  a printer command.
- `/printers/ingest` authenticates and validates only a complete, current, uniquely identified
  16-printer snapshot. The frontend retains the latest snapshot in memory; a process restart safely
  shows unavailable until the next collector run.
- `/printers/data` constructs a public allowlist. It adds current-job identity, filename, material,
  start time and progress only after server-side Leash staff authorization. A missing, invalid,
  expired, member, or otherwise unauthorized session receives the public response.
- The page refreshes from its same-origin endpoint every 15 seconds. Data older than 90 seconds is
  replaced with unavailable condition/activity and no protected job details. A page reload does
  not refresh data age.
- The four currently installed gating UIs are Doris Salcedo, Simone Leigh, Arthur Ganson, and Jean
  Tinguely. They supply current condition and note. Moonraker supplies live activity for the 15
  addressable printers.

## Verification and deployment

- Deployed UI source: `81e42ce3314f4ce793a78cbb450c1632c80030e9`; dual-target collector
  source: `259bd95` on `codex/printer-dashboard-prototype-20260903`.
- Frontend-only [GitHub Actions run 33787507579](https://github.com/umassamherstmakerspace/mkrcx/actions/runs/33787507579)
  passed release lint, all unit tests, both native-architecture builds, and manifest publication.
  Backend jobs were skipped. The focused fleet/privacy tests and four dual-target collector tests
  passed.
- Deployed, independently registry-verified image:
  `ghcr.io/umassamherstmakerspace/mkrcx-frontend@sha256:38bff323258e6f0332738aa342cd03abfff0b77e82bc61cd0981460605cbcbe7`.
  Linux amd64 and arm64 are present.
- Staging and production frontends run that digest at ready/updated/available `1/1/1`. Both
  staging and production backends retained their prior image digests and readiness.
- The staging and production Kubernetes secret references are committed on k8s branch
  `codex/printer-dashboard-staging-ingest-20260903`; production configuration is at `7906205`.
  No secret value is committed.
- Before installation, the collector produced one valid real read-only fleet snapshot with 15/16
  activity readings. Existing Pi station and identity services stayed active. The installed
  collector source SHA-256 matches the reviewed local source.
- Anonymous `/printers/data` on both hosts returned the same current 16-printer snapshot with zero
  job fields and `stale: false`.
  At deployment time, 15 printers supplied current activity and the four installed gating UIs
  supplied condition/notes. The page returned HTTP 200 with 16 rows and no Sample data or Staff
  preview controls.
- Missing/invalid authentication returned the public response with zero protected fields. The
  ingest endpoint returned 401 without its credential. Focused tests prove incomplete, duplicate,
  and stale snapshots are rejected; public responses strip jobs; staff responses retain them; and
  expired readings fail closed.
- At a 390 × 844 browser viewport, the staged page displayed all 16 mobile rows, hid the desktop
  table, showed the mobile sort control, and measured 390 px document width at a 390 px viewport.
  This proves the staged printer list has no horizontal page overflow at that phone size.
- A real collector interruption made every condition/activity unavailable after 90 seconds and
  exposed no job fields. Restarting one refresh restored current data. Both existing Pi services
  remained active. A local assertion command returned exit 1 because its PowerShell dictionary
  check was malformed, but its printed values all passed and the independent recovery readback
  confirmed current data; this was a test-command defect, not a product failure.

The prior production frontend image is
`ghcr.io/umassamherstmakerspace/mkrcx-frontend@sha256:557f31e015ad3d898f96da600f48061706a8e7e18ee205a88609dfbd33eeeea7`.
The guarded production patches are `/tmp/mkr-cx-production-forward.json` and
`/tmp/mkr-cx-production-rollback.json` on Spence, with matching files in the project `.scratch`
directory. The prepared Pi rollback is `.scratch/rollback-printer-fleet-production-collector.sh`;
its pre-production source and environment backups remain on the Pi. Recheck live state before any
rollback.

## Next checkpoint

Complete one real logged-in staff check in production: sign in through Google, return to
`/printers`, confirm `Staff view`, and expand an active printer's protected details. Anonymous,
invalid-token, ingest-auth, fresh-feed, and stale/recovery checks already pass.

For local anonymous development, set `PUBLIC_LEASH_ENDPOINT=http://127.0.0.1:9`. Without a collector
snapshot the fleet correctly displays unavailable.
