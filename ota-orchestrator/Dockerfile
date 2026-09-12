FROM golang:1.26.8-alpine3.24 AS builder

RUN apk add --no-cache wget tar xz

WORKDIR /build
RUN wget -q https://github.com/stunnel/static-curl/releases/download/8.22.0/curl-linux-x86_64-musl-8.22.0.tar.xz && \
    tar -xf curl-linux-x86_64-musl-8.22.0.tar.xz

WORKDIR /app
COPY ota-orchestrator/go.mod ota-orchestrator/go.sum ./ota-orchestrator/
COPY api/gen/go.mod api/gen/go.sum ./api/gen/
COPY tests/testutil/go.mod tests/testutil/go.sum ./tests/testutil/
COPY rollout-controller/go.mod rollout-controller/go.sum ./rollout-controller/

WORKDIR /app/ota-orchestrator
RUN go mod download

WORKDIR /app
COPY api/gen/ ./api/gen
COPY ota-orchestrator/ ./ota-orchestrator

WORKDIR /app/ota-orchestrator
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /ota-orchestrator ./cmd/ota-orchestrator

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=builder --chmod=755 /ota-orchestrator /ota-orchestrator
COPY --from=builder --chmod=755 /build/curl /usr/bin/curl

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/ota-orchestrator"]
