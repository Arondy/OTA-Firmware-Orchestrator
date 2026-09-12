package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const RequestIDHeader = "x-request-id"

type CtxKeyRequestID struct{}

type Config struct {
	ShutdownTimeout   time.Duration           `koanf:"SHUTDOWN_TIMEOUT" validate:"required"`
	HTTPServer        HTTPServerConfig        `koanf:",squash"`
	DB                DBConfig                `koanf:",squash"`
	Cache             CacheConfig             `koanf:",squash"`
	Broker            BrokerConfig            `koanf:",squash"`
	RolloutController RolloutControllerConfig `koanf:",squash"`
}

type HTTPServerConfig struct {
	Host    string        `koanf:"HTTP_SERVER_HOST" validate:"required"`
	Port    uint16        `koanf:"HTTP_SERVER_PORT" validate:"required"`
	Timeout time.Duration `koanf:"HTTP_SERVER_TIMEOUT" validate:"required"`
}

type DBConfig struct {
	Host                  string        `koanf:"DB_HOST" validate:"required"`
	Port                  int           `koanf:"DB_PORT" validate:"required"`
	User                  string        `koanf:"DB_USER" validate:"required"`
	Password              string        `koanf:"DB_PASSWORD" validate:"required"`
	Name                  string        `koanf:"DB_NAME" validate:"required"`
	SSLMode               string        `koanf:"DB_SSL_MODE" validate:"required"`
	MaxConns              int32         `koanf:"DB_MAX_CONNS"`
	MinConns              int32         `koanf:"DB_MIN_CONNS"`
	MaxConnLifetime       time.Duration `koanf:"DB_MAX_CONN_LIFETIME"`
	MaxConnIdleTime       time.Duration `koanf:"DB_MAX_CONN_IDLE_TIME"`
	HealthCheckPeriod     time.Duration `koanf:"DB_HEALTH_CHECK_PERIOD"`
	MaxConnLifetimeJitter time.Duration `koanf:"DB_MAX_CONN_LIFETIME_JITTER"`
	RequestTimeout        time.Duration `koanf:"DB_REQUEST_TIMEOUT" validate:"required"`
	PaginationLimit       int           `koanf:"DB_PAGINATION_LIMIT" validate:"required,gt=0"`
}

func (c DBConfig) ConnString() string {
	var b strings.Builder
	fmt.Fprintf(&b, "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)

	if c.MaxConns > 0 {
		fmt.Fprintf(&b, " pool_max_conns=%d", c.MaxConns)
	}
	if c.MinConns > 0 {
		fmt.Fprintf(&b, " pool_min_conns=%d", c.MinConns)
	}
	if c.MaxConnLifetime > 0 {
		fmt.Fprintf(&b, " pool_max_conn_lifetime=%s", c.MaxConnLifetime)
	}
	if c.MaxConnIdleTime > 0 {
		fmt.Fprintf(&b, " pool_max_conn_idle_time=%s", c.MaxConnIdleTime)
	}
	if c.HealthCheckPeriod > 0 {
		fmt.Fprintf(&b, " pool_health_check_period=%s", c.HealthCheckPeriod)
	}
	if c.MaxConnLifetimeJitter > 0 {
		fmt.Fprintf(&b, " pool_max_conn_lifetime_jitter=%s", c.MaxConnLifetimeJitter)
	}

	return b.String()
}

type CacheConfig struct {
	Host                 string        `koanf:"CACHE_HOST" validate:"required"`
	Port                 int           `koanf:"CACHE_PORT" validate:"required"`
	ReadTimeout          time.Duration `koanf:"CACHE_READ_TIMEOUT" validate:"required"`
	WriteTimeout         time.Duration `koanf:"CACHE_WRITE_TIMEOUT" validate:"required"`
	DeviceCheckinDataTTL time.Duration `koanf:"CACHE_DEVICE_CHECKIN_DATA_TTL" validate:"required"`
}

const (
	UpdateResultsTopic   = "firmware.update-results"
	CheckinsTopic        = "device.checkins"
	RolloutDecisionTopic = "rollout.decisions"
)

type BrokerConfig struct {
	Host           string        `koanf:"BROKER_HOST" validate:"required"`
	Port           int           `koanf:"BROKER_PORT" validate:"required"`
	BatchTimeout   time.Duration `koanf:"BROKER_BATCH_TIMEOUT" validate:"required"`
	Timeout        time.Duration `koanf:"BROKER_TIMEOUT" validate:"required"`
	BufferSize     int           `koanf:"BROKER_BUFFER_SIZE" validate:"required,min=1"`
	GroupID        string        `koanf:"BROKER_GROUP_ID" validate:"required"`
	MinBytes       int           `koanf:"BROKER_MIN_BYTES" validate:"required,min=1"`
	ReaderMaxWait  time.Duration `koanf:"BROKER_READER_MAX_WAIT" validate:"required"`
	CommitTimeout  time.Duration `koanf:"BROKER_COMMIT_TIMEOUT" validate:"required"`
	DLQTimeout     time.Duration `koanf:"BROKER_DLQ_TIMEOUT" validate:"required"`
	DLQMaxAttempts int           `koanf:"BROKER_DLQ_MAX_ATTEMPTS" validate:"required,min=1"`
	Topic          string
}

type RolloutControllerConfig struct {
	Scheme  string        `koanf:"ROLLOUT_CONTROLLER_SCHEME" validate:"required"`
	Host    string        `koanf:"ROLLOUT_CONTROLLER_HOST" validate:"required"`
	Port    int           `koanf:"ROLLOUT_CONTROLLER_PORT" validate:"required"`
	Timeout time.Duration `koanf:"ROLLOUT_CONTROLLER_TIMEOUT" validate:"required"`
}

func LoadConfig() *Config {
	k := koanf.New(".")

	if err := k.Load(file.Provider(".env"), dotenv.Parser()); err != nil && !os.IsNotExist(err) {
		panic(fmt.Sprintf("failed to read .env: %s", err))
	}
	_ = k.Load(file.Provider("config/.env"), dotenv.Parser())
	_ = k.Load(env.Provider(".", env.Opt{}), nil)

	var config Config
	if err := k.UnmarshalWithConf("", &config, koanf.UnmarshalConf{
		Tag:       "koanf",
		FlatPaths: true,
	}); err != nil {
		panic(err)
	}

	validate := validator.New()
	if err := validate.Struct(&config); err != nil {
		panic(fmt.Sprintf("Invalid configuration: %s", err))
	}

	return &config
}
