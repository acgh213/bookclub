-- v0.3: Reading progress tracking, check-ins, remove nickname uniqueness

-- 1. Recreate submissions without UNIQUE(round_id, nickname)
CREATE TABLE IF NOT EXISTS submissions_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    round_id INTEGER NOT NULL,
    nickname TEXT NOT NULL,
    book_title TEXT NOT NULL,
    book_author TEXT NOT NULL,
    discord_user_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (round_id) REFERENCES rounds(id)
);
INSERT INTO submissions_new SELECT * FROM submissions;
DROP TABLE submissions;
ALTER TABLE submissions_new RENAME TO submissions;
CREATE INDEX IF NOT EXISTS idx_submissions_round ON submissions(round_id);

-- 2. Recreate votes without UNIQUE(round_id, nickname)
CREATE TABLE IF NOT EXISTS votes_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    round_id INTEGER NOT NULL,
    nickname TEXT NOT NULL,
    discord_user_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (round_id) REFERENCES rounds(id)
);
INSERT INTO votes_new SELECT * FROM votes;
DROP TABLE votes;
ALTER TABLE votes_new RENAME TO votes;
CREATE INDEX IF NOT EXISTS idx_votes_round ON votes(round_id);

-- 3. Add progress columns to schedule
ALTER TABLE schedule ADD COLUMN total_chapters INTEGER;
ALTER TABLE schedule ADD COLUMN cover_url TEXT;

-- 4. Reading weeks (week-by-week chapter assignments)
CREATE TABLE IF NOT EXISTS reading_weeks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    schedule_id INTEGER NOT NULL REFERENCES schedule(id) ON DELETE CASCADE,
    week_number INTEGER NOT NULL,
    week_label TEXT,
    start_chapter TEXT,
    end_chapter TEXT,
    notes TEXT,
    discussion_date TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(schedule_id, week_number)
);
CREATE INDEX IF NOT EXISTS idx_reading_weeks_schedule ON reading_weeks(schedule_id);

-- 5. Check-ins (members mark they're caught up)
CREATE TABLE IF NOT EXISTS checkins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    schedule_id INTEGER NOT NULL REFERENCES schedule(id) ON DELETE CASCADE,
    nickname TEXT NOT NULL,
    week_number INTEGER NOT NULL,
    emoji TEXT DEFAULT '🦦',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(schedule_id, nickname, week_number)
);
CREATE INDEX IF NOT EXISTS idx_checkins_schedule ON checkins(schedule_id);

INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (004, '004-progress-and-checkins');
