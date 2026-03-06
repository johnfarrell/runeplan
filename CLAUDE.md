# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# Additional Instructions
- Always use the defined Golang standards @.claude/rules/go-conventions.md
- Announce in terminal "I have read CLAUDE.md"


## Project

RunePlan is an OSRS (Old School RuneScape) goal tracker. Users link RSN accounts, browse a catalog of canonical goals (quests, diaries), activate goals against their RSN, and track skill/requirement progress.

## Commands

**Build:**
```sh
templ generate && go build ./...
```
`templ generate` must run before `go build` — both `.templ` and `_templ.go` files are committed.

**Run:**
```sh
DATABASE_URL=postgres://... go run ./cmd/server/main.go
```

**Test:**
```sh
go test ./...                        # all tests
go test ./domain/skill/...           # single package
go test -run TestName ./package/...  # single test
```

**Dependencies:** Do not run `go mod tidy` until all new deps are actually imported in code.

## Architecture

DDD-style layering: `domain/` → `application/` → `infrastructure/` → `interfaces/`

- **`domain/`** — pure value objects and entities, no I/O
  - `skill/` — `Skill`, `XP`, `Level` types + XP table
  - `user/` — `User{ID, RSNs}`, `RSN{ID, UserID, RSN, SkillLevels}`, context helpers (`SetUser`/`GetUser`)
  - `goal/` — `Goal`, requirement types, `Type` enum
  - `catalog/` — `CatalogGoal` with typed requirements (skill/item/boss)

- **`application/`** — one package per domain, holds service logic, depends only on domain + repo interfaces

- **`infrastructure/`**
  - `postgres/` — one repo file per domain (`catalog_repository.go`, `goal_repository.go`, `user_repository.go`); connection and migration runner
  - `hiscores/` — HTTP client that parses the OSRS CSV hiscore API

- **`interfaces/`**
  - `handler/` — HTTP handlers, one file per domain
  - `middleware/` — `devauth.go` (hardcoded user stub), `logging.go`
  - `templates/` — a-h/templ components; subdirs per page (`catalog/`, `goal/`, `skill/`, `user/`); `render.go` helper

## Key Conventions

**Handler signature:** `func Name(svc *app.Service) http.HandlerFunc`

**Rendering:** always use `templates.Render(w, r, status, component)` — never write directly to `w`

**HTMX pattern:** handlers check `r.Header.Get("HX-Request") == "true"` and return a fragment instead of a full page

**User in context:** `user.GetUser(ctx)` / `user.SetUser(ctx, u)` with unexported context key — goals belong to an RSN (`rsn_id`), not directly to a `user_id`

**Router:** gorilla/mux — do NOT use chi

**Migrations:** golang-migrate with embedded SQL files (`migrations/*.sql` via `go:embed`). The URL scheme must use `pgx5://` (not `postgres://`) — main.go does a string replace before passing to the migrate runner.

**Templates:** edit `.templ` source files; `_templ.go` generated files are committed alongside them. Run `templ generate` after any `.templ` change.

**Auth:** `middleware/devauth.go` injects a hardcoded `User` on every request. When real auth is added, only this file changes — all handlers/services are auth-agnostic.

**Config:** loaded via `config.Load()` using viper; required env var is `DATABASE_URL`. Optional config file: `runeplan.config` in `~/.runeplan/` or cwd.

## Routes

| Method | Path | Notes |
|--------|------|-------|
| GET | `/browse` | Catalog browse; HTMX tab swap returns fragment |
| GET | `/browse/catalog/{id}` | Catalog goal detail |
| GET | `/planner` | Per-RSN goal planner |
| GET | `/profile` | User profile |
| POST | `/htmx/goals/activate` | Returns goal card fragment |
| POST | `/htmx/goals/{id}/complete` | Returns updated goal card fragment |
| POST | `/htmx/requirements/{id}/toggle` | Returns requirement row fragment |
| GET | `/htmx/skills` | Skill grid fragment |
| POST | `/htmx/sync` | Hiscores sync, returns skill grid fragment |
