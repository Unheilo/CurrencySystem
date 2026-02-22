CREATE TABLE exchange_rates (
                                id SERIAL PRIMARY KEY,
                                date TIMESTAMPTZ NOT NULL,
                                base_currency VARCHAR(10) NOT NULL DEFAULT 'EUR',
                                currency_rates JSONB NOT NULL,
                                created_at TIMESTAMPTZ DEFAULT NOW(),
                                UNIQUE (base_currency, currency_rates)
);

CREATE INDEX idx_exchange_rates_date_base_currency ON exchange_rates(date, base_currency);
