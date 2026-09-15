# OTA Firmware Orchestrator

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-336791?logo=postgresql)](https://www.postgresql.org)
[![Redis](https://img.shields.io/badge/Redis-8-DC382D?logo=redis)](https://redis.io)
[![Kafka](https://img.shields.io/badge/Kafka-4-231F20?logo=apachekafka)](https://kafka.apache.org)
[![OpenAPI](https://img.shields.io/badge/OpenAPI-3-6BA539?logo=swagger)](ota-orchestrator/api/openapi.yaml)
[![Connect RPC](https://img.shields.io/badge/Connect%20RPC-1.20-77E1FF?logo=grpc)](https://connectrpc.com/)

Пет-проект уровня production-ready: canary-раскатка OTA-прошивок - обновление сначала малой группе, остальным - только при стабильных метриках.

Два Go-сервиса делят ответственность - OTA Orchestrator работает с устройствами и админом, Rollout Controller автоматически двигает раскатку по метрикам. Поверх - веб-консоль из `frontend/`: стадии, метрики и откат в браузере, детали в [`frontend/README.md`](frontend/README.md).

![Архитектура системы](<Задание/Схемы/Общие/Архитектура системы.png>)

## Содержание

- [Быстрый старт](#быстрый-старт)
- [Демо](#демо)
- [Возможности](#возможности)
- [Как это работает](#как-это-работает)
- [Архитектура](#архитектура)
- [Конфигурация](#конфигурация)
- [API](#api)
- [Тестирование](#тестирование)
- [Нагрузочное тестирование](#нагрузочное-тестирование)
- [Ограничения](#ограничения)
- [Возможные расширения](#возможные-расширения)
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

**2. Запуск** - оба сервиса и консоль, каждый в своём терминале:

```bash
task run-orchestrator
task run-controller
task run-frontend # нужен bun; консоль на http://localhost:5173
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
> `docker compose up -d` поднимает и контейнеры приложений. Отдельная сборка образов приложений возможна через `task build`.

## Демо

Полный canary-цикл за пару минут. Тела запросов - по `ota-orchestrator/api/openapi.yaml`.

**1. База** - устройство, прошивка, кампания из двух стадий на 10% и 100%, затем старт:

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

**2. Цикл обновления** - checkin решает попадание в бакет и выдаёт `stage_id`, report фиксирует результат, чтение кампании показывает живые метрики от контроллера:

```bash
CHK=$(curl -s -X POST $BASE/devices/$DEV/checkin -d \
  '{"current_version":"1.0.0"}')
echo $CHK
STAGE=$(echo $CHK | python3 -c \
  'import sys,json;print(json.load(sys.stdin).get("stage_id") or "")')
curl -s -X POST $BASE/devices/$DEV/report -d \
  '{"campaign_id":"'$CAMP'","stage_id":"'$STAGE'","result":"success"}'
curl -s $BASE/campaigns/$CAMP | \
  python3 -m json.tool
```

**3. Принудительный откат** - ответ `202` с пустым телом, итог проверяется опросом статуса до `rolled_back`:

```bash
curl -s -o /dev/null -w "%{http_code}\n" -X POST $BASE/campaigns/$CAMP/rollback
curl -s $BASE/campaigns/$CAMP | \
  python3 -c 'import sys,json;print(json.load(sys.stdin)["status"])'
```

**4. Списки** - фильтры и пагинация: устройства по модели и статусу, прошивки по модели, везде `page`/`limit`:

```bash
curl -s "$BASE/devices?device_model=esp32-temp&status=active&page=1&limit=10" | python3 -m json.tool
curl -s "$BASE/firmware?device_model=esp32-temp&limit=5" | python3 -m json.tool
curl -s "$BASE/campaigns?page=1&limit=10" | python3 -m json.tool
```

> [!TIP]
> `update_available:false` после checkin - чаще всего так задумано: bucket `hash(device_id + campaign_id) % 100` вне `target_percent` стадии, либо устройство уже на целевой версии.
> `stage_id` из ответа checkin возвращается в report без изменений.

## Возможности

Полное ТЗ - в [`Задание/ТЗ.md`](Задание/ТЗ.md), план этапов - в [`Задание/Этапы.md`](Задание/Этапы.md).

- **Устройства** - регистрация, список с фильтром по модели/статусу и пагинацией, вывод из эксплуатации
- **Реестр прошивок** - semver + sha256 + URL бинарника, защита от дублей пары модель/версия, список с фильтром по модели и пагинацией
- **Кампании со стадиями** - создание с упорядоченными стадиями одним запросом, список с пагинацией; цикл `draft - running - paused - running - completed`
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
- **Frontend** - SvelteKit SPA: в dev `:5173` с прокси на оркестратор, в compose статика за Caddy на `:3000`.

Слои main: `transport/http - service - repository`; сборка зависимостей - `internal/core/app.go` в каждом сервисе.
Детали про миграции, ключи Redis и семантику Kafka - в [`docs/IMPLEMENTATION.md`](docs/IMPLEMENTATION.md).

## Конфигурация

Полный список переменных - последним пунктом в [`docs/IMPLEMENTATION.md`](docs/IMPLEMENTATION.md).

## API

Все маршруты, кроме healthcheck, - с префиксом `/api/v1`.
Полная спецификация - [`ota-orchestrator/api/openapi.yaml`](ota-orchestrator/api/openapi.yaml).
Пагинация списков: `page` начинается с 1 и без параметра равен 1, `limit` без параметра равен серверному `DB_PAGINATION_LIMIT`, в `.env.example` это 100, превышать его нельзя - иначе `400`; сортировка везде от новых к старым.

| Метод | Путь | Описание |
|---|---|---|
| `GET` | `/healthz` | здоровье main service |
| `POST` | `/api/v1/devices` | регистрация устройства |
| `GET` | `/api/v1/devices` | список: `?device_model=&status=&page=&limit=`, `last_seen` из Redis |
| `POST` | `/api/v1/devices/{id}/decommission` | вывод из эксплуатации |
| `POST` | `/api/v1/devices/{id}/checkin` | проверка обновлений |
| `POST` | `/api/v1/devices/{id}/report` | результат установки |
| `POST` | `/api/v1/firmware` | регистрация прошивки |
| `GET` | `/api/v1/firmware` | список: `?device_model=&page=&limit=` |
| `POST` | `/api/v1/campaigns` | создание кампании со стадиями |
| `GET` | `/api/v1/campaigns` | список: `?page=&limit=` |
| `GET` | `/api/v1/campaigns/{id}` | кампания + `stats` контроллера |
| `POST` | `/api/v1/campaigns/{id}/start` | запуск: перевод `draft - running` |
| `POST` | `/api/v1/campaigns/{id}/pause` | пауза: перевод `running - paused` |
| `POST` | `/api/v1/campaigns/{id}/resume` | продолжение: перевод `paused - running` |
| `POST` | `/api/v1/campaigns/{id}/rollback` | принудительный откат `running`/`paused`: `202` пустой, решение применяется асинхронно |

## Тестирование

| Уровень | Команда | Требования |
|---|---|---|
| Unit - сервисы, HTTP, моки | `task test-unit` | не требуются |
| Integration + e2e по всем модулям | `task test-all` | Docker, инфра через testcontainers |
| e2e - canary-сценарий | `cd ota-orchestrator && go test -tags e2e ./tests/e2e` | Docker |
| Нагрузочное - vegeta ramp-up | см. раздел «Нагрузочное тестирование» | сид `tests/seed`, оба сервиса, Docker |

> [!NOTE]
> Без тегов - только unit-тесты.
> `task test-unit`/`test-all` гоняют оба модуля через `gotestsum`.
> Моки под каталогами вида `*/mocks/` генерирует `mockery` командой `task mock`, руками не править.

## Нагрузочное тестирование

Стенд этапа 8 собран на сиде `tests/seed` - 5000 устройств и 50 running-кампаний.

На каждый таргет-файл сервисы перезапускались, потом 1 прогревочный прогон и по 5 замеров на каждый таргет. vegeta запускался с отдельного ПК. Нагрузка росла с шагом +100 RPS, тестирование останавливалось при медианной задержке выше 100ms или ошибках выше 1%; стабильным считался последний шаг, удовлетворяющий этим условиям. Все ответы сервера - HTTP 200.

#### Технические характеристики стенда

- CPU: AMD Ryzen 5 5600H, 6C/12T, 3.3 GHz base
- RAM: 16 GB DDR4, 3200 MHz

#### Результаты

- `/checkin` - стабильно 2700-3000 RPS, медиана 2900:

| Прогон | RPS | mean | p99 | ошибки |
|---|---|---|---|---|
| 1 | 3000 | 31ms | 123ms | 0.10% |
| 2 | 2800 | 32ms | 105ms | 0.02% |
| 3 | 2900 | 13ms | 31ms | 0.06% |
| 4 | 2700 | 13ms | 48ms | 0.06% |
| 5 | 2900 | 29ms | 112ms | 0.02% |

- `/report` - стабильно 2000-2300 RPS, медиана 2200:

| Прогон | RPS | mean | p99 | ошибки |
|---|---|---|---|---|
| 1 | 2300 | 98ms | 183ms | 0.02% |
| 2 | 2200 | 50ms | 112ms | 0.04% |
| 3 | 2200 | 45ms | 74ms | 0.02% |
| 4 | 2300 | 30ms | 69ms | 0.10% |
| 5 | 2000 | 22ms | 54ms | 0.06% |

## Ограничения

- Повторный `report` создаёт новую запись в `update_attempts` с новым `event_id`. Если публикация в Kafka не удалась, API вернёт 503, но запись в БД уже сделана.
- checkin-события при переполнении буфера `BROKER_BUFFER_SIZE` дропаются с warn - ответ устройству важнее доставки события.
- Одна running-кампания на модель.
- Откат асинхронный: `POST .../rollback` отвечает `202` с пустым телом и только ставит решение в очередь - Postgres меняется позже тем же consumer, что автоматику. Повторный вызов безопасен: новое решение по уже завершённой кампании станет no-op.

## Возможные расширения

- [ ] mTLS между main service и Rollout Controller
- [ ] Идемпотентность report по ключу, который генерирует и передаёт само устройство
- [ ] Отдельный сервис уведомлений на события rollback
- [ ] Миграция на Kubernetes с масштабированием под нагрузку от количества устройств

## Использование ИИ

- Составление ТЗ и разбивка на этапы
- Уточнения по структуре проекта
- Проверка кода на баги и соответствие ТЗ
- Генерация сообщений коммитов
- Написание README (кроме этого раздела), OpenAPI спецификации, фронтенда и тестов
- Написание полностью однотипного кода (в случае рефакторинга - по моему образцу):
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
  6. Этап 8:
     - изменения в Kafka/Redis для использования новых значений из конфига
     - вставка данных и скрипты для нагрузочного тестирования
  7. Этап 9:
     - помощь с Caddyfile, Dockerfile и docker-compose.yaml