FROM golang:1.26-alpine3.24 AS builder

ENV GOWORK=off


WORKDIR /src

COPY proto-exchange/ /src/proto-exchange/
COPY gw-exchanger/ /src/gw-exchanger/

# Wallet
COPY gw-currency-wallet/ /src/gw-currency-wallet/

WORKDIR /src/gw-currency-wallet

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /out/gw-currency-wallet ./cmd

FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 -h /app app

WORKDIR /app

COPY --from=builder --chown=app:app \
    /out/gw-currency-wallet /app/gw-currency-wallet

USER app

EXPOSE 8080

ENTRYPOINT ["/app/gw-currency-wallet"]