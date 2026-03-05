# Architecture

## System Overview

RunePlan is a self-hostable, open-source web application for Old School RuneScape players to plan and track in-game goals. Its defining feature is intelligent deduplication of shared requirements across goals: a shared skill level or quest appears once across all active goals, and satisfying it propagates automatically.

```
┌──────────────────────────────────────────┐
│               Docker Compose              │
│                                           │
│  ┌────────────────────────────────────┐   │
│  │  Backend (Go)                       │   │
│  │  chi router · Templ SSR · HTMX     │   │
│  │  Alpine.js · Tailwind CSS          │   │
│  │  Serves HTML + static assets       │   │
│  └──────────────┬─────────────────────┘   │
│                 │                          │
│  ┌──────────────▼─────────────────────┐   │
│  │  PostgreSQL 16                      │   │
│  │  (user data + seeded game data)    │   │
│  └────────────────────────────────────┘   │
└──────────────────────────────────────────┘
```

The Go binary serves HTML pages, HTMX fragments, and static assets. There is no separate frontend container and no Nginx proxy. The browser talks directly to `:8080`.

---

## Core Principles

- **Self-host with a single `docker compose up`** — no manual database setup, no external services beyond optional Discord OAuth.
- **Skill level requirements use a threshold model, not checkboxes.** Reaching level 72 automatically satisfies any requirement for 70 or below. Completion is computed, never stored.
- **Shared requirements across goals are resolved server-side.** Templates render a flat deduplicated list — no client-side deduplication logic.
- **The codebase is MIT licensed and structured for external contributors.**

---

## Technology Decisions

### Backend — Go

Go was chosen for its simplicity, fast compile times, small binary size (ideal for Docker), and strong standard library. The router (`chi`), database driver (`pgx`), and template engine (`a-h/templ`) are the only non-stdlib dependencies that touch the request path. There is no ORM — all queries are hand-written SQL.

### Templating — a-h/templ

