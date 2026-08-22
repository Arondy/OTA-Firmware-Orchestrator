# OTA Firmware Orchestrator

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-336791?logo=postgresql)](https://www.postgresql.org)
[![Redis](https://img.shields.io/badge/Redis-8-DC382D?logo=redis)](https://redis.io)
[![Kafka](https://img.shields.io/badge/Kafka-4-231F20?logo=apachekafka)](https://kafka.apache.org)
[![OpenAPI](https://img.shields.io/badge/OpenAPI-3-6BA539?logo=swagger)](https://github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/blob/main/ota-orchestrator/api/openapi.yaml)
[![Connect RPC](https://img.shields.io/badge/Connect%20RPC-1.20-77E1FF?logo=grpc)](https://connectrpc.com/)

## Содержание

- [Возможности](#возможности)
- [Архитектура](#архитектура)
- [Как это работает](#как-это-работает)
- [Быстрый старт](#быстрый-старт)
- [API](#api)
- [План развития](#план-развития)
- [Что реализовано сейчас](#что-реализовано-сейчас)
- [Использование ИИ](#использование-ии)

Canary-раскатка OTA-обновлений прошивок: обновление сначала получает небольшая группа устройств, и только при подтверждённой стабильности раскатка расширяется на следующую стадию. Ответственность разделена между двумя сервисами: **main service** работает с устройствами и администратором, **Rollout Controller** автоматически принимает решения по раскатке по метрикам успешности установки.

Полное описание — в [`Задание/ТЗ.md`](Задание/ТЗ.md), пошаговый план — в [`Задание/Этапы.md`](Задание/Этапы.md).

## Возможности

- **Управление устройствами** — регистрация с моделью и текущей версией, вывод из эксплуатации, список устройств
- **Реестр прошивок** — регистрация версий с моделью, sha256-контрольной суммой и URL бинарника; защита от дубликатов пары модель и версия
- **Кампании раскатки со стадиями** — создание кампании с несколькими упорядоченными стадиями одним запросом; жизненный цикл `draft - running - paused - running - completed`
- **Checkin с Redis на горячем пути** — активная стадия и `target_percent` читаются из Redis, running-кампания ищется в Postgres; детерминированный bucket решает, попадает ли устройство в стадию; событие о checkin публикуется в Kafka асинхронно
  - **Report** — устройство сообщает результат установки: `success`, `failure` или `timeout`. Каждый результат сохраняется в `update_attempts` с уникальным `event_id` и публикуется в Kafka
- **Статистика раскатки** — отдельный сервис агрегирует результаты из Kafka в счётчики Redis и отдаёт долю успешных установок и размер выборки; при недоступном контроллере эндпоинт отдаёт данные без метрик
- **Событийный пайплайн Kafka** — топики `device.checkins` и `firmware.update-results; checkin уходит fire-and-forget через буферизированный канал, report — синхронно, при недоступном брокере возвращается 503
- **Advance-stage** — ручной переход кампании к следующей стадии; после последней кампания завершается
- **Прогрев кэша при старте** — main service восстанавливает ключи активных стадий в Redis по running-кампаниям из Postgres
- **Строгая валидация** — semver, sha256-hex, диапазоны стадий, лимит тела запроса 1 MiB, запрет неизвестных полей в JSON
- **OpenAPI-спецификация** — весь API описан в `ota-orchestrator/api/openapi.yaml`

## Архитектура

- **Main service** (`ota-orchestrator/`) — HTTP API для устройств и администратора, источник правды — PostgreSQL; публикует события checkin и результатов установки в Kafka. Раскаточных решений не принимает: только выполняет их.
- **Rollout Controller** (`rollout-controller/`) — отдельный stateless-сервис: потребляет результаты установки из `firmware.update-results`, ведёт счётчики в Redis и дедуплицирует повторную доставку по `event_id` через `SETNX`, отдаёт метрики раскатки через Connect-RPC `GetCampaignStats` на порту 8090. Там же смонтирован `HealthService.CheckHealth` (`POST /health.v1.HealthService/CheckHealth` → `{"status":"OK"}`) для healthcheck. Main service запрашивает метрики в `GET /campaigns/{id}`.
- **PostgreSQL** — схемы устройств, прошивок, кампаний, стадий и попыток обновления.
- **Redis** — проекция активной стадии (`campaign:{id}:current_stage`, `campaign:{id}:current_target_percent`) и `last_seen`/`current_version` устройств с TTL 24 часа; Postgres остаётся источником правды, а Redis — быстрым путём чтения на checkin.
- **Kafka** — асинхронный пайплайн событий: топик `device.checkins` с ключом `device_id` и топик `firmware.update-results` с ключом `campaign_id`. Все результаты одной кампании попадают в одну партицию, поэтому счётчики этапа 5 читаются последовательно.

Main service построен по чистой архитектуре: `transport/http` - `service` - `repository (postgres, redis, kafka)`; доменные типы и ошибки живут в `domain/`, сборка зависимостей — в отдельном файле `app.go`.

## Как это работает

1. Прошивка регистрируется, для неё создаётся кампания со стадиями — например, 25% и 100% устройств модели.
2. `start` активирует первую стадию: Postgres — `running` + стадия `active`, Redis — ключи активной стадии.
3. Устройство шлёт `checkin` с текущей версией: main service находит running-кампанию по модели в Postgres, читает `current_stage`/`current_target_percent` из Redis и хэширует `device_id` + `campaign_id` в bucket 1–100. Обновление выдаётся, если bucket ≤ `target_percent` и версия устройства ниже целевой. Событие checkin публикуется в Kafka асинхронно — ответ устройству не зависит от брокера.
4. Устройство ставит прошивку и шлёт `report` с `campaign_id` и `stage_id`; результат с уникальным `event_id` пишется в `update_attempts` и синхронно публикуется в `firmware.update-results` — контроллер дедуплицирует повторную доставку по `event_id`.
5. `advance-stage` переводит стадию в `passed` и активирует следующую, обновляя ключи в Redis; после последней стадии кампания завершается.
6. Rollout Controller асинхронно агрегирует результаты из Kafka в счётчики Redis и отдаёт их через `GET /campaigns/{id}` в поле `stats`; при недоступном контроллере эндпоинт возвращает кампанию без `stats`.

> [!NOTE]
> Повторный checkin устройства, уже сидящего на версии кампании, всегда даёт «нет обновления». `stage_id` из ответа checkin устройство возвращает в report без изменений, поэтому результат приписывается той стадии, на которой выдано обновление.

## Быстрый старт

Требования: [Go 1.26+](https://go.dev/dl), [Docker](https://www.docker.com), [Task](https://taskfile.dev) (опционально).

```bash
cp .env.example .env                                          # переменные Postgres, Redis и Kafka
cp ota-orchestrator/.env.example ota-orchestrator/.env        # переменные main service
cp rollout-controller/.env.example rollout-controller/.env    # переменные контроллера
docker compose up -d                                          # поднять Postgres, Redis, Kafka
task kafka-init                                               # создать топики Kafka - один раз
task migrate-up                                               # применить миграции
task run-orchestrator                                         # собрать и запустить main service на порту 8080
task run-controller                                           # запустить Rollout Controller на порту 8090
```

Проверка: `curl http://localhost:8080/healthz` должен вернуть `{"status":"OK"}`.

> [!NOTE]
> Топики Kafka не создаются автоматически — обязателен запуск `task kafka-init` после поднятия Kafka в первый раз. kafka-ui поднимается отдельным профилем: `task kafka-ui`.

> [!NOTE]
> Конфиг приложения читается из `.env` в текущей директории, поэтому `go run` нужно запускать из папки сервиса — `task run-orchestrator` / `task run-controller` уже делают это.

### Конфигурация

| Переменная | Описание |
|---|---|
| `HTTP_SERVER_HOST`, `HTTP_SERVER_PORT`, `HTTP_SERVER_TIMEOUT` | адрес, порт и таймаут graceful shutdown HTTP-сервера |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSL_MODE` | подключение к PostgreSQL |
| `DB_MAX_CONNS`, `DB_MIN_CONNS`, `DB_MAX_CONN_LIFETIME`, `DB_MAX_CONN_IDLE_TIME`, `DB_HEALTH_CHECK_PERIOD`, `DB_MAX_CONN_LIFETIME_JITTER` | параметры пула соединений pgx |
| `CACHE_HOST`, `CACHE_PORT` | подключение к Redis |
| `CACHE_DEVICE_LAST_SEEN_TTL`, `CACHE_DEVICE_CURRENT_VERSION_TTL` | TTL ключей устройств, по умолчанию 24 часа |
| `BROKER_HOST`, `BROKER_PORT`, `BROKER_TIMEOUT` | подключение к Kafka и таймаут одной записи |
| `BROKER_BUFFER_SIZE` | размер буфера checkin-продюсера; при переполнении события дропаются с warn-логом |
| `ROLLOUT_CONTROLLER_SCHEME`, `ROLLOUT_CONTROLLER_HOST`, `ROLLOUT_CONTROLLER_PORT`, `ROLLOUT_CONTROLLER_TIMEOUT` | подключение клиента к Rollout Controller через `GetCampaignStats`; при недоступности `GET /campaigns/{id}` отдаётся без `stats` |
| `REQUEST_TIMEOUT` | таймаут одного запроса к БД |

## Тестирование

Проект покрыт unit-, integration- и e2e-тестами. Команды запускаются из папки соответствующего модуля (`ota-orchestrator/` или `rollout-controller/`).

| Уровень | Команда | Требования |
| --- | --- | --- |
| Unit (сервисы и HTTP-обработчики, на моках) | `task test-unit` либо `cd ota-orchestrator && go test ./...` | не требуются |
| Integration (`repository/postgres`, `repository/redis`, `repository/kafka`) | `task test-all` либо `go test -tags integration ./...` | запущенный **Docker** (testcontainers поднимает Postgres и Redis) |
| e2e (сценарий canary-раскатки, `tests/e2e`) | `go test -tags e2e ./tests/e2e` | Docker; оба сервиса собираются локально через `replace` |

> [!NOTE]
> `task test-unit` и `task test-all` используют `gotestsum` и прогоняют тесты в обоих модулях. Без билд-тега выполняются только unit-тесты; integration и e2e изолированы тегами `integration` и `e2e` соответственно.

> [!TIP]
> Моки зависимостей в каталогах `*/mocks/` генерируются `mockery` по `.mockery.yaml` — править их вручную не нужно. CI в репозитории отсутствует.

## API

Все маршруты, кроме health check, имеют префикс `/api/v1`. Полная спецификация — [`ota-orchestrator/api/openapi.yaml`](ota-orchestrator/api/openapi.yaml).

| Метод | Путь | Описание |
|---|---|---|
| `GET` | `/healthz` | проверка здоровья сервиса |
| `POST` | `/devices` | регистрация устройства |
| `GET` | `/devices` | список устройств с `last_seen` и `current_version` из Redis, когда ключи не истекли |
| `POST` | `/devices/{id}/decommission` | вывод устройства из эксплуатации |
| `POST` | `/devices/{id}/checkin` | устройство сообщает текущую версию; в ответе — доступность обновления и данные бинарника; событие публикуется в Kafka |
| `POST` | `/devices/{id}/report` | устройство сообщает результат установки; запись в `update_attempts` с `event_id` и публикация в Kafka; при недоступном брокере — 503 |
| `POST` | `/firmware` | регистрация версии прошивки |
| `GET` | `/firmware` | список версий прошивок |
| `POST` | `/campaigns` | создание кампании со стадиями; `device_model` копируется из прошивки |
| `GET` | `/campaigns` | список кампаний |
| `GET` | `/campaigns/{id}` | кампания целиком: статус, все стадии с `status` и `entered_at`, плюс `stats` от Rollout Controller — доля успешных установок и размер выборки, если контроллер доступен |
| `POST` | `/campaigns/{id}/start` | `draft - running`, активация первой стадии |
| `POST` | `/campaigns/{id}/pause` | `running - paused` |
| `POST` | `/campaigns/{id}/resume` | `paused - running` |
| `POST` | `/campaigns/{id}/advance-stage` | переход к следующей стадии; последняя стадия завершает кампанию |

## План развития

| Этап | Содержание | Статус |
|---|---|---|
| 1 | Каркас, основная схема БД, устройства/прошивки/кампании без бизнес-логики раскатки | Готово |
| 2 | Checkin и report поверх Postgres, advance-stage | Готово |
| 3 | Redis как быстрый путь чтения активной стадии | Готово |
| 4 | Kafka: события checkin и report | Готово |
| 5 | Rollout Controller: consumer результатов и счётчики в Redis | Готово |
| 6 | Автоматические решения evaluator'а, consumer решений в main service | ... |
| 7 | Ручной откат через gRPC ForceRollback | ... |
| 8 | Индексация и устойчивость | ... |
| 9 | Полный docker-compose стек и frontend | ... |

## Что реализовано сейчас

Состояние соответствует **этапам 1–5**.

**Main service** (`ota-orchestrator/`):
- слоистая структура `transport/http - service - repository/postgres + repository/redis + repository/kafka`, сборка зависимостей в `internal/core/app.go`
- HTTP-сервер на стандартной библиотеке с middleware: request ID, access-логирование, трейсинг, восстановление после паник
- конфигурация на koanf с валидацией обязательных переменных, логирование zap
- строгая обработка JSON: лимит тела 1 MiB, запрет неизвестных полей, подробные ошибки валидации с разбивкой по полям
- Checkin: running-кампания ищется в Postgres по `device_model`, активная стадия и `target_percent` читаются из Redis; ответ — обновление или «нет обновлений»; событие checkin публикуется в Kafka асинхронно, не влияя на ответ
- Report: результат установки — `success`, `failure` или `timeout` — валидируется: кампания и стадия существуют, модель совпадает; результат пишется в `update_attempts` с `event_id` и синхронно публикуется в `firmware.update-results`. Неудачная публикация даёт 503, но запись в БД уже сделана; повторный report создаёт новую запись с новым `event_id`. Это осознанное ограничение пет-проекта.
- Advance-stage: активная стадия - `passed`, следующая - `active`; после последней стадии кампания - `completed` — всё в одной транзакции, ключи стадии обновляются в Redis
- GET /v1/devices подмешивает `last_seen`/`current_version` из Redis поверх строки Postgres; при истёкших ключах — значения из Postgres
- прогрев кэша при старте: `WarmUpCache` восстанавливает ключи активных стадий по running-кампаниям
- OpenAPI-спецификация в `ota-orchestrator/api/openapi.yaml`

**База данных** (`migrations/`):
- `000001_init`: статусные ENUM-типы (`DEVICE_STATUS`, `ROLLOUT_CAMPAIGNS_STATUS`, `ROLLOUT_STAGES_STATUS`), таблицы `devices`, `firmware_versions`, `rollout_campaigns`, `rollout_stages`; PK — `uuidv7()`, именование колонок с префиксом сущности (`device_model`, `fw_version`, `fw_checksum`); partial unique index `one_running_campaign_per_model`
- `000002_create_update_attempts`: таблица `update_attempts` с ENUM `UPDATE_ATTEMPTS_RESULT` (`success`/`failure`/`timeout`) для истории попыток установки
- `000003_create_idx_campaigns_device_model_status`: индекс для поиска активных кампаний при checkin
- `000004`–`000007`: колонка `event_id` в `update_attempts` expand-contract'ом — колонка + бэкафилл `uuidv7()` - NOT NULL - UNIQUE-индекс - constraint; `event_id` генерирует main service на report, и по нему контроллер дедуплицирует повторную доставку

**Кэш** (`repository/redis`): ключи `campaign:{id}:current_stage` и `campaign:{id}:current_target_percent` хранятся без TTL, а `device:{id}:last_seen` и `device:{id}:current_version` — с TTL из конфига; Postgres остаётся источником правды, ошибки записи в кэш не роняют checkin.

**События** (`repository/kafka`): продюсер checkin — буферизированный канал с неблокирующей отправкой через `select`/`default`, события дропаются при переполнении с warn-логом, запись `RequireNone`; продюсер результатов — синхронная запись `RequireOne`, ключ `campaign_id` держит результаты одной кампании в одной партиции. `event_id` генерируется на report как `uuidv7` и совпадает в БД и в событии.

**Инфраструктура**: `docker-compose.yml` — Postgres 18 + Redis 8 + Kafka 4.3.1 + контейнер миграций с профилем `migrate`; Kafka-топики, включая `firmware.update-results.dlq`, создаются профилем `kafka-init` через `task kafka-init`, визуальная отладка — профилем `kafka-ui` через `task kafka-ui`; `Taskfile.yml` — run-orchestrator/run-controller/stop/migrate-up/migrate-down/create-migration/kafka-init/kafka-ui/psql/gen.

**Rollout Controller** (`rollout-controller/`):
- отдельный stateless-сервис на Connect-RPC поверх HTTP/JSON — не нативный gRPC-фрейминг — на порту 8090; смонтированы `CampaignService.GetCampaignStats` (метрики раскатки) и `HealthService.CheckHealth` (healthcheck, `POST /health.v1.HealthService/CheckHealth` → `{"status":"OK"}`); все хендлеры проходят через интерцепторы (request ID/`x-request-id`, per-procedure лог, трейсинг латентности, восстановление после паник)
- consumer группы на `firmware.update-results`: дедуп повторной доставки через `SETNX campaign:{id}:stage:{stage_id}:seen:{event_id}` с TTL из `CACHE_CAMPAIGN_EVENT_ID_SEEN_TTL` + инкремент счётчиков `:success`/`:failure`/`:timeout` в Redis; невалидные и необрабатываемые сообщения уходят в `firmware.update-results.dlq`
- gRPC-клиент в main service (`ROLLOUT_CONTROLLER_*`): при недоступности контроллера `GET /campaigns/{id}` отдаёт данные без `stats` и пишет warn; иначе подмешивает `Stats` с полями `active_stage_id`, `success_rate`, `sample_size`
- `api/` — отдельный модуль со сгенерированным protobuf/Connect-кодом (`task gen` после правки `api/proto/**`); подключён через `replace` в go.mod обоих сервисов

## Использование ИИ

ИИ в проекте использовался в следующих сценариях:
- Составление ТЗ и разбивка на этапы
- Уточнения по структуре проекта
- Проверка кода на баги и соответствие ТЗ
- Генерация сообщений коммитов
- Написание README (кроме этого раздела), OpenAPI спецификации, тестов
- Написание полностью однотипного кода:
  1. Этап 1:
     - структуры конфигов с тегами, JSON теги в DTO
     - методы репозиторного слоя для получения/создания объектов по образцу
     - интерфейсы репозиториев в сервисном слое
  2. Этап 2:
     - рефакторинг сервисного слоя с разнесением по подпапкам
     - рефакторинг названий методов репозиторного слоя
  3. Этап 4:
     - добавление JSON тегов к Event структурам
     - рефакторинг сервисного слоя устройств с вынесением сервиса обновления
  4. Этап 5:
     - рефакторинг транспортного слоя ota-orchestrator с объединением хэндлеров и DTO и разбиением на отдельные файлы
     - небольшие правки для облегчения тестов и соответствия идиоматичному Go
