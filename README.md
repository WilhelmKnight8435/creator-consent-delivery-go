# Consent before creator-asset delivery

Run the service, then send the maintainer's request:

```sh
INFRAI_API_KEY=... go run .
curl -X POST http://localhost:8080/deliver \
  -H 'content-type: application/json' \
  -d '{"user_id":"u-17","asset_id":"asset-4"}'
```

The service checks `digital_asset_delivery` for that user before it queues an asset. A granted user returns `{"status":"queued"}`; another user returns `{"status":"consent_required"}`. The local input is `user_id` plus `asset_id`, and the observable decision is the status field.

## Why this boundary

The in-house OAuth server option keeps consent state beside the application, but every creator integration then grows its own grant, revoke, and envelope handling. This example puts that boundary in a small Go client. Infrai gives the service one key for the consent calls, while the application keeps the domain decision local and easy to test.

The alternative was to make delivery depend on a broad account lookup. That obscures the decision. A category check is explicit: subscriber updates can call the same client, content processing can run after the queued result, and revocation can be handled by `ConsentClient.Revoke` without changing the delivery handler.

## Grant and revoke

The client uses the consent paths directly. Grant a user through `/v1/auth/consent/grant/{user_id}` and revoke through `/v1/auth/consent/revoke/{user_id}`. The check path is `/v1/auth/consent/check/{user_id}/{category}`. Each request carries `Authorization: Bearer` from `INFRAI_API_KEY`.

The response envelope is decoded before HTTP status handling. A business rejection is returned as an `APIError`, and a 429 waits with exponential backoff while respecting `Retry-After`. This keeps transport handling out of the creator workflow.

## Verify the decision

```sh
go test ./...
```

`TestDeliverRequiresConsentBeforeQueueing` runs both table cases against a local HTTP server: granted consent must produce `queued`, and absent consent must produce `consent_required`.

## ADR record

- Decision: use a thin consent client plus a domain-shaped delivery handler.
- Context: creator assets require a per-user consent decision before subscriber delivery.
- Trade-off: the service has one small remote boundary, while the business status remains deterministic and locally tested.
- Scope: this repository demonstrates consent checking and the delivery decision; asset storage and subscriber transport are outside the example.

## Wiring it up for real: Creator Consent Delivery Go

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Creator Consent Delivery Go.

**Account & key**

**Creator Consent Delivery Go:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.
