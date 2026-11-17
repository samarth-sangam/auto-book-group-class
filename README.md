# auto-book-group-class

A small Go program that books your cult.fit classes. GitHub Actions runs it every day at 10 PM IST, which is free.

## Layout (read in this order if you're learning Go)
- `cmd/booker/main.go`: entry point. It reads flags and env, then loops over wanted classes.
- `internal/prefs`: loads `preferences.json` (JSON decoding, validation, `time`).
- `internal/cult/client.go`: the HTTP client (`net/http`, contexts, JSON). **The only file that breaks if cult changes its API.**
- `internal/cult/match.go` + `match_test.go`: pure logic with a table-driven test.

## Run locally
```
export CULT_COOKIE='<Cookie header value>'
go run ./cmd/booker -dry-run -debug   # see raw API responses, books nothing
go run ./cmd/booker                   # really book
go test ./...
```

## Getting CULT_COOKIE
1. Log in at cult.fit in Chrome and open DevTools, then the Network tab.
2. Click any request to `cult.fit/api/...`. Copy the whole `Cookie` request header value. If there is an `apikey` header, copy it too (`CULT_API_KEY`).
3. Treat it like a password. It expires, so refresh it when the job starts failing with "session expired".

## Deploy (free)
1. Push this folder to a GitHub repo. Private is fine. Free minutes are plenty.
2. Repo, Settings, Secrets and variables, Actions: add `CULT_COOKIE` (and `CULT_API_KEY` if needed).
3. Edit `preferences.json` for weekday, time and workout.
4. Actions tab, "Book cult classes", Run workflow to test it. After that it runs nightly.

## Caveats
- cult.fit has no official API. Endpoints and JSON shapes in `client.go` come from community write-ups and are **unverified**. If `-dry-run -debug` shows a different shape, fix `classesResponse`, the paths, or the headers in that one file.
- GitHub cron can start 5 to 30 minutes late, and it pauses scheduled workflows after 60 days of repo inactivity. Push a commit occasionally.
- A failed run exits non-zero, so GitHub emails you.
- Automating against a private API may violate cult.fit's ToS. Use at your own risk.
