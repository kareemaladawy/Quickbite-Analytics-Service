# analytics-service

Go + MongoDB + RabbitMQ service that owns per-day rollups of QuickBite order activity. See `CLAUDE.md` for the full rulebook, `plan.md` for what's built vs. homework, and `docs/` for design docs.

## Prereqs

- Go 1.21+
- A running MongoDB (`mongod` or a container) and RabbitMQ (with the management plugin if you want the UI).

```bash
docker run -d --name analytics-mongo -p 27017:27017 mongo:7
docker run -d --name analytics-rabbit -p 5672:5672 -p 15672:15672 rabbitmq:3-management
```

## Setup

```bash
cp .env.example .env
go mod tidy
```

`ACCESS_SECRET` in `.env` must match whatever secret you mint tokens with (the default in `.env.example` works with `play/mint-jwt` as-is). `CORE_SERVICE_BASE_URL`/`CORE_INTERNAL_API_KEY` should point at a real `core-service`, or at `play/mock-core` for local dev (the defaults already do).

## Running it — three terminals

**Terminal 1 — mock core-service** (stands in for the real one; serves the internal RBAC permissions endpoint):

```bash
go run ./play/mock-core -port 4200 -api-key dev-internal-key
```

**Terminal 2 — the API**:

```bash
go run ./cmd/api
```

Expected boot log (one JSON line each, in order):

```
{"level":"INFO","msg":"mongo connected"}
{"level":"INFO","msg":"rabbit connected"}
{"level":"INFO","msg":"event consumer started","queue":"analytics-service.order-events","bindings":["order.#","payment.#"]}
{"level":"INFO","msg":"http listening","port":4100}
```

**Terminal 3 — publish a test event and read it back**:

```bash
go run ./play/publish-test -restaurant-id 42 -total 2500 -currency EGP
go run ./play/check-mongo -restaurant-id 42
# {"orders_count": 1, "revenue_sum": 2500, ...}

TOKEN=$(go run ./play/mint-jwt -role restaurant_user -restaurant-role owner)
curl -s -b "access_token=$TOKEN" \
  "http://localhost:4100/api/v1/analytics/restaurants/42/days?from=2026-01-01&to=2026-12-31"
# {"success":true,"data":[{"date":"...","ordersCount":1,"revenueMinor":2500,"currency":"EGP","avgOrderMinor":2500}]}
```

Publish the same `-event-id` twice to confirm dedupe (mongo stays unchanged); omit auth or pass a garbage cookie to confirm `401 UNAUTHENTICATED`.

## Build / vet

```bash
go build ./...
go vet ./...
```

## Play aids (`play/`, gitignored)

| Binary          | Purpose                                                              |
| --------------- | ---------------------------------------------------------------------- |
| `mock-core`     | Serves `GET /api/v1/internal/rbac/permissions?role=` so RBAC works without a real core-service. |
| `mint-jwt`      | Prints a signed access token matching core/order's claim shape.       |
| `publish-test`  | Publishes an `order.placed` envelope to RabbitMQ, mirroring order-service's outbox. |
| `check-mongo`   | Dumps `agg_restaurant_day` rows for a restaurant.                      |
