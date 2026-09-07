package config

import (
	"fmt"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	ShutdownTimeout time.Duration   `koanf:"SHUTDOWN_TIMEOUT" validate:"required"`
	Server          ServerConfig    `koanf:",squash"`
	Cache           CacheConfig     `koanf:",squash"`
	Broker          BrokerConfig    `koanf:",squash"`
	Evaluator       EvaluatorConfig `koanf:",squash"`
}

type ServerConfig struct {
	Host    string        `koanf:"SERVER_HOST" validate:"required"`
	Port    uint16        `koanf:"SERVER_PORT" validate:"required"`
	Timeout time.Duration `koanf:"SERVER_TIMEOUT" validate:"required"`
}

type CacheConfig struct {
	Host                   string        `koanf:"CACHE_HOST" validate:"required"`
	Port                   int           `koanf:"CACHE_PORT" validate:"required"`
	ReadTimeout            time.Duration `koanf:"CACHE_READ_TIMEOUT" validate:"required"`
	WriteTimeout           time.Duration `koanf:"CACHE_WRITE_TIMEOUT" validate:"required"`
	CampaignEventIDSeenTTL time.Duration `koanf:"CACHE_CAMPAIGN_EVENT_ID_SEEN_TTL" validate:"required"`
}

const (
	UpdateResultsTopic   = "firmware.update-results"
	RolloutDecisionTopic = "rollout.decisions"
)

type BrokerConfig struct {
	Host           string        `koanf:"BROKER_HOST" validate:"required"`
	Port           int           `koanf:"BROKER_PORT" validate:"required"`
	BatchTimeout   time.Duration `koanf:"BROKER_BATCH_TIMEOUT" validate:"required"`
	Timeout        time.Duration `koanf:"BROKER_TIMEOUT" validate:"required"`
	GroupID        string        `koanf:"BROKER_GROUP_ID" validate:"required"`
	MinBytes       int           `koanf:"BROKER_MIN_BYTES" validate:"required,min=1"`
	ReaderMaxWait  time.Duration `koanf:"BROKER_READER_MAX_WAIT" validate:"required"`
	CommitTimeout  time.Duration `koanf:"BROKER_COMMIT_TIMEOUT" validate:"required"`
	DLQTimeout     time.Duration `koanf:"BROKER_DLQ_TIMEOUT" validate:"required"`
	DLQMaxAttempts int           `koanf:"BROKER_DLQ_MAX_ATTEMPTS" validate:"required,min=1"`
	Topic          string
}

type EvaluatorConfig struct {
	Frequency            time.Duration `koanf:"EVALUATOR_FREQUENCY" validate:"required"`
	RequiredStableCycles int64         `koanf:"EVALUATOR_REQUIRED_STABLE_CYCLES" validate:"required"`
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