[Templ](https://templ.guide) compiles `.templ` files to type-safe Go functions. Templates are Go code — no runtime template parsing, no `html/template` string injection risk, full IDE support. The `templ generate` command produces `_templ.go` files which are committed alongside the `.templ` source.

### Interactivity — HTMX + Alpine.js

[HTMX](https://htmx.org) (v2.x) handles server-driven partial page updates via HTML attributes (`hx-get`, `hx-post`, `hx-target`, etc.). This eliminates the need for a JavaScript framework for data fetching and DOM swapping.

[Alpine.js](https://alpinejs.dev) (v3.x) handles purely client-side UI state — toggle visibility, form validation state, modal open/close — where a round-trip to the server is unnecessary.

### Styling — Tailwind CSS

Tailwind standalone CLI generates `static/app.css` from `.templ` source files. The compiled CSS is committed to the repository and embedded in the binary via `go:embed`.

### Database — PostgreSQL

Sessions are stored in Postgres, eliminating the need for Redis. At the expected scale (hundreds to low thousands of users on a self-hosted instance), a single Postgres instance with proper indexing is more than sufficient.

### No React, No Node.js, No Nginx

These were explicitly rejected to keep the deployment footprint minimal. A self-hoster needs only Docker and a `.env` file. The production image is a single Go binary with embedded templates, CSS, and JavaScript files.

---

## Domain-Driven Design

The backend is organized around domain packages under `internal/`. Each domain owns its models, database queries, HTTP handlers, and Templ templates.

```
internal/
  auth/               # Registration, login, session middleware, Discord OAuth
  goals/              # Goals CRUD, skill thresholds, copy-on-activate
  requirements/       # Non-skill requirements CRUD, deduplication
  user/               # User profile, skill levels, Hiscores sync
  catalog/            # Read-only pre-seeded diary/quest data
  db/                 # pgx pool, migration runner
  httputil/           # Shared render + JSON response helpers
  templates/          # Shared layout and component templates
```

Each domain package (except `db`, `httputil`, `templates`) follows this layout:

```
<domain>/
  model.go            # Domain types (structs + constants)
  repository.go       # All SQL queries for this domain
  handler.go          # HTTP handlers — func Name(pool) http.HandlerFunc
  templates/          # .templ files and generated _templ.go files
```

---

## Repository Structure

```
runeplan/
  cmd/
    server/
      main.go                       # Wire dependencies, start server

  internal/
    domain/                         # Pure Go — no SQL, no HTTP, no templates
      goal/
        goal.go                     # Goal, SkillLadder, SkillThreshold entities
        repository.go               # GoalRepository, SkillLadderRepository interfaces
      requirement/
        requirement.go              # Requirement entity
        repository.go               # RequirementRepository interface
      user/
        user.go                     # User entity
        session.go                  # Session entity
        repository.go               # UserRepository, SessionRepository interfaces
      catalog/
        catalog.go                  # CatalogGoal, Diary, Quest value objects
        repository.go               # CatalogRepository interface

    application/                    # Use cases — imports domain, nothing else
      goal/
        service.go                  # AddGoal, RemoveGoal, LinkSkill, ListGoals
        service_test.go
      requirement/
        service.go                  # CreateRequirement, LinkToGoal, UnlinkFromGoal
        service_test.go
      user/
        service.go                  # GetMe, UpdateMe, SyncHiscores
        service_test.go
      auth/
        service.go                  # Login, Logout, DiscordOAuth flow
        service_test.go
      catalog/
        service.go                  # ListDiaries, ListQuests, ListSkills
        service_test.go

    infrastructure/                 # Implements domain interfaces
      postgres/
        goal_repository.go          # Implements domain/goal.Repository
        requirement_repository.go   # Implements domain/requirement.Repository
        user_repository.go          # Implements domain/user.Repository
        session_repository.go       # Implements domain/user.SessionRepository
        catalog_repository.go       # Implements domain/catalog.Repository
        db.go                       # pgxpool setup
        migrations.go               # golang-migrate runner
      hiscores/
        client.go                   # External OSRS hiscores API
        model.go                    # Raw API response types (internal to package)
      discord/
        client.go                   # Discord OAuth API calls
        model.go                    # Raw Discord response types

    interfaces/                     # HTTP — imports application only
      handler/
        goal.go                     # Goal HTTP handlers
        requirement.go              # Requirement HTTP handlers
        user.go                     # User HTTP handlers
        auth.go                     # Auth + Discord OAuth handlers
        catalog.go                  # Catalog HTTP handlers
      middleware/
        auth.go                     # SessionMiddleware, RequireAuth
        logging.go
      templates/
        goal/
          sidebar.templ             # Left panel: active goals list
          item.templ                # Single goal row (HTMX swap target)
          skill_ladder_row.templ    # Expandable skill ladder row
          add_modal.templ           # Catalog browser modal
        requirement/
          list.templ                # Center panel: skill ladders + requirements
          row.templ                 # Single requirement row (HTMX swap target)
          detail.templ              # Right panel: detail view
        user/
          profile.templ             # Profile page
          skill_grid.templ          # Skill level grid + sync UI
        auth/
          login.templ               # Login page
          register.templ            # Register page
        catalog/
          browse.templ              # Catalog browse page
        shared/
          layouts/
            base.templ              # HTML shell, head, scripts, body
            nav.templ               # Top navigation bar
          components/
            error.templ             # Reusable error display
            planner.templ           # 3-panel planner layout
            empty_state.templ       # Reusable empty list state

    config/
      config.go                     # Typed config loaded from env vars

  migrations/
    001_init.sql
    002_seed_diaries.sql
    003_seed_quests.sql

  static/
    htmx.min.js
    alpine.min.js
    app.css

  tailwind.config.js
  go.mod
  go.sum
  Dockerfile
  docker-compose.yml
  .env.example
  README.md
  LICENSE
```

---

## Request Flow

**Full page load:**
1. Browser requests `GET /planner`
2. `SessionMiddleware` validates cookie → attaches user to context
3. Handler queries goals + requirements + skill ladders
4. Templ renders full HTML page (base layout wrapping planner panels)
5. Response: complete HTML document with HTMX/Alpine/Tailwind loaded

**HTMX partial update:**
1. User clicks "Mark complete" on a requirement → `hx-patch="/htmx/requirements/{id}" hx-target="#req-{id}" hx-swap="outerHTML"`
2. Handler updates DB row, renders updated `requirement_row` component
3. HTMX swaps the single row in-place — no full page reload

**JSON data endpoint:**
1. Browser or HTMX requests `GET /api/catalog/skills`
2. Handler returns `application/json` response
3. Used for catalog browsing and data export only

---

## Static Asset Serving

Static files are embedded in the binary using `go:embed` and served from memory:

```go
//go:embed static
var staticFiles embed.FS

r.Handle("/static/*", http.FileServer(http.FS(staticFiles)))
```

The Tailwind build step (`tailwindcss --input ... --output static/app.css`) is run before `go build`. In development, run `tailwindcss --watch` alongside `go run ./cmd/server`.

---

## Environment Variables

All configuration is provided via environment variables. Copy `.env.example` to `.env` and fill in values before running.

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | Yes | Full postgres connection string |
| `DB_PASSWORD` | Yes | Used by docker-compose to set the Postgres password |
| `SESSION_SECRET` | No | Entropy for session token generation (crypto/rand is used directly) |
| `DISCORD_CLIENT_ID` | No | Discord OAuth app client ID — leave blank to disable |
| `DISCORD_CLIENT_SECRET` | No | Discord OAuth app client secret |

```bash
# .env.example
DB_PASSWORD=changeme
DATABASE_URL=postgres://runeplan:${DB_PASSWORD}@db:5432/runeplan

# Optional — leave blank to disable Discord login
DISCORD_CLIENT_ID=
DISCORD_CLIENT_SECRET=
```

If `DISCORD_CLIENT_ID` is empty, the Discord OAuth routes return `501 Not Implemented` and the UI hides the Discord login button.
