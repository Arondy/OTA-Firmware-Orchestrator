# IMPLEMENTATION.md

Внутреннее устройство OTA-Firmware-Orchestrator. Пользовательский обзор - в [README.md](../README.md).

## Слои и точки входа

- `ota-orchestrator/` - основной HTTP-сервис. Вход `cmd/ota-orchestrator/main.go`, сборка зависимостей `internal/core/app.go`. Слои выстроены как `transport/http` из handlers и DTO → `service/{campaign,device,update,firmware}` → `repository/{postgres,redis,kafka}` плюс `transport/kafka` с consumer'ом `rollout.decisions`. Домен - `internal/core/domain`.
- `rollout-controller/` - stateless-сервис. Вход `cmd/rollout-controller/main.go`, сборка `internal/core/app.go`. Слои: `transport/{kafka,connect}` → `service/{results,evaluator}` → `repository/{redis,kafka}`. Порт 8090, Connect-RPC поверх HTTP/JSON, а не нативный gRPC-фрейминг.
- `api/gen/` - отдельный Go-модуль со сгенерированным protobuf/Connect-кодом из `api/proto/**`. Подключён через `replace` в обоих `go.mod`. Правится только командой `task gen`, для неё нужен `buf`; руками не трогать. Пакеты: `api/gen/rollout/v1` с CampaignService, `api/gen/health/v1` с HealthService.
- Модули связаны `replace`-директивами без go workspace: `go build` / `go test` запускать из папки каждого модуля.

Транзакции: Postgres-репозитории принимают `TxManager` из `internal/core/repository/postgres/transaction_manager.go`. `r.exec(ctx)` берёт `pgx.Tx` из контекста, иначе - пул; вложенный `TxManager.Do` переиспользует открытую транзакцию. `pool.BeginTx` вне `Do` не открывать.

## Миграции в `migrations/` на golang-migrate

Одна миграция - одно атомарное изменение, всегда с down. Применённую не редактировать, фикс - новой нумерованной.

| Миграция | Содержание |
|---|---|
| `000001_init` | ENUM `DEVICE_STATUS`, `ROLLOUT_CAMPAIGNS_STATUS`, `ROLLOUT_STAGES_STATUS`; таблицы `devices`, `firmware_versions`, `rollout_campaigns`, `rollout_stages`; PK `uuidv7()`; partial unique index `one_running_campaign_per_model` |
| `000002_create_update_attempts` | таблица `update_attempts` с ENUM `UPDATE_ATTEMPTS_RESULT` из значений `success`, `failure` и `timeout` |
| `000003_create_idx_campaigns_device_model_status` | индекс поиска активных кампаний для checkin |
| `000004–000007` | колонка `event_id` в `update_attempts` поэтапно: колонка - бэкфилл `uuidv7()` - NOT NULL - UNIQUE |
| `000008_create_applied_decisions` | ENUM `DECISION_TYPE` со значениями `advance_stage` и `rollback` плюс таблица `applied_decisions` с колонками `decision_id` PK, `campaign_id` FK, `decision_type` и `applied_at` |

Статусные поля - отдельные Postgres ENUM, не TEXT. Колонки с префиксом сущности, например `device_model` и `fw_version`.

## Ключи Redis

Владелец проекции активной стадии - только main service через start, resume и ApplyDecision; контроллер её только читает. Счётчики и ключи стабильности пишет только контроллер.

| Ключ | Кто пишет | TTL |
|---|---|---|
| `campaign:{id}:checkin_data` — hash из `stage_id` на 16 байт UUID и `target_percent` | main | нет |
| `stage:{stage_id}:stats` — hash из `min_sample_size` и `success_threshold` | main | нет |
| `campaign:running_campaigns` — set | main | нет |
| `campaign:{id}:stage:{stage_id}:{success,failure,timeout}` | controller | нет |
| `campaign:{id}:stable_cycles`, `campaign:{id}:decision` | controller | нет |
| `campaign:{id}:stage:{stage_id}:seen:{event_id}` — дедуп | controller | `CACHE_CAMPAIGN_EVENT_ID_SEEN_TTL` |
| `device:{id}:checkin_data` — hash из `current_version` и `last_seen` | main | TTL всего ключа, ставит каждый сеттер |

