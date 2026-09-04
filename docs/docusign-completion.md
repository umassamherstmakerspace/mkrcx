# DocuSign waiver completion

mkr.cx creates an active hold named `docusign` when a member registers. The
hold is cleared automatically when DocuSign Connect reports that the matching
Makerspace Use Agreement envelope and its `Member` signer are complete.

## Receiver contract

DocuSign posts JSON notifications to:

```text
POST https://leash.mkr.cx/api/webhooks/docusign
```

The receiver is public only in the HTTP-routing sense. It accepts a completion
only when all of these checks pass:

- `X-DocuSign-Signature-1` is a valid HMAC-SHA256 signature of the exact raw
  request body;
- the Connect event and envelope status are both complete;
- `data.accountId` matches the configured DocuSign account;
- the envelope subject starts with the configured waiver-only prefix;
- exactly one completed signer has the configured `Member` role;
- that signer's normalized email matches an active mkr.cx account.

The handler clears only an active `docusign` hold created at or before the
envelope completion time. That time boundary makes Connect retries idempotent
and prevents a replayed old completion from clearing a newer waiver hold.
The hold's existing `AddedBy` service actor is recorded as `RemovedBy`.

Responses are intentionally free of member details:

- `200 resolved` — an eligible hold was cleared;
- `200 already_resolved` — the event was a safe duplicate or no eligible hold
  remained;
- `202 unmatched` — no active member matched the completed signer email;
- `204` — the signed event is not for the configured account or waiver subject;
- `401` — the HMAC signature failed;
- `422` — the trusted event omitted required completion or signer data;
- `503` — the receiver is not fully configured.

## Required Connect configuration

Create a DocuSign Connect configuration for completed envelope events using
JSON notifications. Enable envelope data and recipient data. Configure HMAC
signing, and use the same secret in the mkr.cx deployment. Do not include
documents or document PDFs; the receiver neither needs nor accepts them.

Configure all four backend values together:

```text
DOCUSIGN_CONNECT_HMAC_SECRET=<secret shared only with DocuSign Connect>
DOCUSIGN_ACCOUNT_ID=<the sending DocuSign account ID>
DOCUSIGN_WAIVER_SUBJECT_PREFIX=UMass Makerspace Use Agreement |
DOCUSIGN_WAIVER_RECIPIENT_ROLE=Member
```

The HMAC secret and account ID belong in the deployment's Kubernetes Secret;
the subject prefix and role are stored in its ConfigMap. The Secret references
are optional so an image can deploy before Connect is enabled, but a partially
configured receiver stays fail-closed with `503`. Restart the backend after
adding or rotating either Secret value because configuration is loaded at
startup.

## Browser behavior

The mkr.cx header refreshes account holds when its tab regains focus or becomes
visible. The waiver banner also provides **I signed—check again**. If Connect
has not arrived yet, the member is explicitly told to wait and not sign a
second copy. Once the hold has cleared, the next refresh removes the banner.

Card-tap history remains an immutable statement about the hold state at the
original tap. A new tap after the completion is processed returns green.

## Release sequence

1. Deploy the code to staging with a staging-only Connect configuration and
   secret.
2. Create a staging member and hold, complete one staging envelope, and verify
   the hold clears and the banner disappears on return to the tab.
3. Replay the same notification and verify it is a no-op.
4. Create a newer hold and replay the old notification; verify the newer hold
   remains active.
5. Test a wrong account, unrelated subject, bad HMAC, and unmatched email.
6. After explicit production approval, add the production secret/configuration,
   deploy the reviewed images, and perform one controlled real completion.
7. Reconcile already-completed envelopes against still-active holds separately;
   Connect does not make that historical cleanup implicit.
