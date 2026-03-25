-- name: CreateRecord :one
INSERT INTO records (id, user_id, title, artist, year, label, format, condition, genre, notes, cover_art_url, discogs_id, favorite, personal_note, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetRecordByID :one
SELECT * FROM records WHERE id = ? AND user_id = ? LIMIT 1;

-- name: UpdateRecord :one
UPDATE records
SET title = ?,
    artist = ?,
    year = ?,
    label = ?,
    format = ?,
    condition = ?,
    genre = ?,
    notes = ?,
    cover_art_url = ?,
    favorite = ?,
    personal_note = ?,
    updated_at = ?
WHERE id = ? AND user_id = ?
RETURNING *;

-- name: DeleteRecord :exec
DELETE FROM records WHERE id = ? AND user_id = ?;

-- name: ListRecords :many
SELECT * FROM records
WHERE user_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountRecords :one
SELECT COUNT(*) FROM records WHERE user_id = ?;

-- name: FindDuplicateRecord :one
SELECT id FROM records
WHERE user_id = ? AND LOWER(title) = LOWER(?) AND LOWER(artist) = LOWER(?)
LIMIT 1;

-- name: UpdateRecordCoverArt :exec
UPDATE records SET cover_art_url = ?, updated_at = ? WHERE id = ? AND user_id = ?;
