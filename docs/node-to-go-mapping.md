# Node → Go mapping

This is the doc to reach for when you (or an AI) are porting a new piece of `order-service` into this codebase. Each section shows the Node idiom from `order-service`/`core-service` next to its Go equivalent actually used in this service, then explains *why* the translation looks the way it does, not just *that* it does.

---

## Config

**Node** (`order-service/src/lib/config/env.ts`) — a `zod` schema, parsed once, exported as a plain object other files import:

```ts
const baseSchema = z.object({
  PORT: z.string().default("4000"),
  ACCESS_SECRET: z.string(),
});
const parsed = baseSchema.parse(process.env);
export const env = { port: Number(parsed.PORT), jwt: { accessSecret: parsed.ACCESS_SECRET } };
```

**Go** (`lib/config/env.go`) — a struct with `env` tags, parsed by `caarlos0/env`:

```go
type Env struct {
    Port         int    `env:"PORT" envDefault:"4100"`
    AccessSecret string `env:"ACCESS_SECRET,required"`
}

func Load() (Env, error) {
    _ = godotenv.Load()
    var cfg Env
    if err := env.Parse(&cfg); err != nil {
        return Env{}, err
    }
    return cfg, nil
}
```

The shape is identical: one validated struct, one `Load()`/`.parse()` call, no env var read anywhere else in the codebase. The difference is *where* required-ness and defaults live — `zod` methods vs. struct tags — but the rule ("every new env var goes in this one file") carries over exactly.

---

## Logger

**Node** — a hand-rolled `Logger` class wrapping `console.log(JSON.stringify(...))`.

**Go** (`lib/logger/logger.go`) — stdlib `log/slog` with a JSON handler:

```go
var base = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

func FromContext(ctx context.Context) *slog.Logger {
    if id := appcontext.CorrelationID(ctx); id != "" {
        return base.With("correlation_id", id)
    }
    return base
}
```

Go's stdlib already gives you structured JSON logging — don't reach for a third-party logger (zap, zerolog) just because Node needed a custom class. `slog.With(...)` returning a new logger scoped with extra fields is the direct analogue of passing `metadata` into every `logger.info(msg, metadata)` call.

---

## Auth (JWT)

**Node** (`lib/auth/jwt.ts`):

```ts
export function verifyAccessToken(token: string): JWTPayload {
    try {
        const decoded = jwt.verify(token, env.jwt.accessSecret) as jwt.JwtPayload & JWTPayload;
        return { userId: decoded.userId, role: decoded.role, ... };
    } catch {
        throw NotAuthenticated;
    }
}
```

**Go** (`lib/auth/jwt.go`):

```go
func VerifyAccessToken(token, secret string) (Claims, error) {
    parsed := &accessTokenClaims{}
    _, err := jwt.ParseWithClaims(token, parsed, func(t *jwt.Token) (any, error) {
        if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, jwt.ErrSignatureInvalid
        }
        return []byte(secret), nil
    })
    if err != nil {
        return Claims{}, ErrNotAuthenticated
    }
    return Claims{UserID: parsed.UserID, Role: parsed.Role, ...}, nil
}
```

`throw`/`try`/`catch` becomes `(value, error)` and an `if err != nil` check — this is the single most common translation in the whole codebase, see "Common gotchas" below. Note the explicit algorithm check (`SigningMethodHMAC`) — `jsonwebtoken` defaults to rejecting `alg: none`/mismatched algorithms for you; `golang-jwt` makes you check in the keyfunc. Skipping that check is a real vulnerability (algorithm-confusion attacks), not a style nit.

---

## Errors

