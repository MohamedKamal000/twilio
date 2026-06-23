# Umami Analytics — Design

**Ticket:** [OneBusAway/twilio#14](https://github.com/OneBusAway/twilio/issues/14) — Add Umami Analytics support (server-side event emission)
**Date:** 2026-06-22
**Status:** Approved (brainstorming → ready for implementation plan)

## Summary

Add [Umami](https://umami.is) analytics emission to the Twilio voice/SMS service by
registering a new **Umami provider** in the analytics broker that already exists in
this repository. Umami events are emitted server-side by POSTing directly to Umami's
unauthenticated `/api/send` ingestion endpoint. No new broker is built — the existing
`analytics.Manager` is already the single call-site abstraction the ticket's spirit
asks for.

## Decisions (resolved during brainstorming)

| Question | Decision |
| --- | --- |
| How to obtain `umamiAnalytics` `{url, id}`? | **Env vars only** (`UMAMI_URL`, `UMAMI_WEBSITE_ID`). The app does not fetch the region feed today; we do not add region-feed fetching. |
| Plausible provider? | **Keep it.** Add Umami alongside Plausible; the `Manager` fans out each event to both when both are enabled. |
| Session attribution / User-Agent? | **Per-caller UA.** A browser/device-like UA embedding channel + the salted caller hash, so Umami's `isbot` check passes and we get per-caller/per-device session rollups. |

## Background: the existing broker

The `analytics` package already implements a pluggable broker:

- `analytics.Analytics` interface: `TrackEvent` / `Flush` / `Close`.
- `analytics.Manager`: fans one generic `analytics.Event` out to every registered
  provider on a worker pool, each wrapped in a circuit breaker. Enqueue is
  non-blocking (drops on full queue).
- Existing providers: `plausible`, `mock`, `noop`.
- Call-sites already route through one place: `middleware.TrackSMSRequest(...)`,
  `TrackVoiceRequest(...)`, `TrackStopLookup(...)`, `TrackError(...)`,
  `TrackDisambiguation*(...)`. Each dispatches on a goroutine.

```
handler → middleware.Track*  →  Manager (worker pool, circuit breaker)
                                   ├── plausible.Provider   (existing)
                                   └── umami.Provider        (NEW)
```

Nothing at the call-sites changes. The work is: a new provider, config loading, and
`main.go` wiring.

## Component: `analytics/providers/umami/`

A `Provider` implementing `analytics.Analytics`.

### Config

```go
type Config struct {
    ServerURL   string        // e.g. https://analytics.onebusawaycloud.com (required)
    WebsiteID   string        // Umami website UUID (required)
    Hostname    string        // e.g. api.pugetsound.onebusaway.org
    HTTPTimeout time.Duration // default 5s
}
```

`Validate()` returns new sentinel errors added to `analytics/errors.go`:
`ErrMissingServerURL`, `ErrMissingWebsiteID`. Defaults: `HTTPTimeout = 5s`.

### Behavior

- **`TrackEvent(ctx, event)`**: build the Umami payload (below) and do a
  **synchronous POST** to `<ServerURL>/api/send` using an `http.Client` with a tight
  timeout. No internal batching or goroutine — the `Manager`'s worker pool already
  runs this off the request path, so this provider stays simpler than the batching
  Plausible provider. Returns an error on failure so the `Manager` logs it and the
  circuit breaker can trip; the error never reaches a user-facing caller.
- **`Flush(ctx)`**: no-op (nothing buffered).
- **`Close()`**: marks the provider closed; returns `ErrProviderClosed` on reuse,
  matching the existing provider convention.

## Wire format & event mapping

Mirrors the iOS (`UmamiAnalytics.swift`) and Android (`UmamiAnalytics.java`) prior art.

```json
{
  "type": "event",
  "payload": {
    "website": "<WebsiteID>",
    "hostname": "<Hostname>",
    "url": "/sms",
    "name": "sms_request",
    "data": { "language": "en-US", "query": "75403" }
  }
}
```

Mapping from `analytics.Event`:

- `website` ← `Config.WebsiteID`.
- `hostname` ← `Config.Hostname` (defaults to the host of `ONEBUSAWAY_BASE_URL`).
- `name` ← `Event.Name`. All current events are custom events (always set `name`).
- `url` ← synthetic path from an event-name-prefix map so Umami dashboards group
  sensibly:
  - `sms_*` → `/sms`
  - `voice_*` → `/voice`
  - `stop_lookup*` → `/stop`
  - `error_*` → `/error`
  - default → `/`
- `data` ← sanitized `Event.Properties`: keep `string` / number / `bool`; stringify
  anything else; drop nil keys/values (mirrors Android `sanitizeProps`). Add
  `user_id` and `session_id` when present. Omit `data` entirely when empty.

## User-Agent & session attribution

Per-caller, browser/device-like, so Umami's `isbot` check does not silently drop it:

```
OneBusAway-Twilio/1.0 (SMS; <first-12-of-hashed-caller>)
```

- Channel (`SMS` / `Voice` / `Server`) is inferred from the event-name prefix.
- The short hash is derived from the already-salted `Event.UserID`; `anon` when empty.
- Set per request because it varies per caller. This yields per-caller/per-device
  session rollups (Umami attributes sessions from client IP + UA).

### Success detection

```go
// isSuccessfulIngest returns false for non-2xx responses and for the
// {"beep":"boop"} body Umami returns when isbot drops a request.
func isSuccessfulIngest(statusCode int, body []byte) bool
```

A successful ingest body contains `cache` / `sessionId` / `visitId`. A dropped event
returns HTTP 200 with `{"beep":"boop"}` and MUST be treated as a failure.

## Configuration (env-only)

Loaded in `analytics/config_loader.go` via a new `loadUmamiConfig()` added to
`loadProviderConfigs()`, following the existing `loadPlausibleConfig()` pattern:

| Env var | Required | Default | Notes |
| --- | --- | --- | --- |
| `UMAMI_ENABLED` | no | `false` | Master switch for the provider. |
| `UMAMI_URL` | when enabled | — | Umami host; POSTs go to `<UMAMI_URL>/api/send`. |
| `UMAMI_WEBSITE_ID` | when enabled | — | Umami website UUID. |
| `UMAMI_HOSTNAME` | no | host of `ONEBUSAWAY_BASE_URL` | `hostname` field in the payload. |
| `UMAMI_HTTP_TIMEOUT` | no | `5s` | Go duration string. |

`main.go` gets a registration block mirroring the existing Plausible block:
extract config → `umami.NewProvider(...)` → `analyticsManager.RegisterProvider("umami", ...)`.
Misconfiguration when enabled → provider not registered, logged, app continues.

README and CLAUDE.md env tables updated with the `UMAMI_*` vars.

## Event coverage

The ticket's suggested events map onto existing event constructors in
`analytics/events.go`:

| Suggested event | Existing | Notes |
| --- | --- | --- |
| Inbound call | `VoiceRequestEvent` | ✓ |
| Menu selection | `EventVoiceMenuChoice` const | Add emission at the handler call-site if missing. |
| Stop / arrivals lookup | `StopLookupEvent` | ✓ |
| SMS query | `SMSRequestEvent` | ✓ |
| Error / no-results | `ErrorEvent` | Ensure no-results path emits. |

As part of this work, **audit the handler call-sites** and add emissions for any
suggested event not currently fired, so an exercised call/SMS flow actually produces
events in Umami (acceptance criterion).

## Error handling

Fire-and-forget is guaranteed at three layers:

1. `middleware.Track*` dispatches each track on its own goroutine with a bounded
   context.
2. `Manager.TrackEvent` enqueues non-blocking and drops on a full queue.
3. The Umami provider uses a short HTTP timeout and converts all failures into a
   returned error that the `Manager` logs (and the circuit breaker consumes).

No analytics path can block or break a call or SMS.

## Testing

- `analytics/providers/umami` unit tests using `httptest.Server`:
  - Asserts payload JSON shape for both a custom event and a pageview.
  - Asserts the synthetic `url` path mapping.
  - Asserts `data` sanitization (types kept/stringified/dropped).
  - Asserts the request `User-Agent` header is present and non-bot.
  - Asserts a `{"beep":"boop"}` 200 response is detected as failure.
- Table-driven tests for `isSuccessfulIngest` (non-2xx, beep/boop, success bodies,
  empty body).
- A "never blocks / swallows errors" test (server hangs or 500s → caller unaffected).
- `analytics/config_loader` tests for the `UMAMI_*` vars (enabled/disabled, missing
  required, hostname default).
- All existing tests stay green.
- `make lint`, `make vet`, `make test`, `make fmt` must pass before completion.

## Out of scope

- Region-feed (`regions-v3.json`) fetching and per-region matching (deferred; env-only
  config chosen instead).
- Removing or deprecating the Plausible provider (kept as-is).
- Any client-side / JavaScript tracker (the product is server-side only).
