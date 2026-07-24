CREATE TABLE IF NOT EXISTS zili_daily_log (
    id SERIAL PRIMARY KEY,
    log_date DATE NOT NULL,
    log_time TIME,
    daily_summary TEXT,
    status_weight_g INTEGER,
    pre_feed_weight_g INTEGER,
    post_feed_weight_g INTEGER,
    milk_transfer_g INTEGER,
    expressed_left_ml INTEGER,
    measurement_weight_g INTEGER
);

CREATE TABLE IF NOT EXISTS vitamin_checks (
    key TEXT PRIMARY KEY,
    checked BOOLEAN NOT NULL DEFAULT FALSE,
    date TEXT
);

