# Check-in export access

The pseudonymous check-in CSV is restricted to individually approved professional staff. Access
requires all three conditions:

1. a signed-in human session;
2. an account with `type=employee` and role `staff` or `admin`; and
3. the direct user permission `leash.checkins:export`.

The endpoint does not honor a role-level copy of this permission. Student staff, volunteers, API
keys, and service users are rejected even if they otherwise have broad access. The first release
does not include names, emails, internal user IDs, card numbers, or card fingerprints.

## Grant procedure

The Leash image contains the `checkin_export_access` administrative command. It prompts for the
target account email instead of requiring the email on the command line, and its output contains
only role, type, and permission-state booleans. Run it interactively inside the selected staging or
production Leash pod.

First preview the exact change; omitting `-apply` never writes policy:

```console
/leash checkin_export_access -action grant
```

Confirm the result identifies an employee with the expected staff/admin role and reports
`had_permission=false has_permission=true change_required=true applied=false`. Then repeat with the
write flag:

```console
/leash checkin_export_access -action grant -apply
```

Repeat the dry run once more. It must report `had_permission=true`, `change_required=false`, and
`applied=false`. Grant people one at a time; never add this permission to a role.

## Revoke and rollback

Revocation also previews by default and is allowed even if the target is no longer an employee or
staff member:

```console
/leash checkin_export_access -action revoke
/leash checkin_export_access -action revoke -apply
```

After applying, repeat the dry run and verify the target receives `403` from both the staff page and
the CSV endpoint. Revocation removes only `leash.checkins:export`; it preserves every other direct
permission. Existing export-audit rows remain.

## Future identified export

Names or emails are not an extension flag on this CSV. If an identified export is approved later,
give it a distinct permission (for example `leash.checkins:export_identified`), a distinct page and
endpoint, explicit fields and purpose, and separate audit and retention review. Use `member_uuid` as
the stable join key for registrar or self-reported data kept under that separate governance.
