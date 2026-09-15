# frontend

SvelteKit 2 + Svelte 5 (только runes), strict TS, Tailwind v4 CSS-first, `adapter-static` в SPA-режиме. SSR нет: `src/routes/+layout.ts` ставит `ssr = false, prerender = false`, `fallback: "index.html"`. `load`-функции для данных не используются - только `createResource`.

## Команды

Все команды - под `bun` (engines `>=1.2`), из каталога `frontend/`. `npm` не использовать.
- `bun install` - строго по `bun.lock`; `bun run dev` - `:5173` с прокси на `:8080`
- `bun run check` - `svelte-check`, цель 0 ошибок / 0 предупреждений
- `bun run lint` - `prettier --check` + `eslint`; `bun run format` - починить форматирование
- `bun run test` - Vitest; один файл: `bun run test -- src/lib/domain/status.test.ts`
- `bun run build` - статическая сборка в `build/`; `bun run preview` - предпросмотр сборки через тот же прокси
- `bun run gen:api` - перегенерация `src/lib/api/schema.d.ts` из `../ota-orchestrator/api/openapi.yaml`
- Из корня: `task run-frontend` делает `install + gen:api + vite dev`

Порядок проверки: `check - lint - test - build`.

## Слой API

- `src/lib/api/endpoints.ts` - единственная точка вызова API (15 функций). Компоненты пути руками не собирают, новый вызов - только новой функцией туда.
- `src/lib/api/client.ts` - `API_BASE = ""`, все пути относительные (`/api/v1/...`, `/healthz`, `/controller/...`). `localhost` разрешен только в `vite.config.ts` (прокси dev/preview). Абсолютные URL в `lib/api/` роняют `project-rules.test.ts`.
- `src/lib/api/schema.d.ts` - генерируется, руками не править, коммитить обязательно. Docker-сборка падает, если он расходится со спекой. Алиасы - в `types.ts`.
- Сервер декодирует тела с `DisallowUnknownFields`: отправлять ровно поля DTO, лишних нет.
- Контракт аскетичен: общего числа записей нет - пейджер только «Назад/Дальше», признак следующей страницы `rows.length === limit`; `limit` больше 100 дает `400`; `stats` кампании опционален (нет при мертвом контроллере - показывать «недоступно», не ноль); откат `POST /campaigns/{id}/rollback` отвечает `202` с пустым телом - звать через `requestVoid`, статус оптимистично не менять, опрашивать до `rolled_back`.

## Состояние и опрос

- `state/resource.svelte.ts` (`createResource`) - единственная обертка загрузки: `data/error/pending/refreshing/lastUpdatedAt`, поллинг цепочкой `setTimeout` с джиттером, `AbortSignal`, отмена предыдущего запроса, стоп в скрытой вкладке, `keepPrevious` (stale-while-revalidate). `setInterval` в `.svelte` запрещен тестом.
- Списки - через `state/paged-resource.svelte.ts`; детали кампаний - через агрегатор `state/campaign-details.svelte.ts` (`setPolled` для обзора, `ensure` по видимости строк).
- Бюджеты опроса: обзор 30 с, health оркестратора/контроллера по 15 с, страница живой кампании 3 с, откат in-flight 1.5 с; списки не опрашиваются вовсе - только ручной `refresh()`. Флаг доступности API взводят только статус 0 и 502.
- Мутации - через `state/campaign-actions.svelte.ts`; порядок: действие - `endpoints.ts` - перечитывание ресурса - тост. Тексты ошибок только из `ApiError` (`api/errors.ts`), коды в компонентах не ветвить.
- Фильтры/пагинация списков - часть URL; клиентского (фильтр кампаний по статусу, сортировки) в URL нет.

## Куда класть код

- Экран: `src/routes/<маршрут>/+page.svelte` + виджеты в `src/lib/screens/<имя>/`
- Переиспользуемый контрол - в `src/lib/ui/` и сразу во все экраны с таким же элементом, дубли запрещены; единственная таблица - `ui/DataTable.svelte`
- Чистая логика - `src/lib/domain/` или `src/lib/logic/` с тестом рядом; реактивное состояние - `src/lib/state/*.svelte.ts`; графика - inline SVG в `src/lib/viz/`, без библиотек
- Компонентные тесты - через `*Harness.svelte` рядом (пример: `DataTableHarness.svelte`)

## Жесткие запреты (роняют тесты и линт)

Проверяет `src/lib/project-rules.test.ts` и `theme-contract.test.ts` - запускаются в `bun run test`:
- Svelte: только runes (`$state/$derived/$effect`), запрещены `export let`, `on:click`, `$:` и `writable/derived/readable`
- Стиль: hex только в `app.css` (токены), в разметке только токен-утилиты (`bg-bg-surface`, `text-fg-*`); палитра Tailwind вычищена (`--color-*: initial`) - запрещены `purple/indigo/...-500` и подобные; нет `serif`, `backdrop-blur`, `box-shadow: 0 0 Npx`, `h-screen` (только `dvh`), `!important`, статическим `style=` в `.svelte` (только data-driven со `{}`)
- Текст исходников: нет тире `-`/`-`, эмодзи, `BETA/LIVE`, `total_count/totalCount`
- Код: нет `any`, `@ts-ignore/@ts-expect-error/@ts-nocheck`, `eslint-disable`, `console.log` вне тестов, `alert/confirm/prompt`, `gsap/chart.js/d3`, scroll-слушателей на window
- CSP в `svelte.config.js` (`mode: hash`): inline-скриптов нет, тема ставится внешним `static/theme-init.js` до `%sveltekit.head%`; `theme-init.js` поддерживается руками и обязан совпадать с `mode-watcher` (ключ `mode-watcher-mode`, класс `dark`) - дрейф ловит `theme-contract.test.ts`

## Тема и шрифты

Токены `--bg-*/--fg-*/--accent-*/--state-*` живут в `src/app.css` (`:root` светлая, `.dark` темная, дефолт `dark` в `app.html`); `@theme inline` маппит их в утилиты. Шрифты Geist Variable самосборные в `static/fonts` с preload в `app.html`, сторонних запросов нет. Длительности анимаций - только через `--dur-*`.

## Окружение и поставка

- Приложению `VITE_*` не нужны: `frontend/.env.example` пустой, сборка работает без env. Версия/время сборки - через `define` (`__APP_VERSION__`, `__BUILD_TIME__`), руками не дублировать.
- `vite.config.ts` - единственное место с `localhost:8080` (`VITE_DEV_UPSTREAM`) и `:8090` (`VITE_DEV_CONTROLLER_UPSTREAM`).
- `Caddyfile`: `/_health` отдает `ok` (сюда ходит compose healthcheck), `/api/*` + `/healthz` - на `ORCHESTRATOR_UPSTREAM` (дефолт `ota-orchestrator:8080`), `/controller/*` - на `CONTROLLER_UPSTREAM`; `localhost:8080` внутри контейнера на хост не попадает, для хостового оркестратора использовать `host.docker.internal:8080`. CORS-заголовков нет (same-origin), CSP только в `<meta>` сборки, не в заголовке.
- `Dockerfile`: контекст - корень репо (нужна спека для `gen:api`); `oven/bun:1.4.2-alpine` собирает, `caddy:2.11-alpine` отдает `/srv`. Проверка без docker: `caddy validate --config frontend/Caddyfile --adapter caddyfile`.
- Коммиты - по корневому `AGENTS.md`: на русском, conventional commits со скоупом `frontend`.
