# Historical check-in backfill

This one-shot command migrates card-server `swipes` into mkr.cx's durable `checkin_events` table.
It is dry-run-only unless `-apply` is supplied. Run it only after the mkr.cx schema migration and
before retiring the card-server read routes in production.

Required environment variables:

- `CARD_DATABASE_DSN`: a **read-only** MySQL DSN for card-server.
- `MKRCX_DATABASE_DSN`: a MySQL DSN for mkr.cx. Use read-only credentials for the dry run and a
  separately approved least-privilege credential for `-apply`.

Optional flags are `-start` (inclusive RFC3339), `-end` (exclusive RFC3339), and `-batch-size`
(default 500, maximum 5000). The command prints aggregate counts only. It never prints card values,
member identifiers, UUIDs, or either DSN.

Historical ownership cannot be proven from the card-server rows alone. A swipe whose card is linked
to exactly one current mkr.cx account at import time receives that member's persistent analytics UUID,
but is marked `linked_at_tap=false` and `identity_resolution=historical_current_link`. Unlinked,
invalid, and normalized ownership-conflict cards are imported with a blank member UUID. No raw card
value or card fingerprint is written to mkr.cx.

Safe promotion sequence:

1. Run the dry run with the full intended time range and save only its aggregate counts.
2. Reconcile `scanned` against a separate aggregate source row count.
3. Review `linked`, `unresolved`, `invalid_card`, and `ambiguous_card` counts.
4. Run the same command and range with `-apply` after approval.
5. Rerun the dry run; every row should be `already_exists` and `would_insert` should be zero.
6. Test a bounded CSV through the signed-in staff page before promoting the card-server retirement.

The import key is the source swipe row ID under `card-server:historical`, so reruns are idempotent.
If an existing event's timestamp or source conflicts with its source row, the command stops instead
of overwriting it.
