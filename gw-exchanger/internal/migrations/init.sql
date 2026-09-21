CREATE TABLE IF NOT EXISTS rates (
    currency     TEXT PRIMARY KEY,
    rate_to_rub  NUMERIC(20, 8) NOT NULL CHECK (rate_to_rub > 0),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO rates (currency, rate_to_rub) VALUES
    ('USD', 90.00000000),
    ('EUR', 100.00000000),
    ('RUB', 1.00000000)
ON CONFLICT (currency) DO UPDATE
SET rate_to_rub = EXCLUDED.rate_to_rub,
    updated_at  = NOW();