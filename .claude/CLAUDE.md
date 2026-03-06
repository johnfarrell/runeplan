# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

RunePlan is a self-hostable OSRS (Old School RuneScape) goal-planning web app. Players activate pre-seeded Achievement Diary / quest goals and add custom skill thresholds; the app deduplicates shared requirements across goals.

**Status:** Documentation-complete, implementation not yet started. All specs live in `docs/`.

## Commands

```bash
# Start full stack (db + backend)
docker compose up

# Backend only (requires DATABASE_URL env var)
DATABASE_URL=postgres://runeplan:changeme@localhost:5432/runeplan go run ./cmd/server

# Generate Templ templates (required before go build)
templ generate

# Watch templates during development
templ generate --watch

# Compile Tailwind CSS
tailwindcss --input static/app.css.src --output static/app.css --content "./internal/**/*.templ"

# Watch Tailwind during development
tailwindcss --input static/app.css.src --output static/app.css --watch --content "./internal/**/*.templ"

# Go tests
go test ./...

# Type-check (build without running)
go build ./...
```

## Architecture

Two Docker services: Go backend (port 8080) → PostgreSQL 16. The Go binary serves HTML pages, HTMX fragments, static assets, and JSON API endpoints. No Nginx, no frontend container.

**Backend** (`internal/<domain>/`): Go 1.22 + chi v5 router. Domain-driven design — each domain (`auth`, `goals`, `requirements`, `user`, `catalog`) owns its models, queries, handlers, and templates. No ORM — hand-written SQL with pgx/v5. Session auth via HTTP-only cookies; optional Discord OAuth.

**Templates**: [a-h/templ](https://templ.guide) compiles `.templ` files to type-safe `_templ.go` Go functions. Run `templ generate` before `go build`. Both `.templ` and `_templ.go` files are committed.

**Interactivity**: [HTMX 2.x](https://htmx.org) for server-driven partial updates. [Alpine.js 3.x](https://alpinejs.dev) for client-side UI state (modals, toggles). No JavaScript framework.

**Styling**: Tailwind CSS standalone CLI. `static/app.css` is compiled from `.templ` sources and committed.

**Static assets**: `static/` — htmx.min.js, alpine.min.js, app.css — committed and embedded via `go:embed`.

**Database**: 6 tables — `users`, `sessions`, `goals`, `requirements`, `goal_requirements`, `goal_skill_requirements`. Skill levels stored as JSONB on `users`; satisfaction computed at query time, never stored. Pre-seeded goals/requirements use `canonical_key` for deduplication. Migrations in `migrations/` are append-only and embedded in the binary.

**Key flow**: User activates a catalog goal → copy-on-activate transaction clones goal+requirements to user's plan → user syncs Hiscores (backend proxies OSRS CSV API) → `GET /htmx/skills` aggregates thresholds, computes satisfaction from current skill levels, returns HTML fragment.

## Conventions

**Go:**
- Handlers: `func Name(pool *pgxpool.Pool) http.HandlerFunc`
- HTML responses: `httputil.Render(w, r, status, component)` — never `templ.Handler` directly
- JSON responses: `httputil.WriteJSON(w, status, v)` — only for `/api/*` endpoints
- HTML errors: `httputil.Render(w, r, status, components.Error("msg"))`
- Context user: `auth.GetUser(ctx)` — unexported contextKey type
- UUIDs v4 everywhere; bcrypt cost 12 for passwords
- Stdlib-first; no frameworks beyond chi + pgx + templ

**Routing:**
- Full pages: top-level paths (`/planner`, `/browse`, `/profile`)
- HTMX fragments: `/htmx/*` prefix
- JSON data: `/api/*` prefix
- HTMX redirect on 401: set `HX-Redirect` header, not 302

**Database:**
- `snake_case` identifiers, `created_at` on every table
- `ON DELETE CASCADE` on foreign keys
- Never modify existing migration files — append new ones

**Templ:**
- `templ generate` must run before `go build`
- Generated `_templ.go` files are committed alongside `.templ` files
- Never use `go mod tidy` until all deps are actually imported in code

**Git:** `main` is always deployable. Imperative commit messages.

## Module

`github.com/johnfarrell/runeplan` (in `go.mod`)

## Reference Docs

- `docs/architecture.md` — System diagram, DDD structure, tech decisions
- `docs/backend.md` — Go conventions, router wiring, auth, response helpers
- `docs/frontend.md` — Templ patterns, HTMX usage, Alpine.js, Tailwind CSS
- `docs/database.md` — Full schema DDL, migration strategy
- `docs/api.md` — All routes (page routes, HTMX fragments, JSON API)
- `docs/delivery.md` — Phase 1 MVP checklist, out-of-scope features
- `docs/deployment.md` — Docker Compose, Dockerfile, dev workflow
