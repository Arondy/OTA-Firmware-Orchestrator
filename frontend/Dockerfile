
FROM oven/bun:1.4.2-alpine AS builder

WORKDIR /app

COPY frontend/package.json frontend/bun.lock ./frontend/
RUN cd frontend && bun install --frozen-lockfile

COPY frontend/ ./frontend/
COPY ota-orchestrator/api/openapi.yaml ./ota-orchestrator/api/openapi.yaml

# Схема API перегенерируется из спеки и сверяется с закоммиченной копией
RUN cd frontend \
	&& cp src/lib/api/schema.d.ts /tmp/schema.committed.d.ts \
	&& bun run gen:api \
	&& if ! cmp -s /tmp/schema.committed.d.ts src/lib/api/schema.d.ts; then \
	echo "ОШИБКА: закоммиченный src/lib/api/schema.d.ts расходится с openapi.yaml."; \
	echo "Выполните bun run gen:api локально и закоммитьте результат."; \
	exit 1; \
	fi \
	&& bun run build

FROM caddy:2.11-alpine

COPY frontend/Caddyfile /etc/caddy/Caddyfile
COPY --from=builder /app/frontend/build /srv

USER 1000:1000

EXPOSE 80