`stage_id` — сырые 16 байт UUID в кодировке `BinaryMarshaler`; читать его надо через `HGet(...).Bytes()` и `uuid.FromBytes`, а писать явными парами field/value, а не через `Scan` структуры с `uuid`, потому что hscan сначала пробует `TextUnmarshaler` и падает на бинарных байтах. У device-hash пофилдового TTL нет: последний записавший сеттер перезаписывает TTL всего ключа своим значением. Отдельный ключ `stage:{id}:target_percent` не заводится. `WarmUpCache` при старте main восстанавливает `running_campaigns` и проекцию активной стадии по running-кампаниям из Postgres; без него evaluator после рестарта main ничего не видит. Ошибка записи в кэш — только warn, а ошибка чтения ключей стадии роняет checkin.

## Kafka: семантика продюсеров и консьюмеров

Топики — по 3 партиции, автосоздание выключено: `device.checkins` с ключом `device_id`, `firmware.update-results` с `.dlq` и ключом `campaign_id`, `rollout.decisions` с `.dlq` и ключом `campaign_id`.

- Checkin-продюсер — fire-and-forget через буферизованный канал размером `BROKER_BUFFER_SIZE`, при переполнении события дропаются с warn, плюс graceful drain с таймаутом `BROKER_TIMEOUT` на `Close()`.
- Report-продюсер — синхронный `RequireOne`; при неудаче 503, но запись в `update_attempts` уже сделана.
- `rollout.decisions` — `RequireOne` и `Hash`, поле `decision_id` в формате uuidv7 генерирует evaluator; `previous_stage_id` — стадия, с которой уходим; поля `next_stage_id` в событии нет, main вычисляет следующую стадию по `order_index`.
- Consumer main из `transport/kafka/apply_decision.go` в группе `BROKER_GROUP_ID` идемпотентен: сначала ранний no-op по `SELECT` из `applied_decisions` по `decision_id`, затем stale-check того, что стадия `previous_stage_id` всё ещё в статусе `active`, а кампания — в `running`, обе строки берутся под `FOR UPDATE`, иначе фиксируется no-op с debug-логом. Невалидный JSON уходит в DLQ с commit.
- Consumer контроллера дедуплицирует по `event_id` в формате uuidv7, который генерирует main на report: Lua-скрипт делает `SETNX seen` и инкрементит счётчик.

## Пайплайн решений

Ручной `POST /v1/campaigns/{id}/advance-stage` удалён из роутера; доменная логика живёт в `RolloutCampaignService.ApplyDecision`, ветки `complete` нет — `AdvanceStage` сам завершает кампанию без следующей стадии.

Evaluator из `service/evaluator` работает по тикеру `EVALUATOR_FREQUENCY`: на каждую кампанию из `running_campaigns` читает `current_stage` как поле `stage_id` hash `checkin_data`, считает `success_rate` через `GetCampaignStats` с MGet счётчиков, а также пороги стадии. Если `sample_size` меньше `min`, возвращает `ErrNotEnoughSamples` и пропускает кампанию. Иначе при `success_rate` не ниже `threshold` решает `advance_stage`, иначе — `rollback`. Стабильность отслеживает сравнением с ключом `campaign:{id}:decision`: при смене решения сбрасывает `stable_cycles`, иначе делает `INCR`; когда число циклов достигает `EVALUATOR_REQUIRED_STABLE_CYCLES`, публикует `DecisionEvent` и чистит оба ключа.

Main применяет решение в `TxManager.Do` атомарно в три шага: проверка `applied_decisions`, затем `Create` и `AdvanceStage` либо `Rollback`, а после коммита обновляет Redis: при `advance` с найденной стадией сдвигает проекцию через удаление кэша старой стадии вызовом `deleteStageCache` и запись новой вызовом `setCampaignStageCache`, а при `advance` без следующей стадии и при `rollback` вызывает `RemoveRunningCampaigns` и чистит кэши.

## Сквозные соглашения

