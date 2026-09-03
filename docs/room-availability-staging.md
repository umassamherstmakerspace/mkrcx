# Ag Eng 119 availability — live staging

Updated: 2026-09-03. [Staging](https://staging.mkr.cx/room) has a logged-in room-availability
calendar backed by the private Ag Eng 119 Google Calendar feed. Production is unchanged.

## Settled behavior

- The page is available to every authenticated mkr.cx user and is linked as `Room Availability`
  in the hamburger menu. Logged-out requests return to the page after login.
- Ordinary members see only `Busy`, start/end time, and all-day status. The server removes the
  title, description, location, source UID, and every other source-calendar field before returning
  JSON.
- Authenticated staff receive the booking title, description, location, and time. They can select a
  booking to open the same detail dialog used by the staff calendar.
- The private Google iCal URL is a credential. It is stored only as `ROOM_CALENDAR_ENDPOINT` in the
  existing `mkrcx-frontend-staging-secrets` Kubernetes Secret. The endpoint never returns or logs
  the source URL. Temporary local and Spence copies used during installation were removed.
- The calendar is read-only. The page provides no create, edit, delete, or booking action.

## Implementation and verification

- Source commit: `89b7b1b5a3a016f733e9c58a2b3a9a69fbd3d212` on
  `codex/printer-dashboard-prototype-20260903`.
- Frontend-only [GitHub Actions run 33786418583](https://github.com/umassamherstmakerspace/mkrcx/actions/runs/33786418583)
  passed the full frontend test/lint gate, both native-architecture builds, and manifest
  publication. Backend jobs were skipped.
- Deployed, registry-verified image:
  `ghcr.io/umassamherstmakerspace/mkrcx-frontend@sha256:d108f5553fcc631268317666bb08ca203f59190b3d6d8351f7052319b6096f7c`.
  Linux amd64 and arm64 are present.
- The staging manifest secret reference is committed on k8s branch
  `codex/printer-dashboard-staging-ingest-20260903` at `4b7fb45`. No credential value is committed.
- Svelte type/markup checks passed with zero errors and warnings. Ten focused calendar,
  authentication, and privacy tests passed. Tests prove anonymous denial, ordinary-member access,
  field-level redaction, staff detail access, invalid-token denial, and correct bearer-token use.
- The supplied private URL matched the known Ag Eng 119 calendar ID and Google private-iCal URL
  shape. From Spence it returned HTTP 200, a valid VCALENDAR envelope, and 15 events. The installed
  Kubernetes value was compared byte-for-byte with the supplied URL before temporary copies were
  removed.
- Live anonymous checks passed: `/room` returned a 307 redirect to
  `/login?return_to=%2Froom`; `/room/calendar` returned 401. Staging rolled out at ready/updated/
  available `1/1/1`; the production frontend and both staging/production backends retained their
  prior images and readiness.

Guarded rollback files are `/tmp/room-calendar-staging-rollback.json` on Spence and
`.scratch/room-calendar-staging-rollback.json` locally. The optional staging-secret removal patch
is `/tmp/room-calendar-secret-rollback.json` on Spence and
`.scratch/room-calendar-secret-rollback.json` locally. Recheck current state before rollback.

## Next checkpoint

Complete one real staff login: confirm the hamburger menu contains `Room Availability`, the page
shows `Staff view`, bookings have their real titles, and selecting one opens its details. Then use
an authenticated ordinary-member account or a bounded server-side acceptance fixture to confirm
the live response contains only `Busy` plus time fields before making a separate production
decision.
