# plan.md — video scope vs. homework

**Update:** phases 7–11 below are now built (see `docs/implementation-plan.md`'s updated acceptance notes). This section is kept as-written for context on the original video/homework split; treat "not built" in the headings below as historical.

## In the video (built, tested end-to-end)

- One aggregate: `agg_restaurant_day`.
- One inbound event: `order.placed`, consumed from `order.events` (queue `analytics-service.order-events`, bindings `order.#, payment.#`, DLQ wired — the bindings are wider than the one event handled, so adding `payment.completed` etc. later needs no topology change).
- Idempotency via Mongo `event_ids` (unique index -> dup-key -> ack-and-skip).
- One read endpoint: `GET /api/v1/analytics/restaurants/:restaurantId/days?from=&to=`.
- All cross-cutting infra: JWT auth, RBAC read-through cache from core (HTTP), correlation IDs, structured logging, error envelope, graceful shutdown.
- Four `play/` dev aids: `mock-core`, `mint-jwt`, `publish-test`, `check-mongo`.

## Homework (not built — see `docs/implementation-plan.md` phases 7+)

1. **3 more aggregates**: `agg_branch_day`, `agg_product_day`, `agg_platform_day`.
2. **3 more event handlers**: `payment.completed`, `order.delivered`, `order.rejected`.
   - Gotcha: out-of-order delivery — `order.delivered` can arrive before `order.placed` finishes dedupe-and-upsert on a slow consumer; the upsert's `$setOnInsert` must not assume `agg_restaurant_day` already exists.
   - Gotcha: `agg_product_day` is one upsert **per line item** — use `BulkWrite`, not a loop of `UpdateOne`.
3. **7 more endpoints**: failures, delivery-avg, active-restaurants, etc. The failure-rate and delivery-avg endpoints are **derived** from fields already on `agg_restaurant_day`/`agg_branch_day` — no new collections.
4. **`rbac.permissions_changed` consumer** calling `cache.Invalidate(role)`.
5. **Backfill command** under `cmd/backfill-aggs/`.

See `docs/ai-prompts.md` for prompt templates to tackle each of these with an AI, anchored to the patterns this slice already establishes.