- Request ID: middleware `WrapInMiddleware` идёт в порядке `Recover`, `Trace`, `Logger` и `RequestID`, где `RequestID` самый внешний; клиент к контроллеру проставляет `x-request-id` из `config.CtxKeyRequestID`. Connect-хендлеры монтировать через `NewInterceptorsOption` с порядком `RequestID`, `Logger`, `Trace` и `Recover`.
- Конфиг на koanf читается из `.env` в текущей директории: `go run` запускать из папки сервиса.
- Graceful shutdown обоих сервисов — `errgroup.WithContext` + `SHUTDOWN_TIMEOUT`.
- Моки под каталогами вида `*/mocks/` генерирует `mockery` по файлу `.mockery.yaml` командой `task mock` с `GOTOOLCHAIN=go1.26.0`; руками не править.

## Конфигурация

Корневой `.env` - для compose, остальные - для приложений.

| Переменная | Описание |
|---|---|
| `POSTGRES_USER`, `POSTGRES_PASSWORD` | креды Postgres для compose и обоих `.env` |
| `POSTGRES_DB`, `POSTGRES_PORT` | база и порт Postgres |
| `REDIS_PORT`, `KAFKA_PORT` | порты Redis и Kafka наружу |
| `KAFKA_TOPIC_PARTITIONS` | число партиций на топик, по умолчанию 3 |
| `KAFKA_LOG_RETENTION_HOURS` | ретеншн логов Kafka |
| `KAFKA_UI_PORT` | порт kafka-ui, поднимается профилем `task kafka-ui` |
| `HTTP_SERVER_HOST`, `HTTP_SERVER_PORT` | адрес main service - `:8080` |
| `HTTP_SERVER_TIMEOUT` | таймаут graceful shutdown HTTP |
| `DB_HOST`, `DB_PORT`, `DB_NAME` | подключение к Postgres |
| `DB_SSL_MODE` | SSL-режим, локально `disable` |
| `DB_MAX_CONNS`, `DB_MIN_CONNS` | размер пула pgx |
| `DB_MAX_CONN_LIFETIME`, `DB_MAX_CONN_IDLE_TIME` | время жизни соединений |
| `DB_HEALTH_CHECK_PERIOD` | период health-check пула |
| `DB_MAX_CONN_LIFETIME_JITTER` | джиттер lifetime |
| `DB_REQUEST_TIMEOUT` | таймаут одного запроса к БД |
| `CACHE_HOST`, `CACHE_PORT` | подключение к Redis |
| `CACHE_DEVICE_CHECKIN_DATA_TTL` | TTL device-hash `checkin_data`, по умолчанию 24h |
| `CACHE_CAMPAIGN_EVENT_ID_SEEN_TTL` | TTL дедупа `event_id` для контроллера |
| `BROKER_HOST`, `BROKER_PORT` | подключение к Kafka |
| `BROKER_BATCH_TIMEOUT` | `BatchTimeout` writer'а |
| `BROKER_TIMEOUT` | таймаут записи и drain при shutdown |
| `BROKER_BUFFER_SIZE` | буфер checkin-канала, при переполнении события дропаются |
| `BROKER_GROUP_ID` | consumer group |
| `BROKER_MIN_BYTES` | `MinBytes` consumer'а |
| `ROLLOUT_CONTROLLER_SCHEME` | схема клиента - `http` |
| `ROLLOUT_CONTROLLER_HOST` | хост контроллера |
| `ROLLOUT_CONTROLLER_PORT` | порт контроллера - `8090` |
| `ROLLOUT_CONTROLLER_TIMEOUT` | таймаут `GetCampaignStats` |
| `SERVER_HOST`, `SERVER_PORT` | адрес контроллера - `:8090` |
| `SERVER_TIMEOUT` | таймаут graceful shutdown контроллера |
| `EVALUATOR_FREQUENCY` | период тика evaluator |
| `EVALUATOR_REQUIRED_STABLE_CYCLES` | стабильных циклов до решения |
| `SHUTDOWN_TIMEOUT` | graceful shutdown обоих сервисов, 30s |
