-- v0.2: Book library, round entries, and round results
--
-- Adds standalone book library (separate from submissions),
-- round-book junction table, and spin/selection result tracking.

-- Book library (standalone, admin-curated)
CREATE TABLE IF NOT EXISTS books (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    author TEXT NOT NULL,
    submitter TEXT,
    pitch TEXT,
    tags TEXT,
    page_count INTEGER,
    description TEXT,
    content_notes TEXT,
    cover_url TEXT,
    isbn TEXT,
    metadata_source TEXT,     -- google_books, open_library, manual
    metadata_id TEXT,         -- external ID from metadata source
    status TEXT NOT NULL DEFAULT 'proposed',
    deleted_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Junction: which books are in which round
CREATE TABLE IF NOT EXISTS round_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    round_id INTEGER NOT NULL REFERENCES rounds(id),
    book_id INTEGER NOT NULL REFERENCES books(id),
    added_by TEXT,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(round_id, book_id)
);

-- Recorded spin/selection results with preview→lock workflow
CREATE TABLE IF NOT EXISTS round_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    round_id INTEGER NOT NULL REFERENCES rounds(id),
    book_id INTEGER REFERENCES books(id),
    result_type TEXT NOT NULL DEFAULT 'spin',
    selected_by TEXT,
    selected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    confirmed INTEGER NOT NULL DEFAULT 0,
    confirmed_by TEXT,
    confirmed_at TIMESTAMP
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_books_status ON books(status);
CREATE INDEX IF NOT EXISTS idx_books_deleted ON books(deleted_at);
CREATE INDEX IF NOT EXISTS idx_round_entries_round ON round_entries(round_id);
CREATE INDEX IF NOT EXISTS idx_round_entries_book ON round_entries(book_id);
CREATE INDEX IF NOT EXISTS idx_round_results_round ON round_results(round_id);

-- Record migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (003, '003-v0.2-books');
