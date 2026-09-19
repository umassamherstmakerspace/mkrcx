# mkr.cx stats page (staff activity dashboard): active handoff

Updated: 2026-09-18

This is the one handoff for the stats page. Update it in place; do not create dated copies.

## Where to work

- Folder: `C:\Users\shira\Makerspace\mkrcx` (the permanent local checkout of
  `umassamherstmakerspace/mkrcx`). Do not start new dated clones under
  `C:\Users\shira\Claude\Makerspace\.scratch\`.
- Branch: `stats/activity-dashboard-20260918`. Local only, not pushed, no upstream set on purpose.
- Page: `frontend/src/routes/(authenticated)/staff/activity/+page.svelte` (one ~570-line file).
- Backend: `backend/src/leash/api/activity.go`, `activity_test.go`,
  `backend/src/leash/commands/activity_snapshot.go`.

## Goal

A read-only page Shira and Lauren glance at. In priority order: (1) pulse and brag numbers they can
quote offhand, (2) catch a problem the same day, (3) staffing by time of day. It is not a report
builder; exact date ranges come from the check-in export. Keep it to one screen with no controls.
Do not add charts, filters, or data sources without a new settled decision below.

## Verified current state (2026-09-18)

- Branch = `origin/main` at `fd9f39c` (PR 39) plus the seven commits from
  `codex/activity-dashboard-main-20260903` (`57162f6` through `2453a18`), cherry-picked with no
  conflicts. The original branch is still pushed and untouched.
- Backend: `go test ./...` passes (run offline with the Go 1.26.7 toolchain and module cache under
  `C:\Users\shira\Claude\Makerspace\.scratch\`).
- Frontend: NOT verified. Node and pnpm are not installed for the `shira` Windows user. CI
  equivalents: pnpm 9.15.5 / Node 22, `pnpm install --frozen-lockfile`, `pnpm run lint`,
  `pnpm run test:unit -- --run`, `pnpm run check`, `pnpm run build`.
- Staging: `/staff/activity` returns 404. Later main-based staging rollouts replaced the
  September 3 deployment. Production never had this page.

## Known hazard

Staging currently carries a login change that is not on `main`: branch
`codex/login-account-chooser-20260918`, commit `427eb13`. A stats build made from this branch and
deployed to staging would remove it. Before any staging deploy, either merge that login change to
`main` and rebase this branch, or explicitly retire it with Shira.

## Deploy route (summary; nothing here authorizes a deploy)

Branch pushes do not build images. Dispatch the `Docker` workflow for frontend and backend, verify
the multi-arch immutable digests, then use the guarded staging deploy helper over VPN and SSH, and
keep exact rollback digests. Production deploys need Shira's explicit approval.

## Unrelated material, do not mistake for this project

- `.scratch/backfill-exploration-20260918/` is printer-filament allowance analysis.
- `.scratch/leadership_stats/` and `attendance_data/` are August data artifacts. They may become
  data sources later, only if the settled goal calls for them.

## Settled decisions (Shira, 2026-09-18)

- Audience: staff, mainly Shira and Lauren.
- Fixed windows only: today, past 7 days, past 30 days, plus semester heatmap and academic-year new
  members. No date picker, no view toggles.
- Headline = people through the door per day, linked or not. One person counts once per day.
  Averages divide by open days (days with at least one tap).
- Unknown cards: keep only a per-day count of distinct unknown cards, computed by the hourly
  check-in maintenance while the seven-day fingerprints exist (`checkin_unknown_dailies` table).
  No change to the approved seven-day fingerprint retention.
- OPEN FORK, parked on purpose: keeping unknown-card fingerprints longer (or a daily-rotating
  fingerprint in the durable row). Only needed to follow one unlinked card over weeks. Privacy
  decision for Shira; do not build without her.

## Built on this branch, not yet verified end to end

- Backend (tests pass): unknown-card daily counts, `pulse` windows, taps per weekday-hour.
- Frontend: page rewritten to pulse cards, "Anything to fix?", "When are we busy?" heatmap,
  academic-year new members. NOT type-checked, linted, or built yet (no Node on this machine).
- Warning thresholds in the page are first guesses: 25% not-linked share, 4 quiet days.

## Next action

Run the frontend checks, then stage it together with the login-account-chooser change (see hazard
above). Staging has no real taps: the September 3 deployment served a production aggregate
snapshot through `ACTIVITY_SNAPSHOT_FILE`, produced by the `activity_snapshot` command. A fresh
snapshot must come from a build that includes this branch, or the pulse section will be empty.
