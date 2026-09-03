# Printer dashboard — active prototype

Updated: 2026-09-03. Deployed to [staging](https://staging.mkr.cx/printers) and verified ready. Sample-data prototype; not connected to printers.

The staged follow-up includes `Send us a note`, Laurie Anderson in the 16-printer roster, and
material types without filament colors. The earlier approval-service blocker is resolved.

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
  (`Send us a note` and `3D Printer Status`) and a public `3D Printers` menu link directly below Home.
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
- Material is known only as a type such as `PLA` or `PETG`; filament color is not available.
  All sample print details now omit color. Preserve this limit when connecting live data.

Working interpretation announced to Shira: anonymous viewers still see busy/idle and estimated
time remaining, as originally requested; protected current-print information means identity,
filename, material and job details. Resolve this if Shira intends to hide time estimates too.

## Current implementation

- Isolated copy of `umassamherstmakerspace/mkrcx`, based on `f45eaeae894846834257cd0c2a3d3c62de1882cc`.
- `frontend/src/routes/printers/+page.svelte` adds `/printers` within the existing portal layout.
- `frontend/src/lib/printers/prototype-data.ts` now contains 16 real fleet names with entirely
  invented condition, activity, note, time and person fixtures. No sample status is operational truth.
- Laurie Anderson was omitted from the initial 15-name prototype. The fleet seed and the design
  checkout's inventory both identify her as `fdm-k1max-anderson`, a K1 Max; the discovery inventory
  has no hardware ID for her. The corrected roster uses that inventory key, leaves Machine ID
  `Not yet recorded`, and shows unknown sample condition/activity. Historical repair notes are
  not asserted as current status. Row identity and displayed machine ID are now separate fields.
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

## Staging deployment

The authorized staging-only frontend deployment completed after Shira confirmed VPN access.
Review the [homepage](https://staging.mkr.cx/) and [printer dashboard](https://staging.mkr.cx/printers).

Deployed artifact (reuse; no new build required):

- Code commit: `bc485873751da587f8fc0e85a76870aadb990e4e` on pushed branch
  `codex/printer-dashboard-prototype-20260903`.
- GitHub Actions frontend-only dispatch: [run 33779563235](https://github.com/umassamherstmakerspace/mkrcx/actions/runs/33779563235).
  Frontend tests, both native architecture builds, and manifest publication succeeded. Backend
  jobs were skipped as intended. Release lint, all 83 unit tests, and the frontend build passed.
  The four focused local fleet tests also passed after updating expectations for Laurie.
- Published and independently registry-verified image:
  `ghcr.io/umassamherstmakerspace/mkrcx-frontend@sha256:c9dae9fafe1789e58cdc620f6d2c3d258f1acb810d2c0b995ab47413ba0112af`.
  Registry digest matched; `linux/amd64` and `linux/arm64` are available.
- The full local formatting pass encountered Windows checkout CRLF in unchanged files. The
  actual Linux release lint gate passed; unrelated files were not reformatted.
- The current release build was validated in CI. Browser interaction and visual QA were not run.

Deployment evidence:

- Used `ssh maker@spence.infra.mkr.cx` and `kubectl --kubeconfig=/home/maker/.kube/config`.
  The image-only JSON patch tested the existing container name and old image before replacement.
  Server dry-run, apply, and bounded rollout all succeeded for `deployment/mkrcx-frontend-staging`,
  container `mkrcx-frontend`. Readback matched the exact published digest, with ready/updated/available
  replicas all `1/1/1`.
- Production frontend and both staging/production backend images were unchanged and ready.
  Existing staging configuration, database and authentication were preserved.
- Anonymous staging `/` and `/printers` returned HTTP 200. The homepage contained both compact
  shortcuts and `Send us a note`; the printer page contained 16 rows including Laurie Anderson,
  the sample-data label, and Staff preview control. Public HTML omitted the fictional print owner
  and had no Notes sorting control. All ten HTTP acceptance checks passed.
- Source inspection confirmed all six sample jobs contain only `PLA` or `PETG`, without colors.
  The exact validated source is pinned by the deployed image digest above.
- The menu drawer renders its links only when opened. Source inspection confirmed the public
  `3D Printers` menu item directly below Home; no menu interaction test was run.
- The staff preview switch uses only fictional print data; it is not real staff authentication.

Rollback point, measured before deployment and verified pullable for both architectures:
`ghcr.io/umassamherstmakerspace/mkrcx-frontend@sha256:04ec3df5bade2f7104430ca795e4d2990f9b3ed1b2f85c2d4fb97819d95e6395`.
The guarded reverse patch is `/tmp/printer-dashboard-staging-bc48587-rollback.json` on Spence, also
saved as `Makerspace/.scratch/printer-dashboard-staging-bc48587-rollback.json`. It tests the deployed candidate
before replacing only its image; recheck current state before any future rollback.

The local bounded GitHub helper is `Makerspace/.scratch/printer-dashboard-github.py`; it uses the
configured Git credential helper in memory and prints only selected build metadata. The build
image above is already verified; this helper is not a deployment or production permission grant.

## Next checkpoint

Shira reviews the updated staged prototype. All requested follow-up edits are deployed; there is
no remaining staging-build or approval-service blocker.

The user can select `Staff preview` on `/printers` without login, then select a printer name to
expand fictional job details. The existing staging `/login` route is separate from this preview;
signing in does not switch its audience. A read-only check of `/login?return_to=%2Fprinters`
successfully reached Google's sign-in page (HTTP 200). No account sign-in was performed.

After prototype review, before any live-data integration:

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
