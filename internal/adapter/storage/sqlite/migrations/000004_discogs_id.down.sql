-- SQLite does not support DROP COLUMN in older versions; recreate the table without discogs_id
CREATE TABLE records_backup AS SELECT id, user_id, title, artist, year, label, format, condition, genre, notes, cover_art_url, created_at, updated_at FROM records;
DROP TABLE records;
ALTER TABLE records_backup RENAME TO records;
