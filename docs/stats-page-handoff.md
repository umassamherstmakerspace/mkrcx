# mkr.cx stats page (staff activity dashboard): active handoff

Updated: 2026-09-18

This is the one handoff for the stats page. Update it in place; do not create dated copies.

## Where to work

- Folder: `C:\Users\shira\Makerspace\mkrcx` (the permanent local checkout of
  `umassamherstmakerspace/mkrcx`). Do not start new dated clones under
  `C:\Users\shira\Claude\Makerspace\.scratch\`.
- Branch: `stats/activity-dashboard-20260918`. Local only, not pushed, no upstream set on purpose.
  Until it is pushed, this laptop is the only copy.
- Page: `frontend/src/routes/(authenticated)/staff/activity/+page.svelte` (one ~240-line file).
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
- Frontend: verified later the same day, see below. CI
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
- Wording (Shira, 2026-09-19): minimal. No explanatory sentences, footnotes, or method notes on
  the page. Say "visitors" and "card taps". Insights are one short line with the numbers in it.
  No card-reader status here; the front desk HUD already shows Live.
- Cards (Shira, 2026-09-19): four windows: today so far, yesterday, past 7 days, past 30 days.
  Each shows total unique visitors, split into New (account created inside the window; people
  register in person, so on the Today card this correctly means registered today), Returning,
  Student staff, and Unknown (distinct unlinked cards). The 7-day and 30-day cards add a small
  "about N a day" (average over open days; an open day has at least 5 visitors). No raw card-tap
  counts on the cards; the heatmap is the one place raw taps are right (door busyness). No "+"
  marker: Unknown beyond 7 days undercounts until Shira extends fingerprint retention, which she
  plans to do separately.
- Staff (Shira, 2026-09-19): professional staff are not visitors and are dropped from every
  number, heatmap included. They are identified by account fields, never by name: type
  `employee` with role `staff` or `admin` (same rule as the check-in export). Student staff
  (role `staff` or `admin` on a non-employee account) ARE visitors, because they use the space
  off shift too; they get their own group. No calendar or shift matching: decided too complicated.
  CHECK BEFORE RELEASE: confirm the six professional accounts really have type `employee`.
- Bottom "Totals" table: visitors and new registrations for the current semester and each
  academic year. Years before tap records existed show a dash for visitors.
- Unlinked insight = distinct unknown cards in the past 7 days that nobody has linked since, over
  all distinct visitors in that span (`still_unlinked`). People who tapped unlinked and then
  linked do not count. Seven days is the limit because that is how long fingerprints exist.
- OPEN FORK, parked on purpose: keeping unknown-card fingerprints longer (or a daily-rotating
  fingerprint in the durable row). Only needed to follow one unlinked card over weeks. Privacy
  decision for Shira; do not build without her.

## Built and verified locally on this branch (2026-09-18)

- Backend: unknown-card daily counts, `pulse` windows, taps per weekday-hour. `go vet` clean,
  `go test ./...` passes.
- Frontend: page rewritten to pulse cards, "Anything to fix?", "When are we busy?" heatmap,
  academic-year new members. `svelte-check` 0 errors, prettier and eslint clean on the changed
  files, 134 unit tests pass, production build succeeds.
- An open day needs at least 5 people (`activityOpenDayMinimumPeople`), so a staff member tapping
  in on a closed day does not divide the average.
- Seen in a browser with invented numbers on 2026-09-18 (phone and desktop width): renders as
  intended. Not yet seen with real data. To repeat: in the docs repo, `.claude/launch.json` has
  `mkrcx-frontend-preview` (vite dev on port 5199, no backend); open `/zz-preview-activity`. That
  route is a local, git-excluded file (`.git/info/exclude`) holding sample data; it is not in any
  commit and must never be committed. The amber threshold (25% still unlinked) is a first guess.
- Repo-wide `pnpm run lint` reports about 135 files on a fresh Windows clone. That is CRLF line
  endings from checkout, not code; lint the changed files directly. For the same reason, never run
  `go fmt` on a whole package and then `git add -A`; stage named files.

## Local tooling (no system installs)

- Go: `C:\Users\shira\Claude\Makerspace\.scratch\go-toolchain-1.26.7\go\bin\go.exe` with
  `GOMODCACHE` = `.scratch\go-mod-cache-1.26.7`, `GOPROXY=off`, `GOFLAGS=-mod=readonly`.
- Node 24: `.scratch\mkrcx-feed\.tools\node-v24.18.0-win-x64`; pnpm through
  `corepack pnpm@9.15.5` with `COREPACK_HOME=C:\Users\shira\Makerspace\.corepack`.
- Unit tests: `pnpm exec vitest run`. `pnpm run test:unit -- --run` falls into watch mode and hangs.

## Next action

1. Show Shira the local preview (see above) and adjust layout and the two warning thresholds.
2. Stage it together with the login-account-chooser change (see hazard above): push a staging
   branch = this branch + `427eb13`, dispatch the `Docker` workflow, deploy by digest with rollback
   digests saved. Shira approved the staging route on 2026-09-18; SSH to the cluster may be blocked
   for Claude sessions, in which case hand her or Codex one prepared command.
3. Staging has no real taps. The September 3 deployment served a production aggregate snapshot
   through `ACTIVITY_SNAPSHOT_FILE`, produced by the `activity_snapshot` command. A fresh snapshot
   must come from a build that includes this branch, or the pulse cards will be empty. Reading
   production for that snapshot needs Shira's explicit approval each time.
