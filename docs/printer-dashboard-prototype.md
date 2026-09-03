# Printer dashboard — active prototype

Updated: 2026-09-03. Staging deployment authorized and being prepared; not yet staged or connected to printers.

## Goal and settled design

Create an anonymous mkr.cx printer dashboard that can stay open on slicing computers, with
current print identity and job details available to all Makerspace staff.

Shira confirmed:

- Use plain `Idle`; no bed-check reminder is needed in this dashboard.
- All condition notes are public, including Limited use restrictions such as `PLA only` and
  Out of service explanations. Shira will communicate this expectation to staff.
- Current-print person and job details are staff-only. All Makerspace staff can access them.
- Build a local prototype first. Staging on mkr.cx is a later step when the design is ready.
- Shira has now authorized staging the sample-data prototype, plus two compact homepage links
  (`Send a note` and `3D Printer Status`) and a public `3D Printers` menu link directly below Home.
  Production deployment and live printer integration are outside this staging step.
- Programmatic remote condition updates remain a later phase.
- Visual feedback: remove repeated Makerspace branding, the introductory slogan and the
  no-login explanation. Use a simple `3D Printer Fleet Status` heading and a compact fleet list
  that requires much less scrolling. Public notes must remain fully visible.
- List is preferred over cards; search is unnecessary for this fleet and has been removed.
  Both views now use one compact table row per printer. Working printers have green row tint,
  left edge and badge, matching the emphasis for yellow Limited use and red Out of service.
- The list must be sortable. Default: Green, Yellow, Red, then Unknown (the placement of Unknown
  is the implementation recommendation); within each condition, printing/paused precede idle;
  then K1 Max precedes K1/K1C (this model grouping was announced as the working interpretation).
  Names break ties. Column headings reverse sorting; a reset restores this default.
- Notes is a plain, non-sortable heading; alphabetical note sorting has no useful purpose here.
- Staff detail presentation is being tried as a click-to-expand row: selecting a printer name
  opens current-print details directly below it. One detail row is open at a time. Public mode
  closes and removes those details. This is a proposal awaiting Shira's visual feedback.
- Codes such as `1F65` are short inventory-ID suffixes; `K1C-1F65` identifies Gabriela Salazar.
  They are kept in staff details, not the main list. No claim about their hardware derivation
  (such as serial number or MAC address) has been verified.
- The `STAFF` badge next to a current print confused the audience permission with the printer
  user's role. It was removed; `Printing for` labels the fictional person's name without claiming
  that person is staff. Alex Morgan and Jordan Chen are sample users, not actual staff records.

Working interpretation announced to Shira: anonymous viewers still see busy/idle and estimated
time remaining, as originally requested; protected current-print information means identity,
filename, material and job details. Resolve this if Shira intends to hide time estimates too.

## Current implementation

- Isolated copy of `umassamherstmakerspace/mkrcx`, based on `f45eaeae894846834257cd0c2a3d3c62de1882cc`.
- `frontend/src/routes/printers/+page.svelte` adds `/printers` within the existing portal layout.
- `frontend/src/lib/printers/prototype-data.ts` contains 15 real fleet names with entirely
  invented condition, activity, note, time and person fixtures. No sample status is operational truth.
- Public notes, independent condition/activity, time estimates, status filters, fictional staff
  preview, and interrupted-update preview are implemented. Both views are sortable tables with
  Printer, Model, Status, Activity, Est. time left and Notes columns. Staff details expand inline.
  Search and redundant summary, result and density-control rows were removed. Narrow screens
  keep the table with horizontal scrolling; notes wrap and are never truncated.
- `frontend/src/lib/printers/fleet-view.ts` owns the sorting behavior and suppresses stale/missing
  estimates. `fleet-view.test.ts` checks default ordering, reverse sorts, source immutability,
  numerical estimates and missing/stale values remaining last.
- Sample clock is fixed at 2:00 PM. No live polling, printer access, state writes or real login
  implementation has been added. The staff toggle is explicitly a design preview, not authorization.
- Public view contains no current-print person, filename, material, start time or progress display.
  The fixture source includes fictional job details for previewing; this is not a production data boundary.
- Existing site header, colors, dark-mode support, package manager and lockfile are retained.
- Homepage and main-menu navigation now link to `/printers`. The homepage's large Send a note
  card is replaced by two compact links, side by side where the viewport permits.

## Run locally

Use the frontend's pnpm scripts. Run `pnpm exec svelte-kit sync` once before the first `pnpm dev`.
Set `PUBLIC_LEASH_ENDPOINT=http://127.0.0.1:9` for anonymous offline preview; no real service is used.
Run `pnpm run dev --host 127.0.0.1 --port 5183 --strictPort`, then open
`http://127.0.0.1:5183/printers`. This setting intentionally cannot log in.

The local pnpm build approvals are limited to existing locked packages `@parcel/watcher`,
`esbuild` and `svelte-preprocess`, recorded by `pnpm approve-builds` in `frontend/pnpm-workspace.yaml`.
No dependency or lockfile versions were changed.

## Next checkpoint

Validation: the anonymous local `/printers` route returned HTTP 200 and compiled without a route
error. The existing production build passed, including the calendar-worker and calendar-prewarm
smoke checks. Both new source files were formatted with the existing project formatter. Browser
interaction and visual QA were not run; the local preview was queued in Codex for Shira's review.
The running preview uses `http://127.0.0.1:5183/printers`.

Review the visual prototype with Shira and incorporate feedback. Before staging with real data:

1. Replace fixtures with a read-only server integration to printer condition and active-job sources.
   Confirm current fleet coverage; Shira reports four deployed printer UIs, without identifying the
   fourth in this conversation. Do not infer deployment from these examples.
2. Build an allowlisted anonymous response that includes public notes and excludes identities and job
   details. Enforce all-staff authorization on the server for the protected response; remove the
   preview toggle from operational routes. Reuse existing mkr.cx authentication and staff membership.
3. Correlate ownership to the active job; report unknown when a reliable association is absent.
4. Apply per-printer freshness, paused/missing-estimate behavior and end-to-end refresh handling.
   A page reload must not refresh the age of stale printer data.
5. Reconfirm current mkr.cx staging instructions and inspect the resulting staging change before
   the separately authorized deployment. Do not deploy fixture status as live fleet status.

Keep remote condition changes separate from this read-only milestone. The current printer contract
allows future remote Out of service changes but requires return to service at the printer. It exposes
no remote Start, Cancel, Pause or Resume action.
