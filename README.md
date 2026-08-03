# OTA Firmware Orchestrator

Canary-раскатка обновлений прошивки на парк устройств с автоматическим принятием решений.

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-336791?style=flat-square&logo=postgresql)](https://www.postgresql.org)

## Содержание

- [Возможности](#возможности)
- [Архитектура](#архитектура)
- [Быстрый старт](#быстрый-старт)
- [API](#api)
- [План развития](#план-развития)
- [Что реализовано сейчас](#что-реализовано-сейчас)

Система управляет раскаткой обновлений прошивки методом canary-деплоя: обновление сначала получает небольшой процент устройств, и только при подтверждённой стабильности раскатка расширяется на следующую группу. Решение о продвижении или откате стадии принимается автоматически на основе метрик успешности установки, без участия человека в штатном режиме.

Ответственность разделена между двумя сервисами: **main service** работает с устройствами и администратором, **Rollout Controller** - единственный источник решений по раскатке. Полное описание - в [`Задание/ТЗ.md`](Задание/ТЗ.md), пошаговый план - в [`Задание/Этапы.md`](Задание/Этапы.md).

## Возможности

- **Управление устройствами** - регистрация с моделью и начальной версией, списание, список всех устройств
- **Реестр прошивок** - регистрация версий с моделью, sha256-контрольной суммой и URL бинарника; защита от дубликатов пары (модель, версия)
- **Кампании раскатки со стадиями** - создание кампании с несколькими упорядоченными стадиями одним запросом
- **Жизненный цикл кампании** - `draft - running - paused - running`, старт активирует первую стадию
- **Валидация входных данных** - semver, sha256-hex, диапазоны стадий, порядок индексов стадий, лимит размера тела запроса
- **OpenAPI-спецификация** - весь API описан в `ota-orchestrator/api/openapi.yaml`

## Архитектура

- **Main service** (`ota-orchestrator/`) - HTTP API для устройств и администратора, источник правды - PostgreSQL. Раскаточных решений не принимает: только выполняет их.
- **Rollout Controller** - периодически оценивает метрики кампаний и публикует решения в Kafka; состояние счётчиков живёт в Redis.
- **PostgreSQL** - схемы устройств, прошивок, кампаний и стадий.
- **Redis / Kafka** - добавляются на этапах 3–6: Redis ускоряет чтение активной стадии на checkin, Kafka связывает сервисы асинхронным пайплайном событий.

## Быстрый старт

Требования: [Go 1.26+](https://go.dev/dl), [Docker](https://www.docker.com), [Task](https://taskfile.dev) (опционально).

```bash
cp .env.example .env        # задать переменные Postgres
cp ota-orchestrator/.env.example ota-orchestrator/.env # задать переменные приложения
docker compose up -d        # поднять Postgres
task migrate-up             # применить миграции
task run                    # собрать и запустить main service
```

Проверка: `curl http://localhost:8080/healthz` должен вернуть `{"status":"OK"}`.

### Конфигурация

| Переменная | Описание |
|---|---|
| `HTTP_SERVER_HOST`, `HTTP_SERVER_PORT`, `HTTP_SERVER_TIMEOUT` | адрес, порт и таймаут graceful shutdown HTTP-сервера |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSL_MODE` | подключение к PostgreSQL |
| `DB_MAX_CONNS`, `DB_MIN_CONNS`, `DB_MAX_CONN_LIFETIME`, `DB_MAX_CONN_IDLE_TIME`, `DB_HEALTH_CHECK_PERIOD`, `DB_MAX_CONN_LIFETIME_JITTER` | параметры пула соединений pgx |
| `REQUEST_TIMEOUT` | таймаут одного запроса к БД |

## API

Все маршруты, кроме health check, имеют префикс `/api/v1`. Полная спецификация - [`ota-orchestrator/api/openapi.yaml`](ota-orchestrator/api/openapi.yaml).

| Метод | Путь | Описание |
|---|---|---|
| `GET` | `/healthz` | проверка здоровья сервиса |
| `POST` | `/devices` | регистрация устройства |
| `GET` | `/devices` | список устройств |
| `POST` | `/devices/{id}/decommission` | вывод устройства из эксплуатации |
| `POST` | `/firmware` | регистрация версии прошивки |
| `GET` | `/firmware` | список версий прошивок |
| `POST` | `/campaigns` | создание кампании со стадиями; `device_model` копируется из прошивки |
| `GET` | `/campaigns` | список кампаний |
| `GET` | `/campaigns/{id}` | кампания целиком: статус и все стадии с `status` и `entered_at` |
| `POST` | `/campaigns/{id}/start` | `draft - running`, активация первой стадии |
| `POST` | `/campaigns/{id}/pause` | `running - paused` |
| `POST` | `/campaigns/{id}/resume` | `paused - running` |

## План развития

| Этап | Содержание | Статус |
|---|---|---|
| 1 | Каркас, основная схема БД, устройства/прошивки/кампании без бизнес-логики раскатки | Готово |
| 2 | Checkin и report поверх Postgres | ... |
| 3 | Redis как быстрый путь чтения активной стадии | ... |
| 4 | Kafka: события checkin и report | ... |
| 5 | Rollout Controller: consumer результатов и счётчики в Redis | ... |
| 6 | Автоматические решения evaluator'а, consumer решений в main service | ... |
| 7 | Ручной откат через gRPC ForceRollback | ... |
| 8 | Индексация и устойчивость | ... |
| 9 | Полный docker-compose стек и frontend | ... |

## Что реализовано сейчас

Состояние соответствует **этапу 1** - каркас и основная схема, без бизнес-логики раскатки. Redis, Kafka и checkin/report здесь намеренно отсутствуют.

**Main service** (`ota-orchestrator/`):
- слоистая структура: `transport/http - service - repository/postgres`, чистая сборка зависимостей в `internal/core/app.go`
- HTTP-сервер на стандартной библиотеке с middleware: request ID, access-логирование, трейсинг, восстановление после паник
- конфигурация на koanf с валидацией обязательных переменных, логирование zap
- строгая обработка JSON: лимит тела 1 MiB, запрет неизвестных полей, подробные ошибки валидации с разбивкой по полям
- отправка доменных ошибок на уровне репозитория/сервиса
- OpenAPI-спецификация в `ota-orchestrator/api/openapi.yaml`

**База данных** (`migrations/000001_init.up.sql`):
- статусные ENUM-типы: `DEVICE_STATUS`, `ROLLOUT_CAMPAIGNS_STATUS`, `ROLLOUT_STAGES_STATUS`
- таблицы `devices`, `firmware_versions`, `rollout_campaigns`, `rollout_stages`; PK - `uuidv7()`, PK в стиле `device_model` / `fw_version` / `fw_checksum`
- ограничения: unique (модель, версия прошивки), unique (кампания, порядок стадии), CHECK-диапазоны стадий, partial unique index `one_running_campaign_per_model`
- миграция имеет down-версию; применяется отдельным контейнером `migrate` через docker compose

**Инфраструктура**: `docker-compose.yml` - Postgres 18 + контейнер миграций с профилем `migrate`, `Taskfile.yml` - run/stop/migrate/psql.
