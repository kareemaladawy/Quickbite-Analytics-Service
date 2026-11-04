# AI prompts for the homework

The skill you're practicing in this homework isn't "learn Go." It's **directing an AI through a non-trivial implementation in an unfamiliar language by anchoring it to a codebase you already understand** (this service's video slice, plus the Node services it mirrors). Every prompt template below is built around that anchor — point the AI at existing code before asking it to write new code, every single time.

---

## Rule 1: make the AI prove it understood the existing code before writing new code

Don't open with "implement `agg_branch_day`." Open with a comprehension check you can actually verify:

> "Before writing anything: read `app/analytics/repository/restaurant_day.repo.go`, `app/analytics/entity/restaurant_day.go`, and `app/analytics/repository/indexes.go`. Explain back to me, in your own words: (1) why the upsert uses `$inc` instead of `$set`, (2) why `agg_restaurant_day`'s unique index is on `(restaurant_id, date)` and not just `restaurant_id`, (3) what would break if `revenue_sum` were replaced with a stored `avg_order_value` field. Don't write any code yet."

If the AI's answer is vague or wrong on any of those three, it hasn't understood the pattern well enough to extend it — correct it before letting it touch a file. If it nails the "why," you've confirmed it's mirroring the *intent*, not just the *shape*, of the existing code.

---

## Rule 2: ask for the Node analogue, then the Go translation

For anything that isn't a straight copy of an existing Go file, force a two-step answer:

> "Before writing the Go `agg_branch_day` repository: show me the Node analogue first — if `order-service` had a branch-day rollup repo written in the same style as `app/order/repository/order.repo.ts`, what would it look like? Then translate that to Go, following the pattern in `app/analytics/repository/restaurant_day.repo.go` exactly (same upsert shape, same index-naming convention, same file layout)."

This catches two failure modes at once: the AI inventing a Go-only convention that doesn't match this codebase, and the AI porting a Node idiom that doesn't actually apply in Go (see `docs/node-to-go-mapping.md`'s "common gotchas" and "when to break the mapping" sections — hand the AI that doc directly when you suspect it's doing either).

---

## Concrete prompts, one per homework item

### 1. `agg_branch_day`, `agg_product_day`, `agg_platform_day`

> "Add three more aggregate collections, following `agg_restaurant_day` exactly: same entity/repository/indexes split, same `$inc`-based upsert, same unique-index-plus-range-index pair. `agg_product_day` is different in one way: a single `order.placed` event has multiple line items, so its upsert must be one `BulkWrite` call (one `UpdateOne` model per line item), not a loop of individual `UpdateOne` calls — look at how `order-service`'s `order.repo.ts` batches related inserts with `whereIn` for the same N+1-avoidance reason, then find the Mongo equivalent. Show me the entity + indexes + repository for one of the three before doing the other two, so I can review the pattern once."

### 2. `payment.completed`, `order.delivered`, `order.rejected` handlers

> "Add handlers for `payment.completed`, `order.delivered`, `order.rejected`, following `app/analytics/eventhandlers/handlers.go`'s `order.placed` pattern: a local payload struct trimmed to what the handler needs, one `consumer.RegisterHandler` call, dispatch into a new `AnalyticsService` method. Before writing `order.delivered`'s handler: explain the out-of-order-delivery gotcha noted in `plan.md` — what happens if `order.delivered` is processed before `order.placed` for the same order, given Mongo's upsert is unconditional on existence? Show me you've handled that (or explain why `$setOnInsert` already makes it safe) before I approve the diff."

### 3. The 7 remaining endpoints

> "Add the failure-rate and delivery-avg endpoints first — re-read `app/analytics/service/analytics.service.go`'s `GetRestaurantDays` and confirm these two are derivable from fields already on `agg_restaurant_day`/`agg_branch_day` (no new collection, no new event handler). Show me the derivation formula in plain English before writing the Go. Then add the remaining list/active-restaurants endpoints, following `controller/analytics.controller.go` + `dto/days.*.go`'s validate → service → DTO → `SendSuccess` shape exactly."

### 4. `rbac.permissions_changed` consumer

> "Add a second event consumer (or a second binding on the existing one — tell me which you'd pick and why, citing `lib/coreevents/consumer.go`'s `RegisterHandler` map) for `rbac.permissions_changed`, calling `rbacCache.Invalidate(role)`. Read `order-service/src/lib/auth/rbac.events.ts` and `rbac-events.controller.ts` first — that's the Node side of this exact feature — and tell me what payload shape it publishes before writing the Go payload struct."

### 5. `cmd/backfill-aggs/`

> "Write a one-shot backfill command under `cmd/backfill-aggs/main.go` that re-derives `agg_restaurant_day` from a dump of historical `order.placed` events (take a flag for a JSON-lines file path). Before writing it: explain why this can reuse `AnalyticsService.OnOrderPlaced` unchanged rather than writing separate backfill logic, and why it must NOT go through `lib/coreevents.Consumer` (hint: there's no RabbitMQ message to ack/nack, and no dedupe should apply to a one-time backfill of data that's never been seen before) — if your explanation doesn't match, redo the design before coding."

---

## When the AI's Go doesn't match this codebase

If the diff adds a DI framework, an ORM, a new top-level middleware pattern, or anything not covered in `CLAUDE.md`/this mapping doc, say so explicitly rather than editing around it:

> "This doesn't match the codebase's conventions — re-read `CLAUDE.md` §10 ('what to avoid') and `docs/node-to-go-mapping.md`'s 'when to break the mapping' section, then redo this without introducing [the thing]. If you think this case genuinely warrants breaking the pattern, argue for it first instead of just doing it."

The point of all of this: you already know what "good" looks like for the one slice that's built. Every homework prompt should spend its first move pointing the AI at that slice, not describing the new feature in the abstract — the existing code is more precise than any English description of "how we do things here" could be.
