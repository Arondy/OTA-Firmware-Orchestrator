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

const (
	BatchSize         = 1000
	RandSeed          = 42
	CleanupBeforeSeed = true

	NumDevices = 5000
	NumStages  = 3

	NumLoadCampaigns = 50

	SuccessPercent = 98
	FailurePercent = 1

	DeviceModel   = "load-test-model"
	BaseVersion   = "1.0.0"
	TargetVersion = "1.1.0"

	UpToDatePercent = 10

	DecommissionedPercent = 5

	StageTargetPercent    = 100
	StageMinSampleSize    = 20
	StageSuccessThreshold = 0.95

	// Отдельные модели нужны из-за one_running_campaign_per_model:
	// только одна running-кампания на модель.
	DemoModelRunning    = "demo-sensor-v1"
	DemoModelPaused     = "demo-sensor-v2"
	DemoModelDraft      = "demo-gateway-x1"
	DemoModelCompleted  = "demo-gateway-x2"
	DemoModelRolledBack = "demo-camera-z1"
	DemoModelOrphan     = "demo-orphan-m1"

	DemoRunningBaseVersion      = "1.0.0"
	DemoRunningTargetVersion    = "2.0.0"
	DemoPausedBaseVersion       = "1.0.0"
	DemoPausedTargetVersion     = "2.1.0"
	DemoDraftBaseVersion        = "3.0.0"
	DemoDraftTargetVersion      = "3.1.0"
	DemoCompletedBaseVersion    = "3.0.0"
	DemoCompletedTargetVersion  = "4.0.0"
	DemoRolledBackBaseVersion   = "5.0.0"
	DemoRolledBackTargetVersion = "5.1.0"
	DemoOrphanVersion           = "9.0.0"

	DemoDevicesPerCampaign  = 8
	DemoAttemptsPerCampaign = 30
	DemoOrphanDevices       = 3
)

type Config struct {
	DB    DBConfig    `koanf:",squash"`
	Cache CacheConfig `koanf:",squash"`
}

type DBConfig struct {
	Host     string `koanf:"DB_HOST" validate:"required"`
	Port     int    `koanf:"DB_PORT" validate:"required"`
	User     string `koanf:"DB_USER" validate:"required"`
	Password string `koanf:"DB_PASSWORD" validate:"required"`
	Name     string `koanf:"DB_NAME" validate:"required"`
	SSLMode  string `koanf:"DB_SSL_MODE" validate:"required"`
}

func (c DBConfig) ConnString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

type CacheConfig struct {
	Host                 string        `koanf:"CACHE_HOST" validate:"required"`
	Port                 int           `koanf:"CACHE_PORT" validate:"required"`
	DeviceCheckinDataTTL time.Duration `koanf:"CACHE_DEVICE_CHECKIN_DATA_TTL" validate:"required"`
}

func LoadConfig() *Config {
	k := koanf.New(".")

	if err := k.Load(file.Provider(".env"), dotenv.Parser()); err != nil && !os.IsNotExist(err) {
		panic(fmt.Sprintf("failed to read .env: %s", err))
	}

	_ = k.Load(env.Provider(".", env.Opt{}), nil)

	var config Config
	if err := k.UnmarshalWithConf("", &config, koanf.UnmarshalConf{
		Tag:       "koanf",
		FlatPaths: true,
	}); err != nil {
		panic(err)
	}

	if err := validator.New().Struct(&config); err != nil {
		panic(fmt.Sprintf("Invalid configuration: %s", err))
	}

	return &config
}
