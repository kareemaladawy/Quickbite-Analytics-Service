# CLAUDE.md — Analytics Service Guidelines

These rules apply to the **`analytics-service`** microservice of the **QuickBite** platform. It is the Go analogue of `core-service`/`order-service` — same philosophy (explicit wiring, no ORM, no DI framework, module-per-bounded-context), different language and database. When in doubt, look at how `order-service` does it in Node/TS and find the Go analogue. Deviate **only** where a deviation is documented here or in `docs/node-to-go-mapping.md`.

---

## 1. Mission

This service owns **per-day rollups** of order/payment/delivery activity:

- **Consumes** events from `order-service` via RabbitMQ (`order.events` exchange).
- **Upserts** day-grained aggregates into MongoDB.
- **Serves** read-only HTTP endpoints under `/api/v1/analytics/...`.

It does **not** own users, restaurants, products, orders, payments. It does **not** write to any operational data store. It does **not** emit events — it is a pure consumer/reader.

---

## 2. Tech stack (locked — do not deviate)

| Concern     | Choice                                   |
| ----------- | ----------------------------------------- |
| Runtime     | Go 1.21+                                  |
| HTTP router | `github.com/go-chi/chi/v5`                |
| DB          | `go.mongodb.org/mongo-driver` (official, v1) |
| Messaging   | `github.com/rabbitmq/amqp091-go`          |
| Config      | `github.com/caarlos0/env/v11`             |
| Logger      | stdlib `log/slog`                         |
| Validation  | `github.com/go-playground/validator/v10`  |
| JWT         | `github.com/golang-jwt/jwt/v5`            |
| UUID        | `github.com/google/uuid`                  |
| DI          | **None.** Explicit constructor wiring in `lib/boot/boot.go`. |

**Forbidden**: GORM/Ent/any ODM, DDL migrations. Mongo is schemaless; indexes are created idempotently on boot via `repository.EnsureIndexes` (declared per-module in `app/<module>/repository/indexes.go`, **not** in `pkg/mongo`).

---

## 3. Folder structure

```
cmd/api/main.go            # ~10 lines: calls lib/boot.Run() and exits
pkg/                        # framework-free, app-agnostic
  mongo/client.go            # Connect/Disconnect ONLY
  messaging/{types.go,amqp.go}  # Broker interface + amqp091-go impl
  httpclient/client.go        # net/http wrapper: timeout, JSON, retry-on-5xx
lib/                         # app-aware glue
  boot/boot.go                # wires every singleton
  config/env.go                # struct + Load()
  logger/logger.go             # slog wrapper + FromContext
  appcontext/context.go        # ctx keys: claims, correlation_id
  errors/{apperror.go,handler.go}  # AppError + Wrap(handler)
  http/response.go             # SendSuccess, SendPaginated
  middleware/correlation.go    # Correlation + AccessLog
  auth/{jwt.go,middleware.go,apikey.go,types.go,errors.go}
  rbac/{cache.go,middleware.go,types.go}
  coreclient/{client.go,rbac.go,types.go}
  coreevents/{consumer.go,payloads.go}  # generic only
app/analytics/                # PACKAGE analytics — shared module types
  types.go                     # OnOrderPlacedInput, RestaurantDayRow, ...
  errors.go                    # var Err... = apperr.New(...)
  enums.go                     # const PermAnalyticsRead = "analytics:read"
  entity/                      # plain structs + bson tags
  repository/                  # ONLY place mongo-driver appears
    indexes.go                  # EnsureIndexes for THIS module's collections
  service/analytics.service.go # struct + methods only
  controller/{analytics.controller.go,routes.go}
  dto/{days.request.go,days.response.go}
  eventhandlers/handlers.go    # event type -> service method map
play/                         # GITIGNORED — dev aids (mock-core, mint-jwt, publish-test, check-mongo)
docs/
```

### Layering rules (enforced by review)

```
app/  -> may import lib, pkg
lib/  -> may import pkg, env; may NOT import app/<module>/*
pkg/  -> no imports from lib or app, no env, no globals, NO app-specific knowledge
```

