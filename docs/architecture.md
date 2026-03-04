# Architecture

## System Overview

RunePlan is a self-hostable, open-source web application for Old School RuneScape players to plan and track in-game goals. It handles both structured goals (Achievement Diaries, quests, skill milestones) and fully freeform goals (boss kill counts, item acquisition, custom checklists). Its defining feature is intelligent deduplication of shared requirements across goals: a shared skill level or quest appears once, never duplicated, and satisfying it propagates across every goal that depends on it.

```
┌──────────────────────────────────────────────────┐
│                  Docker Compose                   │
│                                                   │
│  ┌──────────────┐    ┌──────────────────────┐    │
│  │  Frontend    │    │  Backend (Go)         │    │
│  │  React + TS  │◄───►  REST API + Auth      │    │
│  │  Nginx       │    │                       │    │
│  └──────────────┘    └──────────┬────────────┘   │
│                                 │                 │
│                      ┌──────────▼────────────┐    │
│                      │  PostgreSQL            │    │
│                      │  (user data +          │    │
│                      │   seeded game data)    │    │
│                      └────────────────────────┘   │
└───────────────────────────────────────────────────┘
```

The frontend is a static React SPA served by Nginx. Nginx also proxies all `/api/*` requests to the Go backend, so the browser only ever talks to one origin. No CORS configuration is required.

---

## Core Principles

- **Self-host with a single `docker compose up`** — no manual database setup, no external services beyond optional Discord OAuth.
- **Skill level requirements use a threshold model, not checkboxes.** Reaching level 72 automatically satisfies any requirement for 70 or below. Completion is computed, never stored.
- **Shared requirements across goals are resolved server-side.** The frontend renders a flat deduplicated list — no client-side deduplication logic.
- **The codebase is MIT licensed and structured for external contributors.**

---

## Technology Decisions

### Backend — Go

Go was chosen for its simplicity, fast compile times, small binary size (ideal for Docker), and strong standard library. The router (`chi`) and database driver (`pgx`) are the only non-stdlib dependencies that touch the request path. There is no ORM — all queries are hand-written SQL.

### Frontend — React 18 + TypeScript

Plain React with `useReducer` and Context — no external state library. Tailwind CSS utility classes only — no component library. Native `fetch` — no Axios. This keeps the bundle small and the dependency surface minimal. D3.js is imported only in `GoalGraph.tsx` for the Phase 2 graph view.

### Database — PostgreSQL

Sessions are stored in Postgres, eliminating the need for Redis. At the expected scale (hundreds to low thousands of users on a self-hosted instance), a single Postgres instance with proper indexing is more than sufficient.

### No Redis, No SSR, No GraphQL

These were explicitly rejected to keep the deployment footprint minimal. A self-hoster should need only Docker and a `.env` file.

---

## Repository Structure

```
runeplan/
  backend/
    cmd/server/main.go          # Entrypoint — wires router, runs migrations
    internal/
      auth/                     # Session management, Discord OAuth
      goals/                    # Goals CRUD + skill threshold logic
      requirements/             # Non-skill requirements + deduplication
      user/                     # User profile, skill levels, Hiscores sync
      catalog/                  # Read-only pre-seeded diary/quest data
      db/                       # pgx pool, migration runner
    migrations/
      001_init.sql              # Schema: users, sessions, goals, requirements,
                                #   goal_requirements, goal_skill_requirements
      002_seed_diaries.sql      # All 48 Achievement Diary goals + thresholds
      003_seed_quests.sql       # All OSRS quests (Phase 2)
    Dockerfile
  frontend/
    src/
      api/                      # Typed fetch wrappers (one file per resource)
      components/               # Stateless presentational components
      pages/                    # Planner, Browse, Profile
      context/AppContext.tsx    # Global state via useReducer
      types/index.ts            # TypeScript interfaces mirroring Go structs
    Dockerfile                  # Multi-stage: build then Nginx serve
    nginx.conf
  docs/                         # This documentation
  docker-compose.yml
  .env.example
  README.md
  LICENSE                       # MIT
```

---

## Environment Variables

All configuration is provided via environment variables. Copy `.env.example` to `.env` and fill in values before running.

| Variable | Required | Description |
|---|---|---|
| `DB_PASSWORD` | Yes | PostgreSQL password for the `runeplan` user |
| `SESSION_SECRET` | Yes | Random string (min 32 chars) for session signing |
| `DISCORD_CLIENT_ID` | No | Discord OAuth app client ID — leave blank to disable |
| `DISCORD_CLIENT_SECRET` | No | Discord OAuth app client secret |

```bash
# .env.example
DB_PASSWORD=changeme
SESSION_SECRET=replace-with-long-random-string-at-least-32-chars

# Optional — leave blank to disable Discord login
DISCORD_CLIENT_ID=
DISCORD_CLIENT_SECRET=
```

If `DISCORD_CLIENT_ID` is empty, the `/api/auth/discord` route returns `501 Not Implemented` and the frontend hides the Discord login button.
