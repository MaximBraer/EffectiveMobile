CREATE TABLE IF NOT EXISTS subscription (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID    NOT NULL,
    service_id  INT     NOT NULL REFERENCES service(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    price_rub   INTEGER NOT NULL CHECK (price_rub >= 0),
    start_date  DATE    NOT NULL,
    end_date    DATE    NULL,
    CHECK (start_date = date_trunc('month', start_date)::date),
    CHECK (end_date IS NULL OR end_date = date_trunc('month', end_date)::date),

    CHECK (end_date IS NULL OR end_date >= start_date),

    UNIQUE (user_id, service_id, start_date)
);