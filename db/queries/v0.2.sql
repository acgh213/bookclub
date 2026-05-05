-- Books (library)

-- name: CreateBook :one
INSERT INTO books (title, author, submitter, pitch, tags, page_count, description, content_notes, cover_url, isbn, metadata_source, metadata_id, status)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetBook :one
SELECT * FROM books WHERE id = ? AND deleted_at IS NULL LIMIT 1;

-- name: ListBooks :many
SELECT * FROM books WHERE deleted_at IS NULL ORDER BY created_at DESC;

-- name: ListBooksByStatus :many
SELECT * FROM books WHERE status = ? AND deleted_at IS NULL ORDER BY created_at DESC;

-- name: SearchBooks :many
SELECT * FROM books
WHERE deleted_at IS NULL
  AND (title LIKE '%' || ? || '%' OR author LIKE '%' || ? || '%' OR tags LIKE '%' || ? || '%')
ORDER BY created_at DESC;

-- name: UpdateBook :exec
UPDATE books
SET title = ?, author = ?, submitter = ?, pitch = ?, tags = ?,
    page_count = ?, description = ?, content_notes = ?, cover_url = ?,
    isbn = ?, metadata_source = ?, metadata_id = ?, status = ?
WHERE id = ? AND deleted_at IS NULL;

-- name: UpdateBookStatus :exec
UPDATE books SET status = ?, deleted_at = CASE WHEN ? = 'archived' THEN CURRENT_TIMESTAMP ELSE deleted_at END
WHERE id = ?;

-- name: ArchiveBook :exec
UPDATE books SET status = 'archived', deleted_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: CountBooks :one
SELECT COUNT(*) as count FROM books WHERE deleted_at IS NULL;

-- Round Entries

-- name: AddBookToRound :one
INSERT INTO round_entries (round_id, book_id, added_by)
VALUES (?, ?, ?)
RETURNING *;

-- name: RemoveBookFromRound :exec
DELETE FROM round_entries WHERE round_id = ? AND book_id = ?;

-- name: ListRoundEntries :many
SELECT b.*, re.added_by as entry_added_by, re.added_at as entry_added_at
FROM round_entries re
JOIN books b ON re.book_id = b.id
WHERE re.round_id = ? AND b.deleted_at IS NULL
ORDER BY re.added_at ASC;

-- name: CountRoundEntries :one
SELECT COUNT(*) as count FROM round_entries WHERE round_id = ?;

-- name: GetRoundEntry :one
SELECT * FROM round_entries WHERE round_id = ? AND book_id = ? LIMIT 1;

-- Round Results

-- name: CreateRoundResult :one
INSERT INTO round_results (round_id, book_id, result_type, selected_by)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetLatestRoundResult :one
SELECT * FROM round_results WHERE round_id = ? ORDER BY selected_at DESC LIMIT 1;

-- name: GetConfirmedResult :one
SELECT * FROM round_results WHERE round_id = ? AND confirmed = 1 ORDER BY confirmed_at DESC LIMIT 1;

-- name: ListRoundResults :many
SELECT * FROM round_results WHERE round_id = ? ORDER BY selected_at DESC;

-- name: ConfirmResult :exec
UPDATE round_results
SET confirmed = 1, confirmed_by = ?, confirmed_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: GetCurrentBook :one
SELECT b.*
FROM round_results rr
JOIN books b ON rr.book_id = b.id
WHERE rr.confirmed = 1
ORDER BY rr.confirmed_at DESC
LIMIT 1;