**Node** (`lib/error/AppError.ts` + a module's `errors.ts`):

```ts
export class AppError extends Error {
    constructor(public message: string, public statusCode = 500, ...) { super(message); }
}
export const NotAuthenticated = new AppError("User not authenticated", 401);
```

**Go** (`lib/errors/apperror.go` + a module's `errors.go`):

```go
type AppError struct {
    Code       string
    Message    string
    StatusCode int
    Details    any
}
func (e *AppError) Error() string { return e.Message }

func New(code, message string, statusCode int) *AppError {
    return &AppError{Code: code, Message: message, StatusCode: statusCode}
}

var ErrNotAuthenticated = New("UNAUTHENTICATED", "User not authenticated", http.StatusUnauthorized)
```

A Go error is any type with an `Error() string` method — there's no `extends Error`. `*AppError` is the error; `errors.As(err, &appErr)` (stdlib) is the Go analogue of `err instanceof AppError`. One addition vs. Node: this service's `AppError` carries a `Code` field (`"UNAUTHENTICATED"`), because the brief's envelope contract needs a stable machine-readable code, which order-service's own `{error, details}` envelope doesn't have — see "When to break the mapping" below.

---

## Error-handling middleware

**Node** — Express's 4-arg `errorHandler(err, req, res, next)`, registered last, catches anything a controller `throw`s (Express auto-catches sync throws; async ones need `next(err)` or a wrapper).

**Go** — there's no automatic "catch what the handler threw." Instead, every handler has signature `func(w, r) error`, and `errors.Wrap` converts that into the `http.HandlerFunc` chi actually wants:

```go
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func Wrap(h HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if err := h(w, r); err != nil {
            // render the envelope
        }
    }
}
```

Every route is mounted as `router.Get("/path", errors.Wrap(controller.Method))`. This is the biggest structural difference from Express: Go has no exception-based control flow, so "the error middleware runs after the handler" has to be *this* function wrapping, not a middleware positioned last in a chain.

---

## DTOs / validation

**Node** — `class-validator` decorators + `validateBody(DTO, req.body)`:

```ts
export class CreateOrderRequestDTO {
    @IsString() restaurantId: string;
}
```

**Go** — struct tags + `go-playground/validator`:

```go
type GetRestaurantDaysQuery struct {
    From string `validate:"required,datetime=2006-01-02"`
    To   string `validate:"required,datetime=2006-01-02"`
}
// controller:
if err := c.validate.Struct(query); err != nil { return apperr.New("VALIDATION_ERROR", ...) }
```

Decorators and struct tags play the same role (metadata attached to a field, read by a library at runtime via reflection). The difference that bites people: Go struct tags are **strings parsed at runtime**, not type-checked at compile time — a typo in a tag (`validate:"requried"`) fails silently (the tag is just ignored), where a TypeScript decorator typo is a compile error. Always smoke-test a new validation tag with a request that should fail.

---

## Response DTOs

**Node** (`dto/*.response.dto.ts`):

```ts
export class OrderResponseDTO {
    id!: string;
    static from(order: Order): OrderResponseDTO {
        const dto = new OrderResponseDTO();
        dto.id = order.publicId;
        return dto;
    }
}
```

**Go** (`dto/days.response.go`):

```go
type RestaurantDayResponse struct {
    Date string `json:"date"`
}

func RestaurantDayResponseFrom(row analytics.RestaurantDayRow) RestaurantDayResponse {
    return RestaurantDayResponse{Date: row.Date}
}
```

Same shape, same rule ("controllers never return an entity/service type directly") — a static factory method becomes a plain function, because Go doesn't have classes with static methods, just functions in a package.

---

## Repositories

**Node** (Knex, functions not classes):

```ts
export async function findOrderById(id: number, conn: Knex = db): Promise<Order | null> { ... }
```

**Go** (mongo-driver, methods on a struct wrapping a collection):

```go
type RestaurantDayRepo struct { collection *mongo.Collection }

func (r *RestaurantDayRepo) FindByRestaurantAndDateRange(restaurantID int64, from, to string) ([]entity.RestaurantDay, error) { ... }
```

Go repositories are structs with methods, not free functions, because the struct is how you hold the `*mongo.Collection` without a global. The *rule* carries over unchanged: repositories are the only place the driver appears, and a service composes repositories rather than reaching into another module's.

One real difference: Knex's `conn: Knex = db` default-parameter trick for transaction composition has **no Mongo equivalent in this slice** — there's no multi-document transaction here, because every write is a single associative `$inc` upsert. If a future aggregate genuinely needs a multi-document transaction, pass a `mongo.SessionContext` explicitly (Go has no default-parameter sugar to hide it).

---

## Services

**Node** (`@injectable()` class, DI-resolved):

```ts
@injectable()
export class OrderService {
    constructor(@inject(TOKENS.OrderRepo) private repo: OrderRepo) {}
}
```

**Go** (plain struct, explicit constructor):

```go
type AnalyticsService struct {
    restaurantDayRepo *repository.RestaurantDayRepo
}

func NewAnalyticsService(restaurantDayRepo *repository.RestaurantDayRepo) *AnalyticsService {
    return &AnalyticsService{restaurantDayRepo: restaurantDayRepo}
}
```

No DI container, no decorators, no tokens. `NewXxx(deps...) *Xxx` constructor functions, called explicitly from `lib/boot/boot.go`, are the entire DI story in this codebase. This isn't a workaround for Go lacking `tsyringe` — it's the deliberate choice the brief locks in ("DI: NO framework"), because explicit wiring is easier to trace than a container's implicit resolution once you're reading code you didn't write.

---

## Controllers

**Node** — arrow-function class properties (to preserve `this` when Express calls them):

```ts
getOrder = async (req: Request, res: Response) => {
    const order = await this.service.getOrder(Number(req.params.id));
    sendSuccess(res, OrderResponseDTO.from(order));
};
```

**Go** — ordinary methods; no `this`-binding problem exists because Go methods are called as `receiver.Method`, never detached and passed around the way `this.getOrder` can be in JS:

```go
func (c *AnalyticsController) GetRestaurantDays(w http.ResponseWriter, r *http.Request) error {
    rows, err := c.service.GetRestaurantDays(...)
    if err != nil { return err }
    resp.SendSuccess(w, http.StatusOK, dto.RestaurantDayResponsesFrom(rows))
    return nil
}
```

The arrow-function trick in the Node code is a workaround for a JS footgun (losing `this` when a method is passed as a bare function reference) that doesn't exist in Go — don't port the pattern, just write a normal method.

---

## Routes

**Node** (Express):

```ts
router.get("/restaurants/:restaurantId/days", authenticate, rbac({resource: "analytics", action: "read"}), controller.getDays);
```

**Go** (chi):

```go
sub.Use(auth.Authenticate(accessSecret))
sub.Use(rbac.Require(rbacCache, analytics.PermAnalyticsRead))
sub.Get("/restaurants/{restaurantId}/days", apperr.Wrap(c.GetRestaurantDays))
```

chi's route-group `.Route("/prefix", func(r chi.Router) {...})` + `.Use(middleware)` is a direct analogue of Express's `router.use(...)` scoped to a sub-router. Path params: Express's `:id` becomes chi's `{id}`, read with `chi.URLParam(r, "id")` instead of `req.params.id`.

---

## Messaging (publish/consume)

**Node** (`amqplib`, confirm channel):

```ts
await ch.publish(exchange, routingKey, Buffer.from(JSON.stringify(payload)), { persistent: true }, cb);
```

**Go** (`amqp091-go`, deferred confirms):

```go
confirm, err := b.pubCh.PublishWithDeferredConfirm(exchange, routingKey, false, false, amqp.Publishing{
    DeliveryMode: amqp.Persistent,
    Body:         body,
})
ok := confirm.Wait()
```

Both wait for the broker's publisher-confirm before considering the publish successful — `amqplib`'s callback-based confirm vs. `amqp091-go`'s `DeferredConfirmation.Wait()` are the same wait, different API shape (Go's is blocking-by-default; no callback needed because there's no event loop to avoid blocking).

**Consuming** — Node's manual-ack pattern:

```ts
ch.consume(queue, (msg) => { ... ch.ack(msg) ... }, { noAck: false });
```

**Go** — same manual-ack shape, dispatched through a goroutine per queue rather than an event-loop callback:

```go
deliveries, _ := ch.Consume(queue, "", false, false, false, false, nil)
go func() {
    for d := range deliveries {
        if err := handler(msg); err != nil { d.Nack(false, false) } else { d.Ack(false) }
    }
}()
```

Node's single-threaded event loop processes deliveries one at a time by default; Go's goroutine does the same here (one `for range` loop, not fanned out across workers) — if you want concurrent processing of a high-volume queue, that's an explicit worker-pool you'd add, not something either language gives you for free.

---

## `Promise<T>` → `(T, error)`

The one idiom that shows up in literally every file. Node:

```ts
async function getPermissions(role: string): Promise<string[]> {
    if (cached) return JSON.parse(cached);
    const res = await coreClient.request(...);
    return res.data.permissions;
}
```

Go:

```go
func (c *Cache) getPermissions(role string) ([]string, error) {
    if cached { return cached, nil }
    permissions, err := c.fetcher.FetchPermissions(role)
    if err != nil { return nil, err }
    return permissions, nil
}
```

There is no `await` keyword because Go function calls are synchronous by default (goroutines + channels are the opt-in concurrency primitive, not every function call). `try/catch` becomes `if err != nil { return ..., err }` at every call site — tedious compared to a single `catch` at the end of an `async` chain, but it means every call site explicitly decides what to do with a failure, instead of it propagating invisibly until some `catch` block.

---

## Common gotchas when porting

1. **`if err != nil` is not optional, ever.** Unlike an unhandled Promise rejection (which at least crashes loudly, or triggers an `unhandledRejection` event), an ignored Go error is silently `nil`/zero-valued and the bug surfaces somewhere downstream, much harder to trace back. Never write `result, _ := someCall()` outside of a genuinely-infallible case (and even then, prefer handling it).
2. **Struct tags are stringly-typed.** A typo in a `json:"..."`, `bson:"..."`, or `validate:"..."` tag is not a compile error — it silently does nothing (or, for `json`/`bson`, silently uses the Go field name instead of your intended wire name). Grep for the exact field name in a request/response body after writing a new DTO.
3. **Zero values are not `undefined`.** A Go `int64` field that was never set is `0`, not missing — `if restaurantID == 0` can't distinguish "not provided" from "provided as zero." Use a pointer (`*int64`) when "absent" needs to be distinguishable from the zero value (see `Claims.RestaurantID` in `lib/auth/types.go`).
4. **Interfaces are satisfied implicitly.** There's no `implements` keyword — a type satisfies an interface just by having the right methods. This is how `app/analytics/repository.EventIDsRepo` satisfies `lib/coreevents.EventDeduper` without `lib/coreevents` ever importing `app/analytics` (the layering rule's escape hatch). It also means a method-name typo doesn't fail where you'd expect — it fails at the *assignment* site (`var _ coreevents.EventDeduper = (*EventIDsRepo)(nil)`), not at the method definition.
5. **`nil` map/slice reads are safe; `nil` map writes panic.** Reading a `nil` map returns the zero value (no error); writing to one panics. Always initialize with `make(...)` (see `rbac.Cache.entries`) before writing.
6. **No class inheritance, no decorators, no metadata reflection beyond struct tags.** If you're tempted to port a Node mixin/decorator pattern, stop — the Go answer is almost always composition (embed a struct, or hold an interface field) or a plain function, not a framework feature that doesn't exist here.

---

## When to break the mapping

The mapping is a starting point, not a law. Break it when:

- **The Node pattern exists only to work around a JS limitation that Go doesn't have.** The arrow-function-as-class-property trick (controllers) is the clearest example — porting it to Go would be copying a workaround for a problem Go doesn't have.
- **The brief's explicit contract differs from what order-service actually does.** This service's error envelope (`{success, error: {code, message}}`) is richer than order-service's own (`{error, details}`) because the brief specifies the richer shape — don't "fix" this service to match order-service when the written spec says otherwise; flag the discrepancy in a comment (see `lib/errors/apperror.go`) instead of silently picking one.
- **Go's stdlib already solves the problem** and reaching for a third-party library or a hand-rolled abstraction would just be re-deriving `log/slog`, `net/http`, or `context` badly. Node's ecosystem leans on userland packages for things Go's stdlib already covers well; check the stdlib first.
- **The concurrency model genuinely differs.** Node's single-threaded event loop means "two requests never truly run at the same instant"; Go's goroutines mean they can. Anywhere this service holds mutable shared state across requests (see `lib/rbac/cache.go`'s `sync.RWMutex`), that's a deliberate adaptation, not an oversight — the Node code has no equivalent lock because it's never needed one.
