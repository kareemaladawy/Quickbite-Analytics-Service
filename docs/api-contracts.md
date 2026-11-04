# API contracts

## Conventions

- Success envelope: `{"success": true, "data": ...}` (optionally `"meta": {...}` for paginated lists — none in this slice).
- Error envelope: `{"success": false, "error": {"code": "...", "message": "...", "details"?: ...}}`.
- Money: integer minor units (e.g. `2500` = 25.00 EGP). A `currency` field sits alongside any money field.
- Timestamps: ISO-8601 UTC strings. Dates (no time component): `YYYY-MM-DD`.
- Auth: `access_token` cookie, or `Authorization: Bearer <token>` if no cookie is present. JWT HS256, same `ACCESS_SECRET` as core/order.
- RBAC: every route under `/api/v1/analytics` requires `analytics:read` (or `role: system_admin`, which bypasses the check).

---

## `GET /api/v1/analytics/restaurants/:restaurantId/days`

Day-grained order/revenue rollup for one restaurant.

### Path params

| Param | Type | Notes |
| --- | --- | --- |
| `restaurantId` | positive integer | `400 VALIDATION_ERROR` if not a positive integer. |

### Query params

| Param | Type | Required | Notes |
| --- | --- | --- | --- |
| `from` | `YYYY-MM-DD` | yes | Inclusive lower bound. |
| `to` | `YYYY-MM-DD` | yes | Inclusive upper bound. |

### Response `200`

```json
{
  "success": true,
  "data": [
    {
      "date": "2026-11-03",
      "ordersCount": 2,
      "revenueMinor": 4000,
      "currency": "EGP",
      "avgOrderMinor": 2000
    }
  ]
}
```

- Rows are ordered by `date` ascending.
- A restaurant with no activity in range returns `200` with `"data": []` — not a 404.
- `avgOrderMinor` is `revenueMinor / ordersCount`, computed in the service layer at read time, floor-divided (integer minor units) — never stored.

### Errors

| Code | Status | When |
| --- | --- | --- |
| `UNAUTHENTICATED` | 401 | No token, or token fails verification (expired, bad signature, garbage value). |
| `FORBIDDEN` | 403 | Valid token, but the role/restaurantRole lacks `analytics:read`. |
| `VALIDATION_ERROR` | 400 | `restaurantId` not a positive integer; `from`/`to` missing or not `YYYY-MM-DD`. |
| `ANALYTICS_INVALID_DATE_RANGE` | 400 | `from` is after `to`. |
| `INTERNAL_ERROR` | 500 | Unexpected failure (Mongo down, etc.) — message is generic, detail is logged server-side only. |

---

## `GET /api/v1/analytics/restaurants/:restaurantId/failure-rate`

Day-grained rejection rate for one restaurant — derived from `agg_restaurant_day`'s `orders_count`/`orders_rejected_count`, no new collection.

### Path/query params

Same as the `/days` endpoint above (`restaurantId` path param, `from`/`to` query params, same validation).

### Response `200`

```json
{
  "success": true,
  "data": [
    {"date": "2026-11-03", "ordersCount": 10, "ordersRejectedCount": 1, "failureRate": 0.1}
  ]
}
```

- `failureRate` is `ordersRejectedCount / ordersCount`, computed at read time, `0` when `ordersCount` is `0`.

### Errors

Same set as `/days` (`UNAUTHENTICATED`, `FORBIDDEN`, `VALIDATION_ERROR`, `ANALYTICS_INVALID_DATE_RANGE`, `INTERNAL_ERROR`).

---

## `GET /api/v1/analytics/restaurants/:restaurantId/delivery-avg`

Day-grained average delivery duration for one restaurant — derived from `agg_restaurant_day`'s `delivery_ms_sum`/`delivery_ms_count`, no new collection.

### Response `200`

```json
{
  "success": true,
  "data": [
    {"date": "2026-11-03", "deliveredCount": 4, "avgDeliveryMs": 1800000}
  ]
}
```

### Errors

Same set as `/days`.

---

## `GET /api/v1/analytics/restaurants/active`

Every restaurant with at least one `order.placed` in `[from, to]`, with its order count across that window — a `$group` aggregation over `agg_restaurant_day`, no new collection.

### Query params

`from`/`to`, same shape as `/days`. No path param.

