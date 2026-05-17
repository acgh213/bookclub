# Agent Instructions — The Otter Hole Book Club

## Overview

Go + SQLite web app for a small friend-group book club. Members submit books,
admin spins a wheel to pick the winner, group reads together. Deployed on
[exe.dev](https://exe.dev), lives at `bookclub.ffxxi.com` (reverse proxy on 443,
app listens on `:8000`).

## Architecture

- **Module**: `github.com/acgh213/bookclub`
- **Entry point**: `cmd/srv/main.go` — flags for `-listen` and `-hostname`
- **Server**: `srv/server.go` — HTTP handlers, template rendering, RCV algorithm (legacy, still present)
- **Handler files**: `srv/admin_books.go` (library CRUD), `srv/admin_present.go` (presentation mode + result confirm/lock), `srv/admin_promote.go` (submission → library promotion), `srv/admin_rounds.go` (round book pool management), `srv/wheel_handlers.go` (wheel API: fetch books, record spin, set current book), `srv/metadata.go` (book metadata lookup via Google Books API or manual fallback)
- **Templates**: `srv/templates/` — layout.html + page templates (home, submit, vote, results, schedule, admin, books, library, book_edit, round_books, present)
- **Static assets**: `srv/static/style.css`, `srv/static/script.js` — embedded via `//go:embed`
- **Database**: SQLite at `db.sqlite3` (gitignored), auto-created on first run
- **Migrations**: `db/migrations/` — 001-base.sql, 002-bookclub.sql, 003-v0.2-books.sql
- **Queries**: `db/queries/bookclub.sql` (core CRUD), `db/queries/v0.2.sql` (library/round entries/results)
- **Generated code**: `db/dbgen/` via sqlc — config in `db/sqlc.yaml`. sqlc is NOT installed on the VM; edit generated code by hand if needed.
- **Systemd**: `srv.service` → `/etc/systemd/system/srv.service`
- **Binary**: `bookclub` (gitignored)

## Database Schema (3 migrations)

### Core tables (002)
- `config` — key/value store (admin_token)
- `rounds` — book selection rounds with status (submissions_open, voting_open, closed) and vote_code
- `submissions` — user book submissions per round (nickname, book_title, book_author)
- `votes` / `vote_rankings` — RCV voting (legacy, still functional)
- `schedule` — reading schedule entries with status (upcoming, reading, completed)

### Library tables (003)
- `books` — standalone curated library with rich metadata (title, author, submitter, pitch, tags, page_count, description, content_notes, cover_url, isbn, metadata_source, metadata_id, status, deleted_at)
- `round_entries` — junction table linking books to rounds (replaces submissions for wheel)
- `round_results` — recorded spin results with preview→lock workflow (confirmed flag)

## Two Book Systems

There are two parallel systems that coexist:

1. **Legacy submissions** — users submit via `/submit` form, stored in `submissions` table. The wheel falls back to these if no `round_entries` exist for a round.
2. **Library system** — admin-curated `books` table with rich metadata. Admin adds books to rounds via `round_entries`. The wheel prefers these.

The transition is per-round: if a round has `round_entries`, the wheel uses those. Otherwise it falls back to `submissions`.

## Key Flows

### Public
- **Home** (`/`) — shows current book (from library via confirmed round_result), or next scheduled book, or active round info
- **Submit** (`/submit`) — submit a book for the active round (legacy submissions system)
- **Books** (`/books`) — public library view. Shows library books if any exist, otherwise falls back to legacy submission groups. Supports search (`?q=`), status filter (`?status=`), sort (`?sort=title|author|recent`)
- **Vote/Wheel** (`/vote/{code}`) — wheel spinner page. Fetches books via `/api/round/{id}/wheel-books`
- **Results** (`/results/{id}`) — RCV results page (legacy)
- **Schedule** (`/schedule`) — reading schedule

### API
- `GET /api/round/{id}/wheel-books` — returns books for wheel (round_entries first, submissions fallback)
- `POST /api/round/{id}/spin` — records spin result (book_id) as unconfirmed
- `POST /api/round/{id}/set-current` — sets confirmed winner as current reading
- `POST /api/result/{id}/confirm` — locks a spin result

### Admin (`/admin/{token}/...`)
- Dashboard — overview of rounds, submissions, schedule
- Round management — create, change status, manage book pool, present mode
- Book library — CRUD, search/import via Google Books API, archive
- Submission management — edit, delete, promote to library
- Schedule management — CRUD
- Presentation mode (`/admin/{token}/round/{id}/present`) — fullscreen wheel + preview→lock→set-current workflow

## Auth

Admin auth is a secret token URL. Token is auto-generated on first boot (stored in `config` table) or overridden via `ADMIN_TOKEN` env var. No user auth — submissions/votes use self-reported nicknames.

## Environment Variables

| Variable | Purpose |
|----------|--------|
| `ADMIN_TOKEN` | Override auto-generated admin token |
| `BOOKS_API_KEY` | Google Books API key for metadata lookup (optional) |

## Style & Conventions

- **Aesthetic**: 2000s pastel with gradient backgrounds, Fredoka One + Quicksand fonts
- **Emoji policy**: minimal — only 🦦 otter in key spots
- **Templates**: Go html/template with `layout.html` base, `jsonStr` func for safe JS embedding
- **All assets embedded**: templates + static files via `//go:embed` — single binary deployment
- **No JS framework**: vanilla JS, canvas-based wheel

## Build & Deploy

```bash
make build          # → ./bookclub binary
make test           # runs go test ./...
sudo systemctl restart srv  # restart after rebuild
```

## Database Backups

The database has been lost before. Always back up `db.sqlite3` before major changes:
```bash
cp db.sqlite3 db.sqlite3.bak.$(date +%Y%m%d_%H%M%S)
```

Backup files currently on disk:
- `db.sqlite3.bak.*` — timestamped backups
- `db.sqlite3.pre-merge` — state before otterhole/bookclub DB merge
