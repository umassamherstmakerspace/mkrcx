# Private check-in Discord subscriber

This command is a local prototype adapter between the authenticated mkr.cx `signin` feed and one
private supervisor-only Discord channel. It polls the durable feed cursor, validates the three
approved front-desk outcomes, and edits one rolling Discord webhook message containing the latest
ten valid taps.

It uses only the Go standard library. It does not read card numbers, query users or holds, accept
arbitrary Discord content, or follow HTTP redirects. Discord mentions are disabled in every
message. The webhook URL and API key are never printed.

## Required configuration

Copy the variable names from `config.env.example` into the process environment. Do not put real
values in the repository.

- `MKRCX_SIGNIN_FEED_ITEMS_URL` must be the HTTPS URL for a numeric feed, for example
  `https://mkr.cx/api/feeds/17/items`.
- `MKRCX_SIGNIN_API_KEY` must be a non-full-access service API key. Both the owning service user and
  the API key should have only `leash.feeds.signin:read`; configure the numeric feed ID so neither
  needs `leash.feeds:list`.
- `DISCORD_SIGNIN_WEBHOOK_URL` must be an incoming webhook created in the private supervisor
  channel. Discord role membership controls who can read the channel; the webhook URL is still a
  posting credential and must remain secret.
- `CHECKIN_DISCORD_POLL_INTERVAL` defaults to `2s`.
- `CHECKIN_DISCORD_HTTP_TIMEOUT` defaults to `10s`.

From the `backend` directory:

```powershell
go run ./cmd/checkin-discord
```

This prototype keeps its cursor, latest events, and Discord message ID only in memory. Restarting it
creates a new rolling message and leaves the old message in Discord. That is intentional for the
local demo; durable state and cleanup require a separately reviewed deployment design.

The adapter sends current display names and tap timestamps to Discord. Before using real data,
approve the private channel membership, ten-item visible history, Discord retention implications,
and credential creation. Editing a rolling message limits ordinary visible history but does not
guarantee deletion from Discord systems.

## Tests

The tests use local TLS `httptest` servers and no credentials or external network calls:

```powershell
go test ./cmd/checkin-discord
```
