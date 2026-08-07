# Trusted local NSQ test

The feed topology test uses NSQ v1.3.0 from the official `github.com/nsqio/nsq` module. It does not
use an unsigned Windows release archive or an unpinned container tag.

`scripts/bootstrap-nsq.ps1` restricts module downloads to `proxy.golang.org`, requires verification
through `sum.golang.org`, disables direct VCS fallback, and checks all of the following before it
builds either executable:

- tag: `v1.3.0`;
- upstream repository: `https://github.com/nsqio/nsq`;
- upstream commit: `f580340fd5be61a94d0fef388458f027925c7fc0`;
- module checksum: `h1:v7NtyO844ieTIOCQEqQ7IUSSi1ImhgrTTto1rgIYGEU=`;
- `go.mod` checksum: `h1:RxNr6UC0kSkNF44LnJrlN3U3CQnQGTXk+QKfSZLzqvc=`;
- builder: the separately verified Go 1.26.5 toolchain in `.tools`.

The script builds with read-only module resolution, trimmed paths, and VCS stamping disabled. It
writes an ignored local receipt containing the resulting executable hashes. The test script checks
those hashes before starting anything.

Run the two steps from the repository root in PowerShell:

```powershell
.\scripts\bootstrap-nsq.ps1
.\scripts\test-feed-nsq.ps1
```

The second script binds `nsqlookupd` and `nsqd` only to dynamically selected loopback ports, creates
a temporary data directory, starts three real `FeedRuntime` consumers with unique ephemeral
channels, and verifies ordered fan-out plus feed isolation. It stops both processes and removes the
validated temporary directory in a `finally` block. It does not contact the makerspace network,
pods, databases, or credentials.

The test passes lookupd's HTTP endpoint to the application and its separate TCP endpoint to nsqd,
matching the production configuration contract. Application startup verifies lookupd's HTTP
`/ping` synchronously so direct nsqd connectivity cannot mask a bad discovery address.

On August 3, 2026, repeated local runs delivered eight ordered database-backed events to each of
three runtime hubs in 19.1-55.0 ms total, with no cross-feed leakage. That is evidence for the
broker-to-hub path on this workstation, not yet an end-to-end card-reader, network, authentication,
rendering, or production-load measurement.
