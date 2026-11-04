// Package config is the Go analogue of order-service's lib/config/env.ts —
// a struct with env-parsing tags (caarlos0/env instead of zod), one Load()
// call, one exported Env the rest of the service reads from.
package config

import (
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Env struct {
	Port    int    `env:"PORT" envDefault:"4100"`
	NodeEnv string `env:"NODE_ENV" envDefault:"development"`

	AccessSecret string `env:"ACCESS_SECRET,required"`

	MongoURI string `env:"MONGO_URI" envDefault:"mongodb://localhost:27017"`
	MongoDB  string `env:"MONGO_DB" envDefault:"quickbite_analytics"`

	RabbitMQURL                 string   `env:"RABBITMQ_URL" envDefault:"amqp://guest:guest@localhost:5672"`
	RabbitMQOrderEventsExchange string   `env:"RABBITMQ_ORDER_EVENTS_EXCHANGE" envDefault:"order.events"`
	RabbitMQQueue               string   `env:"RABBITMQ_QUEUE" envDefault:"analytics-service.order-events"`
	RabbitMQBindings            []string `env:"RABBITMQ_BINDINGS" envDefault:"order.#,payment.#" envSeparator:","`
	RabbitMQDLX                 string   `env:"RABBITMQ_DLX" envDefault:"order.events.dlx"`
	RabbitMQDLQ                 string   `env:"RABBITMQ_DLQ" envDefault:"analytics-service.order-events.dlq"`
	RabbitMQPrefetch            int      `env:"RABBITMQ_PREFETCH" envDefault:"32"`

	// Second consumer, bound to core-service's own outbound exchange (not
	// order.events) — carries rbac.permissions_changed. Deliberately a
	// dedicated queue, not a second binding on RabbitMQQueue above: the two
	// events are published on different exchanges by different services
	// (mirrors order-service's own RABBITMQ_CORE_EVENTS_* split from
	// RABBITMQ_ORDER_EVENTS_EXCHANGE).
	RabbitMQCoreEventsExchange string   `env:"RABBITMQ_CORE_EVENTS_EXCHANGE" envDefault:"core.events"`
	RabbitMQCoreEventsQueue    string   `env:"RABBITMQ_CORE_EVENTS_QUEUE" envDefault:"analytics-service.core-events"`
	RabbitMQCoreEventsBindings []string `env:"RABBITMQ_CORE_EVENTS_BINDINGS" envDefault:"rbac.#" envSeparator:","`
	RabbitMQCoreEventsDLX      string   `env:"RABBITMQ_CORE_EVENTS_DLX" envDefault:"core.events.dlx"`
	RabbitMQCoreEventsDLQ      string   `env:"RABBITMQ_CORE_EVENTS_DLQ" envDefault:"analytics-service.core-events.dlq"`

	CoreServiceBaseURL string `env:"CORE_SERVICE_BASE_URL,required"`
	CoreInternalAPIKey string `env:"CORE_INTERNAL_API_KEY,required"`

	RBACCacheTTLSec int `env:"RBAC_CACHE_TTL_SEC" envDefault:"300"`

	EventIDsTTLDays int `env:"EVENT_IDS_TTL_DAYS" envDefault:"7"`

	CorsOrigins []string `env:"CORS_ORIGINS" envDefault:"http://localhost:3000" envSeparator:","`
}

func (e Env) IsProduction() bool {
	return e.NodeEnv == "production"
}

func (e Env) RBACCacheTTL() time.Duration {
	return time.Duration(e.RBACCacheTTLSec) * time.Second
}

func (e Env) EventIDsTTL() time.Duration {
	return time.Duration(e.EventIDsTTLDays) * 24 * time.Hour
}

// Load reads .env (if present — never fatal when absent, e.g. in prod where
// real env vars are injected by the platform) then parses process env into
// Env via struct tags, the Go analogue of zod's baseSchema.parse(process.env).
// Load reads .env (if present — never fatal when absent; in prod real env
// vars are injected by the platform) then parses process env into Env.
// Unlike order-service's env.ts, there is no process.env.TZ = "UTC" step:
// Go has no process-wide TZ knob, so every call site uses time.Now().UTC()
// explicitly instead (see app/analytics's date handling).
func Load() (Env, error) {
	_ = godotenv.Load()

	var cfg Env
	if err := env.Parse(&cfg); err != nil {
		return Env{}, err
	}
	return cfg, nil
}
