# Historical printer records

Historical imports live in `printer_historical_entries`, separate from station outcomes,
current telemetry, and staff assessments. Stable source IDs make an import auditable and
retryable. Private source references and import identifiers are not returned by the API.
An import must not convert an old form submission into a completed print, infer its actual
duration, or update a printer's present condition. Keep ambiguous physical-unit matches out
of the import until reviewed.

The staff history endpoint accepts `?page=1&filter=updates|prints|all` and returns at most
20 selected entries. `nextCursor` continues the same printer, filter, and time boundary.
Filtering happens before pagination. `pageIds` defines the merged order; an adjacent record
snapshot may be included only as context for a change at a page boundary. Usage totals are
independent of the page and include only measured station outcomes. Legacy submission
counts, original start-date evidence, and historical meter readings remain separate fields.
The compact staff heading uses `estimate`: approximate accumulated meter hours plus measured
station printing after the last meter date, the combined old/new print count, and the service
start date (or earliest recorded print when no start date exists). Approximate origins such as
"fall 2023" stay approximate. Missing hours are not displayed as zero.

The UI loads more history only when requested. Automatic refresh pauses after loading
additional pages, with an explicit refresh button to return to the newest entries.
Date-only evidence is labeled “Time not recorded”; it must not acquire a fabricated time.
Historical meter readings can reset. The wear estimate merges same-day readings, treats a
drop below half the prior counter (at least 100 hours) as a likely new counter era, and carries
the prior era forward. Small corrections and isolated high/low outliers do not add an era.
These are estimates, not measured lifetime totals; the underlying evidence remains unchanged.
Filename estimates remain attached to individual submissions and are not added to a meter.

Staff history resolves exact usernames/addresses against account display names and reviewed
`printer_identity_aliases`. Aliases are display-only, contain private source provenance, and
never affect authentication or permissions. Unmatched handles and first names remain as
recorded. Staff PIN actions remain unattributed. The stored source identities are not rewritten.

Retired records have a dedicated staff view. They remain outside the public fleet and
collector roster. Imported retired units should have no network address or MAC address.

Deployment is additive: the normal backend migration creates the new table. Before importing,
back up staging assessments and source tables, record both staging image digests and the
production deployment specs, and prepare deletion of the specific import plus only the new
retired records. Verify exact row readback, unchanged existing assessments, staff protection,
and bounded pagination. Roll back application images independently of the imported table.
Source exports and import payloads contain private data and must remain outside Git.
