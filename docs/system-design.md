# System design

## Platform diagram

```
                         ┌──────────────┐          ┌──────────────┐
   HTTP (sync)           │ core-service │          │ order-service│
   ┌────────────────────▶│ (Node/PG)    │          │ (Node/PG)    │
   │  GET /internal/rbac/permissions     │          └──────┬───────┘
   │                      └──────────────┘                 │
   │                                                        │ events_outbox
   │                                                        │ (transactional,
   │                                                        │  drained to AMQP)
   │                                                        ▼
┌──┴─────────────┐                               ┌───────────────────┐
│ analytics-     │◀──────────────────────────────┤  RabbitMQ          │
│ service (Go)   │   consume: order.events        │  exchange:         │
│                │   queue: analytics-service.     │  order.events       │
│  - event       │          order-events            │  (topic, durable)   │
│    consumer    │   bindings: order.#, payment.#   └───────────────────┘
│  - Mongo       │
│    aggregates  │
│  - read API    │
└──────┬─────────┘
       │ HTTP (sync)
       ▼
┌─────────────┐
│  MongoDB    │
│  agg_*_day  │
│  event_ids  │
└─────────────┘
       ▲
       │ GET /api/v1/analytics/...
       │ (JWT + RBAC)
┌──────┴─────┐
│  Client /  │
│  dashboard │
└────────────┘
```

## Sync vs. async flows

| Flow | Transport | Direction | Why |
| --- | --- | --- | --- |
| RBAC permission lookup | HTTP, `GET /api/v1/internal/rbac/permissions` | analytics-service → core-service | Permissions change rarely but must be correct on every request; a read-through cache with a short TTL (5 min) trades a little staleness for no round-trip on the hot path. |
| Order/payment activity | AMQP, `order.events` topic exchange | order-service → analytics-service | Analytics is not on the write path of placing an order — a queue means order-service's p95 is never affected by analytics being slow or down. |
| Read API | HTTP, `GET /api/v1/analytics/...` | Client → analytics-service | Read-only, cacheable, no reason to be anything but a normal REST call. |

analytics-service never calls order-service or core-service to *write* anything, and never publishes events of its own — it's a leaf node in the event graph.

## Failure-mode table

| Failure | Effect | Recovery |
| --- | --- | --- |
| RabbitMQ unreachable at boot | `boot.Run()` returns an error, process exits non-zero. | Process supervisor restarts; `order.events` queue is durable, so nothing published while we're down is lost once we reconnect. |
| RabbitMQ connection drops mid-run | In-flight `Consume` goroutine's channel closes; no auto-reconnect (see `pkg/messaging/amqp.go`'s doc comment). | Process restart (same tradeoff core-service's own plain-amqplib client accepts) re-establishes the connection and topology (idempotent `assertTopology`). |
| Mongo unreachable at boot | `boot.Run()` returns an error, process exits non-zero. | Process supervisor restart. |
| Mongo unreachable mid-run (event handler) | `ApplyOrderPlaced` returns an error; the event handler returns an error; the consumer Nacks (no requeue) — message goes to the DLQ. | Ops replays the DLQ once Mongo is back. The event is **not** silently dropped. |
| core-service unreachable (RBAC lookup) | `FetchPermissions` returns an error; the request fails with a 5xx (propagated as `AppError.Internal`). | Once core-service recovers, the next request repopulates the cache. No stale-permission risk because a failed fetch never populates the cache. |
| Duplicate event delivery (at-least-once) | `EventIDsRepo.MarkSeen` dup-key → handler is skipped, message is Acked. | Not a failure — this is the expected steady state of AMQP's delivery guarantee. |
| Malformed message body | `coreevents.Consumer` fails to unmarshal the envelope, logs a warning, Nacks without requeue. | Message lands in the DLQ for manual inspection; a malformed envelope is a producer bug, not something to retry blindly. |
| Unknown event type (e.g. a future `order.delivered` before its handler exists) | Logged, Acked — **not** sent to the DLQ. | By design: the DLQ is for failures, not for "we haven't built that handler yet." The binding (`order.#, payment.#`) deliberately covers more routing keys than this slice handles. |

## Why Mongo-dedupe, not Redis `SETNX`

order-service's inbound consumer (`lib/core-events/consumer.ts`) dedupes via Redis `SETNX` with a TTL, because its handlers are **pure cache invalidations** — idempotent by nature, and safe to "forget" after 24h because a stale cache entry just means one extra round-trip to core, not a wrong number.

analytics-service's handlers are **not** idempotent cache invalidations — they're `$inc` upserts into a financial aggregate. If the dedupe record expired and a redelivered `order.placed` slipped through, the result is a silently wrong `revenue_sum` that nothing will ever correct. Two things follow from that:

1. **The dedupe TTL matters for correctness, not just cache hygiene.** A 7-day TTL on `event_ids` (vs. order-service's 24h) reflects that this is a safety window for RabbitMQ redelivery (seconds to minutes in practice), not a cache expiry — it should outlive any realistic redelivery/ops-replay scenario, and if it's ever hit, the failure mode is a double-counted aggregate, not a wrong permission check.
2. **Mongo gives us an atomic, durable, unique-indexed insert for free**, in the same database the aggregate already lives in. A dup-key error on `InsertOne` is both the "have I seen this?" check and its own storage, in one round-trip, with no separate infra dependency (this service already has no Redis — adding one just for dedupe would be a second moving part to get consistent with Mongo's availability, for no benefit over the index Mongo already gives us).

If this service ever needs a hot-path idempotency check with sub-millisecond latency at very high volume, Redis `SETNX` is the right tool — but a few thousand `order.placed` events/day does not need it, and correctness here is worth more than the latency Redis would save.
