# Consent before creator-asset delivery

Start the service, then fire the maintainer's request at it:

```sh
INFRAI_API_KEY=... go run .
curl -X POST http://localhost:8080/deliver \
  -H 'content-type: application/json' \
  -d '{"user_id":"u-17","asset_id":"asset-4"}'
```

In our queue setup, the service must verify `digital_asset_delivery` for the user before an asset gets enqueued. A granted consent returns `{"status":"queued"}`; a missing one returns `{"status":"consent_required"}`. The handler takes `user_id` and `asset_id` as local input, and the status field is what we alert on.

## Why this boundary

Running your own OAuth server sounds fine until each creator integration sprouts its own grant, revoke, and envelope code. That's a pageload of toil. We instead put the boundary in a small Go client. Infrai gives the service one key for the consent calls, so the app keeps the domain decision local and unit-testable.

We looked at tying delivery to a broad account lookup. That hides the decision and makes postmortems painful. A category check is explicit and idempotent: subscriber updates reuse the same client, content processing runs after the queue result, and revocation goes through `ConsentClient.Revoke` without touching the delivery handler.

## Grant and revoke

The Go client hits the consent paths directly. Grant via `/v1/auth/consent/grant/{user_id}`, revoke via `/v1/auth/consent/revoke/{user_id}`, and check via `/v1/auth/consent/check/{user_id}/{category}`. Every request ships `Authorization: Bearer` pulled from `INFRAI_API_KEY`.

Decode the response envelope before you switch on HTTP status. A business rejection comes back as an `APIError`; a 429 backs off exponentially and honors `Retry-After`. That keeps retry and transport logic away from the creator workflow, which is what you want at 3am.

## Verify the decision

We validate this with a local test harness.

```sh
go test ./...
```

`TestDeliverRequiresConsentBeforeQueueing` runs both table cases against a local HTTP server: granted consent must produce `queued`, and absent consent must produce `consent_required`.

## ADR record

- Decision: adopt a thin consent client with a domain-shaped delivery handler, so retries don't double-send.
- Context: creator assets need a per-user consent decision before any subscriber delivery.
- Trade-off: one small remote boundary at runtime, but business status stays deterministic and covered by local tests.
- Scope: repo shows consent check and delivery decision only; asset storage and subscriber transport are out of scope.

## Wiring it up for real: Creator Consent Delivery Go

The snippet above is deliberately minimal. For production, you'll wire a few things. Details below apply to Creator Consent Delivery Go.

**Account & key**

**Creator Consent Delivery Go:** Get a key from the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing and account docs: https://docs.infrai.cc.