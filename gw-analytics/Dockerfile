FROM golang:1.26-alpine3.24 AS builder

ENV GOWORK=off

WORKDIR /src

COPY gw-analytics/go.mod gw-analytics/go.sum /src/gw-analytics/

WORKDIR /src/gw-analytics
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY gw-analytics/ /src/gw-analytics/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/gw-analytics ./cmd

FROM alpine:3.24
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 -h /app app
WORKDIR /app
COPY --from=builder --chown=app:app /out/gw-analytics /app/gw-analytics
USER app
EXPOSE 8082 9092
ENTRYPOINT ["/app/gw-analytics"]