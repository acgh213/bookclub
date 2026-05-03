-- Book Club Schema

CREATE TABLE IF NOT EXISTS config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS rounds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'submissions_open',
    vote_code TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS submissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    round_id INTEGER NOT NULL,
    nickname TEXT NOT NULL,
    book_title TEXT NOT NULL,
    book_author TEXT NOT NULL,
    discord_user_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (round_id) REFERENCES rounds(id),
    UNIQUE(round_id, nickname)
);

CREATE TABLE IF NOT EXISTS votes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    round_id INTEGER NOT NULL,
    nickname TEXT NOT NULL,
    discord_user_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (round_id) REFERENCES rounds(id),
    UNIQUE(round_id, nickname)
);

CREATE TABLE IF NOT EXISTS vote_rankings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    vote_id INTEGER NOT NULL,
    submission_id INTEGER NOT NULL,
    rank INTEGER NOT NULL,
    FOREIGN KEY (vote_id) REFERENCES votes(id) ON DELETE CASCADE,
    FOREIGN KEY (submission_id) REFERENCES submissions(id) ON DELETE CASCADE,
    UNIQUE(vote_id, submission_id),
    UNIQUE(vote_id, rank)
);

CREATE TABLE IF NOT EXISTS schedule (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    round_id INTEGER,
    book_title TEXT NOT NULL,
    book_author TEXT NOT NULL,
    suggested_by TEXT,
    meeting_date TEXT,
    reading_progress TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'upcoming',
    notes TEXT DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (round_id) REFERENCES rounds(id)
);

CREATE INDEX IF NOT EXISTS idx_submissions_round ON submissions(round_id);
CREATE INDEX IF NOT EXISTS idx_votes_round ON votes(round_id);
CREATE INDEX IF NOT EXISTS idx_vote_rankings_vote ON vote_rankings(vote_id);
CREATE INDEX IF NOT EXISTS idx_schedule_status ON schedule(status);

INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (002, '002-bookclub');
