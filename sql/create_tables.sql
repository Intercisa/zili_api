CREATE TABLE IF NOT EXISTS zili_daily_log (
    id                   SERIAL PRIMARY KEY,
    log_date             DATE NOT NULL,
    log_time             TIME,
    daily_summary        TEXT,
    status_weight_g      INTEGER,
    pre_feed_weight_g    INTEGER,
    post_feed_weight_g   INTEGER,
    milk_transfer_g      INTEGER,
    measurement_weight_g INTEGER,
    height_cm            NUMERIC(5,1),
    head_cm              NUMERIC(5,1),
    sleep_event          TEXT,
    diaper               TEXT,
    fed_breast           BOOLEAN DEFAULT FALSE,
    fed_bottle           BOOLEAN DEFAULT FALSE,
    bathed               BOOLEAN DEFAULT FALSE,
    milestone            BOOLEAN DEFAULT FALSE,
    pending              BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS vitamin_checks (
    key     TEXT PRIMARY KEY,
    checked BOOLEAN NOT NULL DEFAULT FALSE,
    date    TEXT
);

CREATE TABLE IF NOT EXISTS app_settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS zili_checklist_items (
    id       SERIAL PRIMARY KEY,
    list_id  INT NOT NULL,
    text     TEXT NOT NULL,
    checked  BOOLEAN NOT NULL DEFAULT FALSE,
    position INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS zili_words (
    id         SERIAL PRIMARY KEY,
    word       TEXT NOT NULL,
    noted_date DATE NOT NULL DEFAULT CURRENT_DATE,
    notes      TEXT
);

CREATE TABLE IF NOT EXISTS zili_events (
    id           SERIAL PRIMARY KEY,
    title        TEXT NOT NULL,
    category     TEXT NOT NULL,
    event_date   DATE NOT NULL,
    event_time   TIME,
    duration_min INT NOT NULL DEFAULT 60,
    notes        TEXT,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    recurring    TEXT NOT NULL DEFAULT 'none',
    all_day      BOOLEAN NOT NULL DEFAULT FALSE
);

