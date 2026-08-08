CREATE TABLE IF NOT EXISTS zili_daily_log (
    id integer NOT NULL,
    log_date date NOT NULL,
    log_time time without time zone,
    daily_summary text,
    status_weight_g integer,
    pre_feed_weight_g integer,
    post_feed_weight_g integer,
    milk_transfer_g integer,
    measurement_weight_g integer,
    height_cm numeric(5,1),
    head_cm numeric(5,1),
    sleep_event text,
    diaper text,
    fed_breast boolean DEFAULT false,
    fed_bottle boolean DEFAULT false,
    bathed boolean DEFAULT false,
    milestone boolean DEFAULT false
);

CREATE TABLE IF NOT EXISTS vitamin_checks (
    key TEXT PRIMARY KEY,
    checked BOOLEAN NOT NULL DEFAULT FALSE,
    date TEXT
);

CREATE TABLE IF NOT EXISTS app_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS zili_checklists (
    id         SERIAL PRIMARY KEY,
    title      TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS zili_checklist_items (
    id         SERIAL PRIMARY KEY,
    list_id    INT NOT NULL REFERENCES zili_checklists(id) ON DELETE CASCADE,
    text       TEXT NOT NULL,
    checked    BOOLEAN NOT NULL DEFAULT FALSE,
    position   INT NOT NULL DEFAULT 0
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
    recurring    TEXT NOT NULL DEFAULT 'none',
    all_day      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

