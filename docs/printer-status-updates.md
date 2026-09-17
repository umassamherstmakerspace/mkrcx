# Printer status and history updates

The registry stores current assessments independently of machine connectivity and print permission.
An actual staff condition or note change at the printer updates its corresponding saved field on
the next collector ingestion. This uses the station's `staff_runtime_changed` event and verified
`conditionChanged`/`noteChanged` flags. No new collector fields or installation are required.

The station's staff API changes condition only when explicitly supplied. An authorization override
alone retains the condition and note. Machine/system events, job outcomes, reconnections, and older
events without verified comparison flags remain observations and do not establish repair success.

`condition_set_at` and `note_set_at` retain separate source timestamps. Migration initializes them
from the existing saved assessment time, without changing the assessment. Newer field changes win;
old collected history and retries cannot replace a more recent reviewed update. A note-only event
does not advance the condition clock. Record version advances when an assessment field is applied,
so an editor holding an earlier version receives a conflict. Fleet placement and next actions stay
unchanged. The original station event supplies the audit history instead of duplicating it as a
website-edit event.

Human reports go through ship's log or an explicitly requested agent update. A history-only summary
does not change current condition. The Makerspace repository's `MANUAL_STANDUP_PROTOCOL.md` routes
explicit publication requests to `tools/printer_updates.py`, which saves history and any reviewed
current-state changes transactionally and verifies readback. That operator path requires an explicit
staging/production destination, stable source IDs and current versions for state edits. It uses
existing operator access and does not deploy the website or modify printer controls.

Operator-reviewed test-history exclusions use `printer_history_events.hidden_reason`. Both history
endpoints omit excluded rows before limiting/pagination, and ingestion preserves the exclusion on
retries. Excluded staff tests cannot update the current assessment. The original event and its raw
fields remain stored. This metadata is private and cannot be supplied through collector JSON.
Real print records, usage calculations, meter readings and source archives are not excluded.
