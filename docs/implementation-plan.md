# Implementation plan

Phases 1–6 are built and verified (this is the video scope). Phases 7–11 (the homework) are now also built — see each phase's updated note below. None of them have been run against real Mongo/RabbitMQ containers yet (phase 6's e2e pass covered phases 1–6 only), so the acceptance checks below are still the thing to run before calling any of phases 7–11 verified, not just implemented.

## Phase 1 — Skeleton + config + logging

- `go.mod`, folder structure, `lib/config/env.go`, `lib/logger/logger.go`, `lib/appcontext`.
- **Acceptance**: `go build ./...` clean. `go run ./cmd/api` fails fast with a clear error if `.env` is missing required vars.

## Phase 2 — pkg/ providers

- `pkg/mongo/client.go`, `pkg/messaging/{types.go,amqp.go}`, `pkg/httpclient/client.go`.
- **Acceptance**: each compiles with zero imports from `lib/` or `app/` (checked by `go vet` catching nothing, and by inspection per the layering rule — there's no automated layer-linter in this slice).

## Phase 3 — Cross-cutting lib/ infra

- `lib/errors`, `lib/http`, `lib/middleware`, `lib/auth`, `lib/rbac`, `lib/coreclient`, `lib/coreevents`.
- **Acceptance**: `GET /health` (wired directly in `boot.go`, no module needed) returns `200 {"success":true,...}` with no auth.

## Phase 4 — `agg_restaurant_day` write path

- `app/analytics/{types,errors,enums}.go`, `entity/restaurant_day.go`, `entity/event_id.go`, `repository/{indexes,restaurant_day.repo,event_id.repo}.go`, `service/analytics.service.go` (`OnOrderPlaced` only), `eventhandlers/handlers.go`.
- **Acceptance**:
  1. Boot logs `mongo connected`, `rabbit connected`, `event consumer started` in order.
  2. `play/publish-test` → `agg_restaurant_day` shows `orders_count: 1, revenue_sum: <total>`.
  3. Same `eventId` replayed → document unchanged.
  4. A second distinct event → counts/sums increment correctly.

## Phase 5 — `agg_restaurant_day` read path

- `service/analytics.service.go` (`GetRestaurantDays`), `controller/{analytics.controller,routes}.go`, `dto/days.*.go`.
- **Acceptance**:
  1. `GET .../days` with no auth → `401 UNAUTHENTICATED`.
  2. With a garbage cookie → `401`.
  3. With a valid JWT + `analytics:read` → `200`, `avgOrderMinor` correctly derived.
  4. `from > to` → `400 ANALYTICS_INVALID_DATE_RANGE`. Missing `from` → `400 VALIDATION_ERROR`. Non-numeric `restaurantId` → `400`.
  5. A restaurant with no rows → `200 []`.

## Phase 6 — `play/` dev aids + full e2e pass

- `play/{mock-core,mint-jwt,publish-test,check-mongo}`.
- **Acceptance**: the full 11-step checklist in the original brief, run against real Mongo + RabbitMQ containers and `play/mock-core` standing in for `core-service` — all steps pass (see `README.md`'s quick-start for the exact commands).

---

## Phase 7 — `agg_branch_day`, `agg_product_day`, `agg_platform_day` *(built)*

- Mirrors Phase 4's entity/repository/indexes split for each: `app/analytics/entity/{branch_day,product_day,platform_day}.go`, `app/analytics/repository/{branch_day,product_day,platform_day}.repo.go`, indexes added to `repository/indexes.go`. `agg_product_day` upserts via `BulkWrite` (`ProductDayRepo.BulkApplyOrderPlaced`, one model per line item) — not a loop. `OnOrderPlaced` now fans out to all four aggregates (restaurant/branch/product/platform) in one call.
- **Acceptance** (run before trusting this phase — not yet exercised against real containers): `play/publish-test -event order.placed` with a line item, then `play/check-mongo -collection agg_branch_day -id-field branch_id -id 1`, `-collection agg_product_day -id-field product_id -id 7`, `-collection agg_platform_day` and confirm each reflects the expected increment; confirm `agg_product_day` gets one document per distinct `productId` after an order with multiple line items.

## Phase 8 — `payment.completed`, `order.delivered`, `order.rejected` handlers *(built)*

- `app/analytics/eventhandlers/handlers.go` adds `handleOrderRejected`/`handleOrderDelivered`/`handlePaymentCompleted`; `AnalyticsService` adds `OnOrderRejected`/`OnOrderDelivered`/`OnPaymentCompleted`. The date key for rejected/delivered is always derived from the order's `createdAt`, never from `rejectedAt`/`deliveredAt` — so the increment lands on the same day-row `order.placed` created even if the lifecycle event itself is processed on a later day.
- **Acceptance** (not yet exercised against real containers): `play/publish-test -event order.delivered` *before* `-event order.placed` for the same `restaurantId` — the upsert must not error (verified via each repo's `$setOnInsert`, which creates the row with zero orders_count/revenue_sum if order.placed hasn't landed yet).

## Phase 9 — 7 remaining endpoints *(built)*

- `restaurants/:id/failure-rate`, `restaurants/:id/delivery-avg` (both derived from `agg_restaurant_day` — no new collection), `restaurants/active`, `branches/:id/days`, `branches/:id/failure-rate`, `products/:id/days`, `platform/days`. All documented in `docs/api-contracts.md` with request/response/error-code triples.
- **Acceptance** (not yet exercised against real containers): each endpoint exercised by at least one `curl` per `docs/api-contracts.md`'s examples, mirroring Phase 6's acceptance style.

## Phase 10 — `rbac.permissions_changed` consumer *(built)*

- A dedicated second queue (`analytics-service.core-events`, bound `rbac.#` on `core.events` — not a second binding on the order-events queue), wired in `lib/boot/boot.go` as a second `coreevents.Consumer`. Handler in `lib/rbac/events.go` calls `cache.Invalidate(role)` per role named in the payload's `roles` list, or `cache.InvalidateAll()` if the list is empty — chosen over a second binding because `rbac.permissions_changed` is published on `core.events`, a different exchange than `order.events`, by a different service (core-service, not order-service); see `docs/ai-prompts.md` item 4 and the comment in `lib/rbac/events.go` for the full reasoning.
- **Acceptance** (not yet exercised against real containers): `play/publish-test -event rbac.permissions_changed -role owner`, confirm the *next* `rbac.Cache.HasPermission("owner", ...)` call misses the TTL'd cache and re-fetches from `play/mock-core`.

## Phase 11 — `cmd/backfill-aggs/` *(built)*

- `cmd/backfill-aggs/main.go` reuses `AnalyticsService.OnOrderPlaced` unchanged against a JSON-lines file of `analytics.OnOrderPlacedInput`-shaped payloads (`-file <path>`), bypassing `lib/coreevents.Consumer` entirely — no ack/nack, no `event_ids` dedupe, since every line is being seen for the first time by definition.
- **Acceptance** (not yet exercised against real containers): run against a JSON-lines fixture of 1000+ historical `order.placed` payloads; `agg_restaurant_day` row counts/sums match a hand-computed expectation for at least one restaurant/date pair.
