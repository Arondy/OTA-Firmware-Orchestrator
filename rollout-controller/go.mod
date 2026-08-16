module github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller

go 1.26.0

require (
	connectrpc.com/connect v1.20.0
	github.com/Arondy/OTA-Firmware-Orchestrator/api/gen v0.0.1
	github.com/go-playground/validator/v10 v10.30.3
	github.com/google/uuid v1.6.0
	github.com/knadh/koanf v1.5.0
	github.com/knadh/koanf/providers/env/v2 v2.0.0
	github.com/knadh/koanf/v2 v2.3.5
	github.com/redis/go-redis/v9 v9.22.0
	github.com/segmentio/kafka-go v0.4.51
	go.uber.org/zap v1.28.0
	golang.org/x/sync v0.22.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.15 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/leodido/go-urn v1.5.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/Arondy/OTA-Firmware-Orchestrator/api/gen => ../api/gen
