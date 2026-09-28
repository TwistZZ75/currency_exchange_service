CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Пользователи
CREATE TABLE IF NOT EXISTS users (
    id          BIGSERIAL PRIMARY KEY,
    username    TEXT UNIQUE NOT NULL,
    email       TEXT UNIQUE NOT NULL,
    password    TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Балансы (одна строка на пользователя, три колонки под валюты)
CREATE TABLE IF NOT EXISTS wallets (
    user_id     BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    usd         NUMERIC(20, 2) NOT NULL DEFAULT 0 CHECK (usd >= 0),
    rub         NUMERIC(20, 2) NOT NULL DEFAULT 0 CHECK (rub >= 0),
    eur         NUMERIC(20, 2) NOT NULL DEFAULT 0 CHECK (eur >= 0),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Операции
CREATE TABLE IF NOT EXISTS operations (
    id              BIGSERIAL PRIMARY KEY,
    transaction_id  UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type            TEXT NOT NULL CHECK (type IN ('deposit', 'withdraw', 'exchange')),
    from_currency   TEXT,
    to_currency     TEXT,
    amount          NUMERIC(20, 2) NOT NULL,
    exchanged_amount NUMERIC(20, 2),
    rate            NUMERIC(20, 8),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_operations_user ON operations(user_id);
CREATE INDEX IF NOT EXISTS idx_operations_created ON operations(created_at DESC);