If a `lib/*` file needs something from `app/*`, invert the dependency: define a small interface in `lib/` and let the `app/` type satisfy it implicitly via Go duck typing. Example: `lib/coreevents.EventDeduper` is satisfied by `app/analytics/repository.EventIDsRepo` without any import from `lib/coreevents` to `app/analytics`.

**Strict reading of "no app-specific knowledge in `pkg/`"**: `pkg/mongo` knows how to connect; it does not know `agg_restaurant_day` exists. If you find yourself writing a collection name in `pkg/`, stop.

---

## 4. Naming conventions

- **Files**: `snake_case.go` for multi-word filenames (`restaurant_day.repo.go`), matching the Go convention (not `kebab-case`, which isn't valid in Go import paths' file names anyway).
- **Go**: `PascalCase` exported types/functions, `camelCase` unexported, `UPPER_SNAKE` has no Go equivalent — use `PascalCase` consts (`PermAnalyticsRead`), grouped in a `const (...)` block.
- **Mongo collections**: plural, `snake_case` (`agg_restaurant_day`, `event_ids`).
- **Mongo fields**: `snake_case` via `bson:"..."` tags. Booleans prefixed `is_`.
- **Routes**: plural resource nouns, nested for relations (`/restaurants/:restaurantId/days`).

---

## 5. Module file conventions (per `app/<module>/`)

1. `entity/*.go` — plain struct, `bson` tags, no methods beyond simple invariants, no mongo-driver import.
2. `dto/*.request.go` — `validator` struct tags. Validated by `validate.Struct(...)` in the controller.
3. `dto/*.response.go` — plain structs + a `XResponseFrom(entity)` factory function (the Go analogue of a static `from()` factory). Controllers never return raw entities or service-layer types.
4. `repository/*.repo.go` — exported methods on a struct wrapping `*mongo.Collection`. Only place mongo-driver appears.
5. `service/*.service.go` — struct + methods. Holds business logic. No inline types/errors — those live in the parent package.
6. `controller/*.controller.go` — methods have signature `func(w http.ResponseWriter, r *http.Request) error`, wrapped by `errors.Wrap` at mount time. Validate -> call service -> DTO -> `http.SendSuccess`.
7. `controller/routes.go` — wires middleware (`auth.Authenticate`, `rbac.Require`) to controller methods.
8. `enums.go` — typed string consts.
9. `errors.go` — `var Err... = apperr.New(code, message, status)`. No inline `apperr.New(...)` at call sites for named cases.
10. `types.go` — module-shared, non-entity types. **Inline `type X struct{...}` declarations inside `*.service.go`, `*.controller.go`, or `*.repo.go` are forbidden** — they belong in `types.go` (or `lib/<area>/types.go` / `payloads.go` for cross-cutting infra).

---

## 6. Response envelope

- Success: `{"success": true, "data": ...}`.
- Error: `{"success": false, "error": {"code": "...", "message": "...", "details"?: ...}}`.
- Every HTTP response is shaped by a response DTO. Never pass an entity or a service-layer type straight to `http.SendSuccess`.
- Money: integer minor units. Timestamps: ISO-8601 UTC strings.

---

## 7. Database (MongoDB) rules

