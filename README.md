# The Otter Hole Book Club 🦦

book club app for me and my friends!! we pick books with a spinning wheel because ranked choice voting was too much lol

go + sqlite, deployed on [exe.dev](https://exe.dev). lives at [bookclub.ffxxi.com](https://bookclub.ffxxi.com).

## what it does

- anyone can submit a book for the current round
- admin spins a wheel to pick the winner (canvas-based, very satisfying)
- presentation mode for doing the spin live on a call
- `/books` page shows our library + everything we've read, grouped by round
- admin dashboard for managing rounds, book library, submissions, schedule
- book metadata lookup from google books API (optional, set `BOOKS_API_KEY`)
- promote user submissions into the curated library
- rcv algorithm is still in the codebase if i ever want it back but the wheel is more fun

## running it

```bash
make build
./bookclub
```

listens on `:8000` by default, override with `-listen :3000` or whatever.
hostname defaults to `bookclub.ffxxi.com`, override with `-hostname whatever`.

needs a `db.sqlite3` in the working directory (auto-created with migrations on first run).

admin token gets generated on first boot and printed to stderr — save it! you can also set `ADMIN_TOKEN` env var to override.

## env / config

| var | what |
|-----|------|
| `ADMIN_TOKEN` | override the auto-generated admin token |
| `BOOKS_API_KEY` | google books API key for search & import (optional) |

admin panel lives at `/admin/{token}`.

## project structure

```
cmd/srv/          entry point
srv/              http handlers, templates, static assets
  server.go       main handlers + RCV algorithm
  admin_books.go  library CRUD
  admin_present.go presentation mode (spin + lock + set current)
  admin_promote.go submission → library promotion
  admin_rounds.go  round book pool management
  wheel_handlers.go wheel API endpoints
  metadata.go     google books integration
  templates/      html templates
  static/         css + js
db/               database layer
  migrations/     sql migrations (auto-run)
  queries/        sqlc query definitions
  dbgen/          generated query code (sqlc v1.28.0)
  sqlc.yaml       sqlc config
srv.service       systemd unit file
```

## deploying

```bash
make build
sudo cp srv.service /etc/systemd/system/srv.service
sudo systemctl daemon-reload && sudo systemctl enable --now srv
```

## the two book systems

there are two ways books get into the wheel:

1. **legacy submissions** — users submit via the form, wheel falls back to these
2. **library books** — admin-curated with metadata, added to rounds explicitly

if a round has library books added to it, the wheel uses those. otherwise it uses submissions. this lets old rounds keep working while new ones use the fancy library.

## future stuff

- discord bot integration (schema already has `discord_user_id` columns)
- maybe make the wheel sounds
- idk we'll see

---

made with love and too much coffee ☕
