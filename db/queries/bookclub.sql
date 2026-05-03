-- name: GetConfig :one
SELECT * FROM config WHERE key = ? LIMIT 1;

-- name: SetConfig :exec
INSERT INTO config (key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value;

-- name: CreateRound :one
INSERT INTO rounds (title, status, vote_code)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetRound :one
SELECT * FROM rounds WHERE id = ? LIMIT 1;

-- name: GetRoundByVoteCode :one
SELECT * FROM rounds WHERE vote_code = ? LIMIT 1;

-- name: ListRounds :many
SELECT * FROM rounds ORDER BY created_at DESC;

-- name: GetCurrentRound :one
SELECT * FROM rounds 
WHERE status IN ('submissions_open', 'voting_open')
ORDER BY created_at DESC LIMIT 1;

-- name: UpdateRoundStatus :exec
UPDATE rounds SET status = ? WHERE id = ?;

-- name: CreateSubmission :one
INSERT INTO submissions (round_id, nickname, book_title, book_author, discord_user_id)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetSubmission :one
SELECT * FROM submissions WHERE id = ? LIMIT 1;

-- name: ListSubmissionsByRound :many
SELECT * FROM submissions WHERE round_id = ? ORDER BY created_at ASC;

-- name: GetSubmissionByNickname :one
SELECT * FROM submissions WHERE round_id = ? AND nickname = ? LIMIT 1;

-- name: CountSubmissionsByRound :one
SELECT COUNT(*) as count FROM submissions WHERE round_id = ?;

-- name: ListAllSubmissions :many
SELECT s.*, r.title as round_title, r.status as round_status
FROM submissions s
JOIN rounds r ON s.round_id = r.id
ORDER BY s.created_at DESC;

-- name: UpdateSubmission :exec
UPDATE submissions
SET book_title = ?, book_author = ?, nickname = ?
WHERE id = ?;

-- name: DeleteSubmission :exec
DELETE FROM submissions WHERE id = ?;

-- name: CreateVote :one
INSERT INTO votes (round_id, nickname, discord_user_id)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetVote :one
SELECT * FROM votes WHERE id = ? LIMIT 1;

-- name: GetVoteByNickname :one
SELECT * FROM votes WHERE round_id = ? AND nickname = ? LIMIT 1;

-- name: ListVotesByRound :many
SELECT * FROM votes WHERE round_id = ? ORDER BY created_at ASC;

-- name: CountVotesByRound :one
SELECT COUNT(*) as count FROM votes WHERE round_id = ?;

-- name: CreateVoteRanking :exec
INSERT INTO vote_rankings (vote_id, submission_id, rank)
VALUES (?, ?, ?);

-- name: ListRankingsByVote :many
SELECT * FROM vote_rankings WHERE vote_id = ? ORDER BY rank ASC;

-- name: ListAllRankingsByRound :many
SELECT vr.* FROM vote_rankings vr
JOIN votes v ON vr.vote_id = v.id
WHERE v.round_id = ?
ORDER BY v.id, vr.rank ASC;

-- name: DeleteVoteRankings :exec
DELETE FROM vote_rankings WHERE vote_id = ?;

-- name: CreateScheduleEntry :one
INSERT INTO schedule (round_id, book_title, book_author, suggested_by, meeting_date, reading_progress, status, notes)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetScheduleEntry :one
SELECT * FROM schedule WHERE id = ? LIMIT 1;

-- name: ListSchedule :many
SELECT * FROM schedule ORDER BY 
    CASE status
        WHEN 'reading' THEN 1
        WHEN 'upcoming' THEN 2
        WHEN 'completed' THEN 3
    END,
    created_at DESC;

-- name: GetCurrentReading :one
SELECT * FROM schedule WHERE status = 'reading' ORDER BY created_at DESC LIMIT 1;

-- name: GetNextUpcoming :one
SELECT * FROM schedule WHERE status = 'upcoming' ORDER BY created_at ASC LIMIT 1;

-- name: UpdateScheduleEntry :exec
UPDATE schedule 
SET book_title = ?, book_author = ?, suggested_by = ?, meeting_date = ?, reading_progress = ?, status = ?, notes = ?
WHERE id = ?;

-- name: UpdateScheduleProgress :exec
UPDATE schedule SET reading_progress = ? WHERE id = ?;

-- name: DeleteScheduleEntry :exec
DELETE FROM schedule WHERE id = ?;