- **No DDL migrations.** Indexes live in `app/<module>/repository/indexes.go`, created idempotently via `Indexes().CreateMany` with named indexes, called from `lib/boot` once per module on startup.
- **Averages are never stored directly.** Store `sum` + `count`; derive the average in the service layer at read time (`avgOrderMinor := revenueSum / ordersCount`). This makes every aggregate update an associative `$inc` — safe to replay.
- **Upserts, not read-modify-write.** `UpdateOne(filter, {$inc: ..., $set: ..., $setOnInsert: ...}, upsert: true)` — never `FindOne` then `Save`; that race-conditions under concurrent handlers for the same (restaurant, date).
- **Idempotency is the event_ids collection, not Mongo transactions.** A unique index on `event_id` turns a replayed insert into a dup-key error, translated to "already seen, ack-and-skip" — no distributed transaction needed because the upsert itself is associative.
- **One upsert per logical unit.** When a handler needs multiple documents (e.g. the homework's `agg_product_day`, one row per line item), use `BulkWrite`, not a loop of individual calls.

---

## 8. Cross-cutting infra

### Auth
- JWT verified with `ACCESS_SECRET` (same secret as core/order). HS256. Claims: `userId, role, restaurantId?, restaurantRole?, branchIds?`.
- Token source: cookie (`access_token`) first, then `Authorization: Bearer <token>`.

### RBAC
- No permissions catalog of its own. Looks up role permissions via core's `GET /api/v1/internal/rbac/permissions?role=` (header `api-key: <internal>`).
- In-process, TTL'd cache (`lib/rbac.Cache`, default 5 min) — no Redis dependency.
- `rbac.permissions_changed` consumer calling `cache.Invalidate(role)` is **homework** — not built in the video slice.

### Idempotency
- Every event handler funnels through `EventDeduper.MarkSeen(eventId)` before dispatching. Duplicate -> ack-and-skip. Unknown event type -> log + ack-skip (never DLQ an unrecognized type).

### Error handling
- Services/controllers return `error`. `errors.Wrap(handlerFunc)` renders the envelope at the HTTP boundary.
- Named, stable errors live in `app/<module>/errors.go` (business) or `lib/<area>/errors.go` (cross-cutting, e.g. `lib/auth/errors.go`). No ad-hoc `apperr.New(...)` for cases that recur.

### Messaging
- Topic exchange `order.events` (durable, declared by `order-service`). This service's queue: `analytics-service.order-events`, bindings `order.#, payment.#`, DLQ `analytics-service.order-events.dlq`.
- At-least-once delivery. Manual ack after the handler succeeds. Dedupe via Mongo `event_ids`, never Redis `SETNX` — see `docs/system-design.md` for why.

---

## 9. Performance budget

- Every Mongo query must be backed by a declared index — check `repository/indexes.go` before adding a new query shape.
- No N+1: a handler that needs related data batches it, same rule as the Node services.
- Event handlers must be idempotent and side-effect-free beyond the one upsert — never call out to another service from inside a handler (keeps consumer throughput decoupled from downstream latency).
- `RBACCacheTTL` defaults to 5 min — tune from the observed rate of permission-catalog changes, not request volume.

---

## 10. Code style — what to avoid

- ❌ GORM/Ent/any ODM — use mongo-driver directly with typed structs.
- ❌ DDL migrations — Mongo is schemaless; indexes only.
- ❌ Storing a derived average, rate, or percentage directly — store sum+count, derive at read time.
- ❌ Collection names, field names, or any app-specific knowledge inside `pkg/`.
- ❌ `lib/` importing `app/<module>/*` directly — invert via an interface.
- ❌ Inline `type X struct{...}` inside `*.service.go`/`*.controller.go`/`*.repo.go` — goes in `types.go`.
- ❌ Returning entities or service-layer types from a controller — always a response DTO.
- ❌ A per-row Mongo call inside a loop where `BulkWrite` applies.
- ❌ Swallowing an unknown event type into the DLQ — only a handler **failure** goes to the DLQ; an unrecognized type is logged and acked.

---

## 11. Reference docs (in `docs/`)

- `docs/folder-structure.md` — annotated tree.
- `docs/system-design.md` — platform diagram, sync/async flows, failure modes, why-Mongo-dedupe-not-Redis.
- `docs/api-contracts.md` — request/response shapes, every error code.
- `docs/node-to-go-mapping.md` — **the most important doc.** TS idiom -> Go idiom, side by side, per layer.
- `docs/ai-prompts.md` — prompt templates for the homework.
- `docs/implementation-plan.md` — phases, acceptance checks.
- `plan.md` — video scope vs. homework.

---

## 12. Out of scope (do not build here)

- `agg_branch_day`, `agg_product_day`, `agg_platform_day` and their event handlers (`payment.completed`, `order.delivered`, `order.rejected`) — homework.
- The other 7 HTTP endpoints (failures, delivery-avg, active-restaurants, ...) — homework.
- `rbac.permissions_changed` consumer / cache invalidation — homework.
- `cmd/backfill-aggs/` — homework.
- Users, restaurants, products, orders, payments, RBAC permission catalog — owned by `core-service`/`order-service`.
- Any write path back to an operational data store. This service only reads events and serves read-only HTTP.
