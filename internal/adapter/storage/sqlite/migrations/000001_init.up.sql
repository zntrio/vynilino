CREATE TABLE IF NOT EXISTS users (
    id          TEXT PRIMARY KEY,
    email       TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'user',  -- 'user' | 'admin'
    failed_login_count INTEGER NOT NULL DEFAULT 0,
    locked_until INTEGER,  -- unix timestamp, NULL = not locked
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS records (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    artist      TEXT NOT NULL,
    year        INTEGER,
    label       TEXT,
    format      TEXT,      -- 'LP' | 'EP' | 'Single' | '7"' | '10"' | '12"'
    condition   TEXT,      -- Goldmine grade
    genre       TEXT,      -- JSON array stored as text
    notes       TEXT,
    cover_art_url TEXT,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_records_user_id ON records(user_id);
CREATE INDEX IF NOT EXISTS idx_records_artist  ON records(artist);
CREATE INDEX IF NOT EXISTS idx_records_title   ON records(title);

CREATE VIRTUAL TABLE IF NOT EXISTS records_fts USING fts5(
    title,
    artist,
    label,
    notes,
    content='records',
    content_rowid='rowid'
);

CREATE TRIGGER IF NOT EXISTS records_fts_insert AFTER INSERT ON records BEGIN
    INSERT INTO records_fts(rowid, title, artist, label, notes)
    VALUES (new.rowid, new.title, new.artist, new.label, new.notes);
END;

CREATE TRIGGER IF NOT EXISTS records_fts_update AFTER UPDATE ON records BEGIN
    INSERT INTO records_fts(records_fts, rowid, title, artist, label, notes)
    VALUES ('delete', old.rowid, old.title, old.artist, old.label, old.notes);
    INSERT INTO records_fts(rowid, title, artist, label, notes)
    VALUES (new.rowid, new.title, new.artist, new.label, new.notes);
END;

CREATE TRIGGER IF NOT EXISTS records_fts_delete AFTER DELETE ON records BEGIN
    INSERT INTO records_fts(records_fts, rowid, title, artist, label, notes)
    VALUES ('delete', old.rowid, old.title, old.artist, old.label, old.notes);
END;

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  INTEGER NOT NULL,
    revoked     INTEGER NOT NULL DEFAULT 0,
    created_at  INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