### Response `200`

```json
{
  "success": true,
  "data": [
    {"restaurantId": 42, "ordersCount": 7}
  ]
}
```

- Ordered by `restaurantId` ascending. No activity in range returns `200 []`.

### Errors

`UNAUTHENTICATED`, `FORBIDDEN`, `VALIDATION_ERROR`, `ANALYTICS_INVALID_DATE_RANGE`, `INTERNAL_ERROR`.

---

## `GET /api/v1/analytics/branches/:branchId/days`

Day-grained order/revenue rollup for one branch — the branch-dimension analogue of the restaurant `/days` endpoint, backed by `agg_branch_day`.

### Path params

| Param | Type | Notes |
| --- | --- | --- |
| `branchId` | positive integer | `400 VALIDATION_ERROR` if not a positive integer. |

### Response `200`

```json
{
  "success": true,
  "data": [
    {"date": "2026-11-03", "ordersCount": 2, "revenueMinor": 4000, "currency": "EGP", "avgOrderMinor": 2000}
  ]
}
```

### Errors

Same set as `/days`, with `VALIDATION_ERROR` also covering a non-positive-integer `branchId`.

---

## `GET /api/v1/analytics/branches/:branchId/failure-rate`

Branch-scoped analogue of the restaurant failure-rate endpoint — derived from `agg_branch_day`, no new collection.

### Response `200`

```json
{
  "success": true,
  "data": [
    {"date": "2026-11-03", "ordersCount": 2, "ordersRejectedCount": 0, "failureRate": 0}
  ]
}
```

### Errors

Same set as `/branches/:branchId/days`.

---

## `GET /api/v1/analytics/products/:productId/days`

Day-grained quantity/revenue rollup for one product, backed by `agg_product_day` — one row per `(product_id, date)`, fed by `order.placed`'s line items via a single `BulkWrite`.

### Path params

| Param | Type | Notes |
| --- | --- | --- |
| `productId` | positive integer | `400 VALIDATION_ERROR` if not a positive integer. |

### Response `200`

```json
{
  "success": true,
  "data": [
    {"date": "2026-11-03", "ordersCount": 3, "quantitySum": 5, "revenueMinor": 1250, "currency": "EGP", "avgUnitPriceMinor": 250}
  ]
}
```

- `ordersCount` counts line items referencing this product, not distinct orders (one order can only reference a product once per line item in this slice).
- `avgUnitPriceMinor` is `revenueMinor / quantitySum`, computed at read time.

### Errors

Same set as `/days`, with `VALIDATION_ERROR` also covering a non-positive-integer `productId`.

---

## `GET /api/v1/analytics/platform/days`

Day-grained, platform-wide rollup with no restaurant/branch/product dimension, backed by `agg_platform_day` — the only aggregate `payment.completed` feeds (its payload carries no `restaurantId`).

### Response `200`

```json
{
  "success": true,
  "data": [
    {
      "date": "2026-11-03",
      "ordersCount": 120,
      "revenueMinor": 240000,
      "currency": "EGP",
      "avgOrderMinor": 2000,
      "ordersRejectedCount": 6,
      "failureRate": 0.05,
      "deliveredCount": 110,
      "avgDeliveryMs": 1620000,
      "paymentsCompletedCount": 95,
      "paymentsCompletedSum": 190000
    }
  ]
}
```

### Errors

`UNAUTHENTICATED`, `FORBIDDEN`, `VALIDATION_ERROR`, `ANALYTICS_INVALID_DATE_RANGE`, `INTERNAL_ERROR`.

---

## `GET /health`

Liveness probe. No auth. Not under `/api/v1` (infra convention, not a versioned business endpoint).

### Response `200`

```json
{"success": true, "data": {"status": "ok"}}
```

---

## Error code reference

| Code | Status | Source |
| --- | --- | --- |
| `UNAUTHENTICATED` | 401 | `lib/auth/errors.go` |
| `FORBIDDEN` | 403 | `lib/auth/errors.go` |
| `VALIDATION_ERROR` | 400 | Dynamic per-request (query/path param shape) |
| `ANALYTICS_INVALID_DATE_RANGE` | 400 | `app/analytics/errors.go` |
| `INTERNAL_ERROR` | 500 | `lib/errors/apperror.go` (fallback wrap of any non-`AppError`) |
