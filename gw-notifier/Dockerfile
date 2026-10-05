FROM golang:1.26-alpine3.24 AS builder

ENV GOWORK=off

WORKDIR /src

COPY gw-notifier/go.mod gw-notifier/go.sum /src/gw-notifier/

WORKDIR /src/gw-notifier
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY gw-notifier/ /src/gw-notifier/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/gw-notifier ./cmd

FROM alpine:3.24
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 -h /app app
WORKDIR /app
COPY --from=builder --chown=app:app /out/gw-notifier /app/gw-notifier
USER app
EXPOSE 9090
ENTRYPOINT ["/app/gw-notifier"]