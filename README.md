# OTA Firmware Orchestrator

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-336791?style=flat-square&logo=postgresql)](https://www.postgresql.org)
[![OpenAPI](https://img.shields.io/badge/OpenAPI-3.0.3-6BA539?style=flat-square&logo=swagger)](https://github.com/Arondy/OTA-Firmware-Orchestrator/blob/main/ota-orchestrator/api/openapi.yaml)

## Содержание

- [Возможности](#возможности)
- [Архитектура](#архитектура)
- [Быстрый старт](#быстрый-старт)
- [API](#api)
- [План развития](#план-развития)
- [Что реализовано сейчас](#что-реализовано-сейчас)
- [Использование ИИ](#использование-ии)

Canary-раскатка OTA-обновлений прошивок: обновление сначала получает небольшая группа устройств, и только при подтверждённой стабильности раскатка расширяется на следующую стадию. Решение о продвижении или откате принимается автоматически по метрикам успешности установки. Ответственность разделена между двумя сервисами: **main service** работает с устройствами и администратором, **Rollout Controller** — источник решений по раскатке.

Полное описание — в [`Задание/ТЗ.md`](Задание/ТЗ.md), пошаговый план — в [`Задание/Этапы.md`](Задание/Этапы.md).

## Возможности

- **Управление устройствами** — регистрация с моделью и текущей версией, вывод из эксплуатации, список устройств
- **Реестр прошивок** — регистрация версий с моделью, sha256-контрольной суммой и URL бинарника; защита от дубликатов пары (модель, версия)
- **Кампании раскатки со стадиями** — создание кампании с несколькими упорядоченными стадиями одним запросом; жизненный цикл `draft - running - paused - running - completed`
- **Checkin** — устройство сообщает текущую версию и получает ответ, доступно ли обновление (с URL бинарника и контрольной суммой)
- **Report** — устройство сообщает результат установки (`success`/`failure`/`timeout`); каждый результат сохраняется в `update_attempts`
- **Advance-stage** — ручной переход кампании к следующей стадии; после последней кампания завершается
- **Строгая валидация** — semver, sha256-hex, диапазоны стадий, лимит тела запроса 1 MiB, запрет неизвестных полей в JSON
- **OpenAPI-спецификация** — весь API описан в `ota-orchestrator/api/openapi.yaml`

## Архитектура

- **Main service** (`ota-orchestrator/`) — HTTP API для устройств и администратора, источник правды — PostgreSQL. Раскаточных решений не принимает: только выполняет их.
- **Rollout Controller** (`rollout-controller/`) — периодически оценивает метрики кампаний и публикует решения в Kafka; состояние счётчиков живёт в Redis. Появится на этапе 5.
- **PostgreSQL** — схемы устройств, прошивок, кампаний, стадий и попыток обновления.
- **Redis / Kafka** — добавляются на этапах 3–4: Redis ускоряет чтение активной стадии на checkin, Kafka связывает сервисы асинхронным пайплайном событий.

Main service построен слоями: `transport/http` (handlers + dto) → `service/<домен>` → `repository/postgres`; доменные типы и ошибки живут в `internal/core/domain`, сборка зависимостей — в `internal/core/app.go`.

## Быстрый старт

Требования: [Go 1.26+](https://go.dev/dl), [Docker](https://www.docker.com), [Task](https://taskfile.dev) (опционально).

```bash
cp .env.example .env                          # переменные Postgres
cp ota-orchestrator/.env.example ota-orchestrator/.env   # переменные приложения
docker compose up -d                          # поднять Postgres 18
task migrate-up                               # применить миграции
task run                                      # собрать и запустить main service
```

Проверка: `curl http://localhost:8080/healthz` должен вернуть `{"status":"OK"}`.

> [!NOTE]
> Конфиг приложения читается из `.env` в текущей директории, поэтому `go run` нужно запускать из `ota-orchestrator/` — `task run` делает это сам. Файлы `.env` игнорируются git, коммитятся только `.env.example`.

### Конфигурация

| Переменная | Описание |
|---|---|
| `HTTP_SERVER_HOST`, `HTTP_SERVER_PORT`, `HTTP_SERVER_TIMEOUT` | адрес, порт и таймаут graceful shutdown HTTP-сервера |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSL_MODE` | подключение к PostgreSQL |
| `DB_MAX_CONNS`, `DB_MIN_CONNS`, `DB_MAX_CONN_LIFETIME`, `DB_MAX_CONN_IDLE_TIME`, `DB_HEALTH_CHECK_PERIOD`, `DB_MAX_CONN_LIFETIME_JITTER` | параметры пула соединений pgx |
| `REQUEST_TIMEOUT` | таймаут одного запроса к БД |

## API

Все маршруты, кроме health check, имеют префикс `/api/v1`. Полная спецификация — [`ota-orchestrator/api/openapi.yaml`](ota-orchestrator/api/openapi.yaml).

| Метод | Путь | Описание |
|---|---|---|
| `GET` | `/healthz` | проверка здоровья сервиса |
| `POST` | `/devices` | регистрация устройства |
| `GET` | `/devices` | список устройств |
| `POST` | `/devices/{id}/decommission` | вывод устройства из эксплуатации |
| `POST` | `/devices/{id}/checkin` | устройство сообщает текущую версию; в ответе — доступность обновления и данные бинарника |
| `POST` | `/devices/{id}/report` | устройство сообщает результат установки; запись в `update_attempts` |
| `POST` | `/firmware` | регистрация версии прошивки |
| `GET` | `/firmware` | список версий прошивок |
| `POST` | `/campaigns` | создание кампании со стадиями; `device_model` копируется из прошивки |
| `GET` | `/campaigns` | список кампаний |
| `GET` | `/campaigns/{id}` | кампания целиком: статус и все стадии с `status` и `entered_at` |
| `POST` | `/campaigns/{id}/start` | `draft - running`, активация первой стадии |
| `POST` | `/campaigns/{id}/pause` | `running - paused` |
| `POST` | `/campaigns/{id}/resume` | `paused - running` |
| `POST` | `/campaigns/{id}/advance-stage` | переход к следующей стадии; последняя стадия завершает кампанию |

## План развития

| Этап | Содержание | Статус |
|---|---|---|
| 1 | Каркас, основная схема БД, устройства/прошивки/кампании без бизнес-логики раскатки | Готово |
| 2 | Checkin и report поверх Postgres, advance-stage | Готово |
| 3 | Redis как быстрый путь чтения активной стадии | ... |
| 4 | Kafka: события checkin и report | ... |
| 5 | Rollout Controller: consumer результатов и счётчики в Redis | ... |
| 6 | Автоматические решения evaluator'а, consumer решений в main service | ... |
| 7 | Ручной откат через gRPC ForceRollback | ... |
| 8 | Индексация и устойчивость | ... |
| 9 | Полный docker-compose стек и frontend | ... |

## Что реализовано сейчас

Состояние соответствует **этапам 1–2**.

**Main service** (`ota-orchestrator/`):
- слоистая структура `transport/http - service - repository/postgres`, чистая сборка зависимостей в `internal/core/app.go`
- HTTP-сервер на стандартной библиотеке с middleware: request ID, access-логирование, трейсинг, восстановление после паник
- конфигурация на koanf с валидацией обязательных переменных, логирование zap
- строгая обработка JSON: лимит тела 1 MiB, запрет неизвестных полей, подробные ошибки валидации с разбивкой по полям
- Checkin: устройство сообщает версию, сервер находит активную стадию running-кампании его модели и возвращает обновление или «нет обновлений»
- Report: результат установки (`success`/`failure`/`timeout`) валидируется (кампания и стадия существуют, модель совпадает) и пишется в `update_attempts`
- Advance-stage: активная стадия → `passed`, следующая → `active`; после последней стадии кампания → `completed` — всё в одной транзакции
- OpenAPI-спецификация в `ota-orchestrator/api/openapi.yaml`

**База данных** (`migrations/`):
- `000001_init`: статусные ENUM-типы (`DEVICE_STATUS`, `ROLLOUT_CAMPAIGNS_STATUS`, `ROLLOUT_STAGES_STATUS`), таблицы `devices`, `firmware_versions`, `rollout_campaigns`, `rollout_stages`; PK — `uuidv7()`, именование колонок с префиксом сущности (`device_model`, `fw_version`, `fw_checksum`); partial unique index `one_running_campaign_per_model`
- `000002_create_update_attempts`: таблица `update_attempts` с ENUM `UPDATE_ATTEMPTS_RESULT` (`success`/`failure`/`timeout`) для истории попыток установки
- `000003_create_idx_campaigns_device_model_status`: индекс для поиска активных кампаний при checkin

**Инфраструктура**: `docker-compose.yml` — Postgres 18 + контейнер миграций с профилем `migrate`; `Taskfile.yml` — run/stop/migrate-up/migrate-down/create-migration/psql.

## Использование ИИ

ИИ в проекте использовался в следующих сценариях:
- Составление ТЗ и разбивка на этапы
- Уточнения по структуре проекта
- Проверка кода на баги и соответствие ТЗ
- Написание README (кроме этого раздела)
- Написание полностью однотипного кода:
  1. Этап 1:
     - структуры конфигов с тегами, JSON теги в DTO
     - методы репозиторного слоя для получения/создания объектов по образцу
     - интерфейсы репозиториев в сервисном слое
  2. Этап 2:
     - рефакторинг сервисного слоя с разнесением по подпапкам
     - рефакторинг названий методов репозиторного слоя
