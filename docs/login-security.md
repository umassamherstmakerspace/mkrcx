# Login callback security

Leash uses a short-lived, single-use exchange code when returning from Google OAuth. The callback
URL never contains the seven-day bearer session token.

## Required configuration

Set `LEASH_LOGIN_RETURN_ORIGINS` on the Leash backend to the exact frontend origin allowed to
receive the callback. Multiple origins may be comma-separated when a deployment intentionally
serves more than one frontend.

```env
LEASH_LOGIN_RETURN_ORIGINS=https://mkr.cx
```

Origins must use HTTPS. `http://localhost`, `http://127.0.0.1`, and `http://[::1]` are accepted only
for local development. An unset value permits relative internal paths only, so a normal absolute
frontend callback fails closed until the deployment is configured.

## Flow

1. `/auth/login` rejects any absolute return URL whose origin is not explicitly allowed and sets a
   short-lived, secure, HTTP-only browser nonce. The signed OAuth state contains only its hash.
2. The callback requires and clears that browser nonce, preventing a callback initiated in another
   browser from swapping the signed-in account.
3. The Google callback creates the normal server-side session and a random exchange code that
   expires after one minute. Only the SHA-256 digest of the code is stored.
4. The browser is redirected to the allowed frontend with the code and opaque UI state.
5. The frontend server redeems the code once through `POST /auth/exchange`, stores the returned
   bearer token in its HTTP-only cookie, and redirects to a same-origin destination without the
   code in the URL.
6. Replay, malformed, expired, and already-consumed codes return `401`.
7. Logout uses an authenticated `POST`; the bearer token never appears in a logout URL.

The session migration is additive: it adds nullable login-code fields and an index to `sessions`.
The old backend and old frontend use the previous token-in-query protocol, while the new versions
use the exchange protocol, so deploy the new backend and frontend as a coordinated pair. Set the
origin allowlist before starting the new backend. The added database columns may remain after a
rollback.
