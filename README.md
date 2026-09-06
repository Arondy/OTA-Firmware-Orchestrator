# OTA Firmware Orchestrator

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-336791?logo=postgresql)](https://www.postgresql.org)
[![Redis](https://img.shields.io/badge/Redis-8-DC382D?logo=redis)](https://redis.io)
[![Kafka](https://img.shields.io/badge/Kafka-4-231F20?logo=apachekafka)](https://kafka.apache.org)
[![OpenAPI](https://img.shields.io/badge/OpenAPI-3-6BA539?logo=swagger)](ota-orchestrator/api/openapi.yaml)
[![Connect RPC](https://img.shields.io/badge/Connect%20RPC-1.20-77E1FF?logo=grpc)](https://connectrpc.com/)

Пет-проект уровня production-ready: canary-раскатка OTA-прошивок - обновление сначала малой группе, остальным - только при стабильных метриках. Два Go-сервиса делят ответственность - OTA Orchestrator работает с устройствами и админом, Rollout Controller автоматически двигает раскатку по метрикам.

![Архитектура системы](<Задание/Схемы/Общие/Архитектура системы.png>)

## Содержание

- [Быстрый старт](#быстрый-старт)
- [Демо](#демо)
- [Возможности](#возможности)
- [Как это работает](#как-это-работает)
- [Архитектура](#архитектура)
- [Конфигурация](#конфигурация)
- [Тестирование](#тестирование)
- [API](#api)
- [Статус и план](#статус-и-план)
- [Ограничения](#ограничения)
- [Использование ИИ](#использование-ии)

## Быстрый старт

**1. Окружение** - конфиги, инфра, топики, миграции:

```bash
cp .env.example .env
cp ota-orchestrator/.env.example ota-orchestrator/.env
cp rollout-controller/.env.example rollout-controller/.env
docker compose up -d
task kafka-init
task migrate-up
```

**2. Запуск** - оба сервиса, каждый в своём терминале:

```bash
task run-orchestrator
task run-controller
```

**3. Проверка** - оба healthcheck отвечают `OK`:

```bash
curl http://localhost:8080/healthz
curl -X POST http://localhost:8090/health.v1.HealthService/CheckHealth \
  -H 'Content-Type: application/json' -d '{}'
```

> [!NOTE]
> Топики Kafka не создаются автоматически, так как выставлен флаг `KAFKA_AUTO_CREATE_TOPICS_ENABLE=false`, и без `task kafka-init` main service не стартует.
> Конфиг читается из `.env` в текущей директории - запуск только из папки сервиса, `task run-*` уже делает это.

## Демо

Полный canary-цикл: устройство - прошивка - кампания из двух стадий на 10% и 100% - старт - checkin - report - метрики.
Тела запросов - по `ota-orchestrator/api/openapi.yaml`.

```bash
BASE=http://localhost:8080/api/v1
CSUM=9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
DEV=$(curl -s -X POST $BASE/devices -d \
  '{"device_model":"esp32-temp","current_version":"1.0.0"}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')
FW=$(curl -s -X POST $BASE/firmware -d \
  '{"device_model":"esp32-temp","fw_version":"1.1.0", \
  "fw_checksum":"'$CSUM'", \
  "binary_url":"http://files.local/fw/1.1.0.bin"}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')
CAMP=$(curl -s -X POST $BASE/campaigns -d \
  '{"firmware_version_id":"'$FW'", \
  "rollout_stages":[{"order_index":0,"target_percent":10, \
  "min_sample_size":5,"success_threshold":0.95}, \
  {"order_index":1,"target_percent":100, \
  "min_sample_size":20,"success_threshold":0.95}]}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')
curl -s -X POST $BASE/campaigns/$CAMP/start | \
  python3 -c 'import sys,json;print(json.load(sys.stdin)["status"])'
```

```bash
CHK=$(curl -s -X POST $BASE/devices/$DEV/checkin -d \
  '{"current_version":"1.0.0"}')
echo $CHK # {"update_available":true,...} или false вне бакета
STAGE=$(echo $CHK | python3 -c \
  'import sys,json;print(json.load(sys.stdin).get("stage_id") or "")')
curl -s -X POST $BASE/devices/$DEV/report -d \
  '{"campaign_id":"'$CAMP'","stage_id":"'$STAGE'","result":"success"}'
curl -s $BASE/campaigns/$CAMP | \
  python3 -m json.tool # rollout_stages + stats от контроллера
```

```bash
# Принудительный откат running/paused кампании: 202 с пустым телом, итог - опросом статуса
curl -s -o /dev/null -w "%{http_code}\n" -X POST $BASE/campaigns/$CAMP/rollback # 202
curl -s $BASE/campaigns/$CAMP | \
  python3 -c 'import sys,json;print(json.load(sys.stdin)["status"])' # rolled_back после применения решения
```

> [!TIP]
> `update_available:false` после checkin - чаще всего так задумано: bucket `hash(device_id + campaign_id) % 100` вне `target_percent` стадии, либо устройство уже на целевой версии.
> `stage_id` из ответа checkin возвращается в report без изменений.

## Возможности

Полное ТЗ - в [`Задание/ТЗ.md`](Задание/ТЗ.md), план этапов - в [`Задание/Этапы.md`](Задание/Этапы.md).

- **Устройства** - регистрация, список, вывод из эксплуатации
- **Реестр прошивок** - semver + sha256 + URL бинарника, защита от дублей пары модель/версия
- **Кампании со стадиями** - создание с упорядоченными стадиями одним запросом; цикл `draft - running - paused - running - completed`
- **Checkin на Redis** - активная стадия читается из кэша, событие уходит в Kafka fire-and-forget, ответ не ждёт брокер
- **Report** - `success`/`failure`/`timeout`, каждый результат с уникальным `event_id` пишется в `update_attempts`
- **Живые метрики** - `GET /campaigns/{id}` отдаёт `stats` с долей успеха и размером выборки; без контроллера - без `stats`
- **Авторешения** - evaluator двигает стадии командами `advance_stage` и `rollback` после стабильных циклов
- **Ручной откат** - `POST /campaigns/{id}/rollback` для `running`/`paused`: отвечает `202` и публикует решение `rollback` тем же пайплайном, что автоматика
- **Строгая валидация** - semver, sha256-hex, диапазоны стадий, лимит тела 1 MiB, запрет неизвестных JSON-полей

## Как это работает

1. Прошивка регистрируется, под неё создаётся кампания со стадиями, например 10% и 100% устройств модели.
2. `start` активирует первую стадию: Postgres - `running` + стадия `active`, Redis - проекция активной стадии.
3. Устройство шлёт `checkin` с текущей версией: сервис ищет running-кампанию по модели, bucket решает попадание в стадию; обновление выдаётся, если bucket в пределах `target_percent` и версия ниже целевой.
4. Устройство ставит прошивку и шлёт `report`; результат пишется в БД и публикуется в `firmware.update-results`.
5. Контроллер агрегирует результаты в счётчики Redis, а его evaluator периодически сверяет `success_rate` с порогом стадии и после `EVALUATOR_REQUIRED_STABLE_CYCLES` стабильных циклов публикует `rollout.decisions`.
6. Main применяет решение идемпотентно в одной транзакции: `advance` - сдвиг на следующую стадию, а если её нет - завершение в `completed`, `rollback` - переход в `rolled_back`; затем обновляет проекцию Redis.
7. Админ может принудительно откатить кампанию в статусе `running` или `paused`: `POST /campaigns/{id}/rollback` возвращает `202` с пустым телом и ничего не пишет в Postgres напрямую - контроллер публикует решение `rollback`, main применяет его тем же consumer; итог проверяется через `GET /campaigns/{id}`.

> [!NOTE]
> Повторная доставка безопасна с обеих сторон: контроллер дедуплицирует результаты по `event_id` через `SETNX` с TTL, а main - решения по `decision_id` через `applied_decisions` плюс stale-check того, что `previous_stage_id` всё ещё в статусе `active`.

## Архитектура

- **Main service** из `ota-orchestrator/` - HTTP API для устройств и админа; PostgreSQL - источник правды; решений о раскатке не принимает, только исполняет
- **Rollout Controller** из `rollout-controller/` - stateless: consumer результатов, счётчики Redis, evaluator решений, Connect-RPC `GetCampaignStats` + `ForceRollback` на `:8090`
- **PostgreSQL 18** - устройства, прошивки, кампании, стадии, попытки обновлений, применённые решения
- **Redis 8** - проекция активной стадии и счётчики; быстрый путь чтения на checkin
- **Kafka 4** - `device.checkins`, `firmware.update-results`, `rollout.decisions` и DLQ к двум последним топикам

Слои main: `transport/http - service - repository`; сборка зависимостей - `internal/core/app.go` в каждом сервисе.
Детали про миграции, ключи Redis и семантику Kafka - в [`docs/IMPLEMENTATION.md`](docs/IMPLEMENTATION.md).

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

## Тестирование

| Уровень | Команда | Требования |
|---|---|---|
| Unit - сервисы, HTTP, моки | `task test-unit` | не требуются |
| Integration + e2e по всем модулям | `task test-all` | Docker, инфра через testcontainers |
| e2e отдельно - canary-сценарий | `cd ota-orchestrator && go test -tags e2e ./tests/e2e` | Docker |

> [!NOTE]
> Без тегов - только unit-тесты.
> `task test-unit`/`test-all` гоняют оба модуля через `gotestsum`.
> Моки под каталогами вида `*/mocks/` генерирует `mockery` командой `task mock`, руками не править.
> CI в репозитории нет.

## API

Все маршруты, кроме healthcheck, - с префиксом `/api/v1`.
Полная спецификация - [`ota-orchestrator/api/openapi.yaml`](ota-orchestrator/api/openapi.yaml).

| Метод | Путь | Описание |
|---|---|---|
| `GET` | `/healthz` | здоровье main service |
| `POST` | `/api/v1/devices` | регистрация устройства |
| `GET` | `/api/v1/devices` | список с `last_seen` из Redis |
| `POST` | `/api/v1/devices/{id}/decommission` | вывод из эксплуатации |
| `POST` | `/api/v1/devices/{id}/checkin` | проверка обновлений |
| `POST` | `/api/v1/devices/{id}/report` | результат установки |
| `POST` | `/api/v1/firmware` | регистрация прошивки |
| `GET` | `/api/v1/firmware` | список прошивок |
| `POST` | `/api/v1/campaigns` | создание кампании со стадиями |
| `GET` | `/api/v1/campaigns` | список кампаний |
| `GET` | `/api/v1/campaigns/{id}` | кампания + `stats` контроллера |
| `POST` | `/api/v1/campaigns/{id}/start` | запуск: перевод `draft - running` |
| `POST` | `/api/v1/campaigns/{id}/pause` | пауза: перевод `running - paused` |
| `POST` | `/api/v1/campaigns/{id}/resume` | продолжение: перевод `paused - running` |
| `POST` | `/api/v1/campaigns/{id}/rollback` | принудительный откат `running`/`paused`: `202` пустой, решение применяется асинхронно |

## Статус и план

Реализованы этапы 1-7, источник правды - код и миграции.

| Этап | Содержание | Статус |
|---|---|---|
| 1 | Каркас, схема БД, CRUD без логики раскатки | Готово |
| 2 | Checkin/report поверх Postgres | Готово |
| 3 | Redis как быстрый путь чтения стадии | Готово |
| 4 | Kafka: события checkin и report | Готово |
| 5 | Controller: consumer результатов, счётчики | Готово |
| 6 | Evaluator, consumer решений в main | Готово |
| 7 | Ручной откат через ForceRollback | Готово |
| 8 | Индексация и устойчивость | ... |
| 9 | Полный compose-стек и frontend | ... |

## Ограничения

- Повторный `report` создаёт новую запись в `update_attempts` с новым `event_id`. Если публикация в Kafka не удалась, API вернёт 503, но запись в БД уже сделана.
- checkin-события при переполнении буфера `BROKER_BUFFER_SIZE` дропаются с warn - ответ устройству важнее доставки события.
- Одна running-кампания на модель.
- Откат асинхронный: `POST .../rollback` отвечает `202` с пустым телом и только ставит решение в очередь - Postgres меняется позже тем же consumer, что автоматику. Повторный вызов безопасен: новое решение по уже завершённой кампании станет no-op.

## Использование ИИ

- Составление ТЗ и разбивка на этапы
- Уточнения по структуре проекта
- Проверка кода на баги и соответствие ТЗ
- Генерация сообщений коммитов
- Написание README (кроме этого раздела), OpenAPI спецификации и тестов
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
  5. Этап 6:
     - рефакторинг БД с использованием TxManager и адаптация ApplyDecision под него
     - рефакторинг Redis для использования hash вместо разрозненных пар ключей
