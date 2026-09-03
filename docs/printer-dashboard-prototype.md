# Printer dashboard — live staging

Updated: 2026-09-03. [Staging](https://staging.mkr.cx/printers) receives real read-only fleet data
from the central printer Pi. Production is unchanged.

## Settled product design

- `/printers` is public without login. It shows all 16 printers, working/limited/out-of-service/
  unavailable condition, activity, estimated time remaining, and public condition notes.
- All condition notes are public. Staff must write them with that audience in mind.
- Current print identity, filename, material, start time and progress are protected. Authenticated
  Makerspace staff select a printer name to expand those details. The server reports
  `Unknown / unassigned` when it cannot correlate an exact active file to a user.
- The fleet is a dense sortable list. Default order is green, yellow, red, unavailable; within
  condition, active precedes idle, then K1 Max precedes K1/K1C, then printer name.
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
  against the station database. It sends one complete bounded snapshot outward over HTTPS using a
  staging-only credential. It holds no printer credential and cannot send a printer command.
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

- Deployed source: `518f526813639408f14705de0162c338f2dfefa0` on
  `codex/printer-dashboard-prototype-20260903`.
- Frontend-only [GitHub Actions run 33781974815](https://github.com/umassamherstmakerspace/mkrcx/actions/runs/33781974815)
  passed release lint, all unit tests, both native-architecture builds, and manifest publication.
  Backend jobs were skipped. Seven focused local fleet/privacy tests passed.
- Deployed, independently registry-verified image:
  `ghcr.io/umassamherstmakerspace/mkrcx-frontend@sha256:aab895b67fb252d7eded87529196207f3d3e5819b5ece01ff6baccb305c09b1a`.
  Linux amd64 and arm64 are present.
- Staging rollout completed at ready/updated/available `1/1/1`. Production frontend and both
  staging/production backends retained their prior image digests and readiness.
- The staging-only Kubernetes secret reference is committed on k8s branch
  `codex/printer-dashboard-staging-ingest-20260903` at `6438e47`. No secret value is committed.
- Before installation, the collector produced one valid real read-only fleet snapshot with 15/16
  activity readings. Existing Pi station and identity services stayed active. The installed
  collector source SHA-256 matches the reviewed local source.
- Anonymous `/printers/data` returned a current 16-printer public response with zero job fields.
  At deployment time, 15 printers supplied current activity and the four installed gating UIs
  supplied condition/notes. The page returned HTTP 200 with 16 rows and no Sample data or Staff
  preview controls.
- Missing/invalid authentication returned the public response with zero protected fields. The
  ingest endpoint returned 401 without its credential. Focused tests prove incomplete, duplicate,
  and stale snapshots are rejected; public responses strip jobs; staff responses retain them; and
  expired readings fail closed.
- A real collector interruption made every condition/activity unavailable after 90 seconds and
  exposed no job fields. Restarting one refresh restored current data. Both existing Pi services
  remained active. A local assertion command returned exit 1 because its PowerShell dictionary
  check was malformed, but its printed values all passed and the independent recovery readback
  confirmed current data; this was a test-command defect, not a product failure.

Rollback frontend image:
`ghcr.io/umassamherstmakerspace/mkrcx-frontend@sha256:c9dae9fafe1789e58cdc620f6d2c3d258f1acb810d2c0b995ab47413ba0112af`.
The guarded staging reverse patch is `/tmp/printer-dashboard-live-staging-rollback.json` on Spence
and `.scratch/printer-dashboard-live-staging-rollback.json` locally. The prepared Pi sidecar
rollback is `.scratch/rollback-printer-fleet-collector.sh`. Recheck live state before rollback.

## Next checkpoint

Complete one real logged-in staff check: sign in through Google, return to `/printers`, confirm
`Staff view`, and expand an active printer's protected details. Anonymous and invalid-token checks
already pass. Then allow a short staging soak covering a natural print start/finish or pause
transition before making the separate production decision.

For local anonymous development, set `PUBLIC_LEASH_ENDPOINT=http://127.0.0.1:9`. Without a collector
snapshot the fleet correctly displays unavailable.
