FROM golang:1.25.6-alpine3.23 AS builder

WORKDIR /src

COPY proto-exchange/go.mod proto-exchange/go.sum /src/proto-exchange/
COPY gw-exchanger/go.mod gw-exchanger/go.sum /src/gw-exchanger/
COPY proto-exchange/ /src/proto-exchange/

WORKDIR /src/gw-exchanger
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY gw-exchanger/ /src/gw-exchanger/

ENV GOWORK=off

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/gw-exchanger ./cmd

FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 app

WORKDIR /app

COPY --from=builder --chown=app:app /out/gw-exchanger /app/gw-exchanger

USER app
EXPOSE 50051

ENTRYPOINT ["/app/gw-exchanger"]