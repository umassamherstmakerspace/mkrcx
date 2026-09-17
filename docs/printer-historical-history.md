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
counts, original start-date evidence, and historical meter readings are separate fields.

The UI loads more history only when requested. Automatic refresh pauses after loading
additional pages, with an explicit refresh button to return to the newest entries.
Date-only evidence is labeled “Time not recorded”; it must not acquire a fabricated time.
Historical meter readings can reset and are not a lifetime total. Filename estimates remain
estimates attached to individual submissions.

Retired records have a dedicated staff view. They remain outside the public fleet and
collector roster. Imported retired units should have no network address or MAC address.

Deployment is additive: the normal backend migration creates the new table. Before importing,
back up staging assessments and source tables, record both staging image digests and the
production deployment specs, and prepare deletion of the specific import plus only the new
retired records. Verify exact row readback, unchanged existing assessments, staff protection,
and bounded pagination. Roll back application images independently of the imported table.
Source exports and import payloads contain private data and must remain outside Git.
