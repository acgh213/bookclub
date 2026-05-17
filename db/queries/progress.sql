-- Reading Weeks

-- name: CreateReadingWeek :one
INSERT INTO reading_weeks (schedule_id, week_number, week_label, start_chapter, end_chapter, notes, discussion_date)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetReadingWeek :one
SELECT * FROM reading_weeks WHERE id = ? LIMIT 1;

-- name: ListReadingWeeks :many
SELECT * FROM reading_weeks WHERE schedule_id = ? ORDER BY week_number ASC;

-- name: UpdateReadingWeek :exec
UPDATE reading_weeks SET week_label = ?, start_chapter = ?, end_chapter = ?, notes = ?, discussion_date = ?
WHERE id = ?;

-- name: DeleteReadingWeek :exec
DELETE FROM reading_weeks WHERE id = ?;

-- name: GetLatestWeek :one
SELECT * FROM reading_weeks WHERE schedule_id = ? ORDER BY week_number DESC LIMIT 1;

-- Check-ins

-- name: UpsertCheckin :exec
INSERT INTO checkins (schedule_id, nickname, week_number, emoji)
VALUES (?, ?, ?, ?)
ON CONFLICT(schedule_id, nickname, week_number) DO UPDATE SET emoji = excluded.emoji;

-- name: ListCheckinsByWeek :many
SELECT * FROM checkins WHERE schedule_id = ? AND week_number = ? ORDER BY created_at ASC;

-- name: ListCheckinsBySchedule :many
SELECT * FROM checkins WHERE schedule_id = ? ORDER BY week_number ASC, created_at ASC;

-- name: CountCheckinsByWeek :one
SELECT COUNT(*) as count FROM checkins WHERE schedule_id = ? AND week_number = ?;

-- name: DeleteCheckin :exec
DELETE FROM checkins WHERE schedule_id = ? AND nickname = ? AND week_number = ?;
