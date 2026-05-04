# The Otter Hole Book Club 🦦

book club app for me and my friends!! we pick books with a spinning wheel because ranked choice voting was too much lol

go + sqlite, deployed on [exe.dev](https://exe.dev). lives at [bookclub.ffxxi.com](https://bookclub.ffxxi.com).

## what it does

- anyone can submit a book for the current round
- admin spins a wheel to pick the winner (canvas-based, very satisfying)
- `/books` page shows everything we've read / are reading, grouped by round
- admin dashboard for managing rounds, submissions, schedule
- rcv algorithm is still in the codebase if i ever want it back but the wheel is more fun

## running it

```bash
make build
./bookclub
```

listens on `:8000` by default, override with `-listen :3000` or whatever.

needs a `db.sqlite3` in the working directory (auto-created with migrations on first run).

admin token gets generated on first boot and printed to stderr — save it! you can also set `ADMIN_TOKEN` env var to override.

## env / config

| var | what |
|-----|------|
| `ADMIN_TOKEN` | override the auto-generated admin token |

admin panel lives at `/admin/{token}`.

## project structure

```
cmd/srv/        entry point
srv/            http handlers, templates, static assets
db/             database layer, migrations, sqlc queries
db/dbgen/       generated query code (sqlc v1.30.0)
srv.service     systemd unit file
```

## deploying

```bash
make build
sudo cp srv.service /etc/systemd/system/srv.service
sudo systemctl daemon-reload && sudo systemctl enable --now srv
```

## future stuff

- discord bot integration (schema already has `discord_user_id` columns)
- maybe make the wheel sounds
- idk we'll see

---

made with love and too much coffee ☕
