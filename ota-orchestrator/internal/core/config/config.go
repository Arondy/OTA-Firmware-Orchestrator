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

type Config struct {
	HTTPServer HTTPServerConfig `koanf:",squash"`
	DB         DBConfig         `koanf:",squash"`
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
	RequestTimeout        time.Duration `koanf:"REQUEST_TIMEOUT" validate:"required"`
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
