# Room 119 Availability — live staging

Updated: 2026-09-03. [Staging](https://staging.mkr.cx/room) has a logged-in room-availability
calendar backed by the private Ag Eng 119 Google Calendar feed. Production is unchanged.

## Settled behavior

- The page is available to every authenticated mkr.cx user and is linked as `Room 119 Availability`
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

- Deployed source commit: `81e42ce3314f4ce793a78cbb450c1632c80030e9` on
  `codex/printer-dashboard-prototype-20260903`.
- Frontend-only [GitHub Actions run 33787507579](https://github.com/umassamherstmakerspace/mkrcx/actions/runs/33787507579)
  passed the full frontend test/lint gate, both native-architecture builds, and manifest
  publication. Backend jobs were skipped.
- Deployed, registry-verified image:
  `ghcr.io/umassamherstmakerspace/mkrcx-frontend@sha256:38bff323258e6f0332738aa342cd03abfff0b77e82bc61cd0981460605cbcbe7`.
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

The current guarded frontend rollback files are `/tmp/room-calendar-name-staging-rollback.json` on
Spence and `.scratch/room-calendar-name-staging-rollback.json` locally. The earlier combined
code/config rollback remains at `/tmp/room-calendar-staging-rollback.json`. The optional
staging-secret removal patch
is `/tmp/room-calendar-secret-rollback.json` on Spence and
`.scratch/room-calendar-secret-rollback.json` locally. Recheck current state before rollback.

## Next checkpoint

Complete one real staff login: confirm the hamburger menu contains `Room 119 Availability`, the page
shows `Staff view`, bookings have their real titles, and selecting one opens its details. Then use
an authenticated ordinary-member account or a bounded server-side acceptance fixture to confirm
the live response contains only `Busy` plus time fields before making a separate production
decision.
