CREATE INDEX IF NOT EXISTS idx_sub_user
    ON subscription(user_id);

CREATE INDEX IF NOT EXISTS idx_sub_service
    ON subscription(service_id);

CREATE INDEX IF NOT EXISTS idx_sub_start_end
    ON subscription(start_date, end_date);