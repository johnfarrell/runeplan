# Delivery & Conventions

---

## Phased Delivery Plan

### Phase 1 — MVP

The MVP is a fully functional self-hostable planner. All items below must be complete before Phase 2 begins.

| # | Deliverable | Notes |
|---|---|---|
| 1 | Docker Compose — 2 services healthy | `docker compose up` + `GET /health` |
| 2 | `001_init.sql` — full schema | All tables including `goal_skill_requirements` |
| 3 | Auth — register, login, logout, SessionMiddleware | bcrypt cost 12, HTTP-only cookie, 30-day session, HTML redirects |
| 4 | `GET /profile` + `PATCH /htmx/user/me` | Skills as JSONB; partial update merges, not replaces |
| 5 | `POST /htmx/user/sync` — Hiscores proxy | Parse CSV in fixed skill order; 429 on rapid re-sync; returns SkillGrid fragment |
| 6 | Goals CRUD — `/htmx/goals` | Including copy-on-activate transaction for catalog goals |
| 7 | Skill thresholds — `/htmx/skills` + goal skill endpoints | Full SkillLadder aggregation query; returns fragment |
| 8 | Requirements CRUD + deduplication | Upsert on `canonical_key`; `shared_by_goals` rendered in template |
| 9 | `002_seed_diaries.sql` — 48 diary goals | All regions, all tiers, skill thresholds + quest requirements |
| 10 | Base layout template | HTML shell: HTMX 2.x, Alpine.js 3.x, Tailwind CSS, nav |
| 11 | Planner page — 3-panel layout | `GoalSidebar` + `RequirementList` + `RequirementDetail` templates |
| 12 | `SkillLadderRow` component | Alpine expand/collapse; tick marks on bar; read-only completion |
| 13 | Browse page + add goal modal | Catalog list (HTMX-loaded) + custom goal form + Alpine modal |
| 14 | Profile page — skill grid + sync UI | Manual skill entry + Hiscores sync button + last-synced timestamp |
| 15 | `GET /api/user/export` — JSON export | Full dump of user's goals, requirements, skill levels |
| 16 | Static assets committed | `htmx.min.js`, `alpine.min.js`, `app.css` embedded via go:embed |
| 17 | `templ generate` → `_templ.go` committed | All `.templ` files have corresponding generated files |

### Phase 2 — Depth

| # | Deliverable | Notes |
|---|---|---|
| 1 | `003_seed_quests.sql` — ~200 quests | Skill requirements + quest prerequisites |
| 2 | Discord OAuth | Guard with `DISCORD_CLIENT_ID` env var; 501 if absent |
| 3 | Graph view | Dependency graph for goals + requirements (Alpine/SVG or D3 loaded lazily) |
| 4 | Requirement notes — inline editable | Per-requirement free-text, auto-saved on blur via `hx-trigger="blur"` |

### Phase 3 — Polish

| # | Deliverable | Notes |
|---|---|---|
| 1 | OSRS Wiki deep links | Each requirement and goal links to its wiki page |
| 2 | Shareable read-only plan URLs | Public token-based view of a user's active goals |
| 3 | Kill count progress (manual updates) | Inline counter input, `hx-trigger="change"` to persist |
| 4 | Self-hosting README | Step-by-step guide with screenshots |

---

## Out of Scope

If a feature is not in the delivery plan above, it is out of scope. Do not implement it. Raise it for discussion before writing any code.

| Feature | Reason excluded |
|---|---|
| Mobile app | Desktop-first. Mobile-responsive is a Phase 3 stretch goal. |
| Analytics / dashboards | Explicitly out of scope. No XP tracking, no time estimates. |
| Recommendation engine | No "optimal order" suggestions or heuristic ranking. |
| Notifications / alerts | No email, push, or in-app notification system. |
| Community features | No sharing of custom goals, no commenting, no social mechanics. |
| Group Ironman | Not in v1. Revisit only if there is demonstrated demand. |
| Runtime data scraping | All seed data is SQL. No wiki API calls at runtime, ever. |
| Redis | Sessions in Postgres. No cache layer needed at this scale. |
| GraphQL | REST-like HTML endpoints only. No GraphQL, tRPC, or other query layers. |
| External component libraries | Tailwind utilities only. No shadcn, MUI, Chakra, or similar. |
| React / Vue / Svelte | Templ + HTMX + Alpine.js only. No JavaScript framework. |
| Vite / webpack / esbuild | No JavaScript build step. Static files committed directly. |
| Ironman mode | Item requirement logic is not mode-aware in v1. |

---

## Conventions

### Backend (Go)

- All HTTP handlers have the signature `func Name(pool *pgxpool.Pool) http.HandlerFunc`.
- HTML responses use `httputil.Render(w, r, status, component)` — never `templ.Handler` directly.
- JSON responses use `httputil.WriteJSON(w, status, v)` — only for `/api/*` endpoints.
- HTML errors: `httputil.Render(w, r, status, components.Error("message"))`.
- JSON errors: `httputil.WriteJSON(w, status, map[string]string{"error": "..."})`.
- All database queries use pgx positional parameters (`$1`, `$2`, ...) — never string interpolation.
- Context is threaded through all DB calls: `pool.QueryRow(r.Context(), ...)`.
- UUIDs generated with `github.com/google/uuid` v4.
- Passwords hashed with `golang.org/x/crypto/bcrypt`, cost 12.
- Run `templ generate` before `go build`. Generated `_templ.go` files are committed.

### Templates (Templ)

- Each domain owns its templates under `internal/<domain>/templates/`.
- Shared layout and components live in `internal/templates/`.
- Full pages use `layout.Base(title, content)`. HTMX fragments do not use the layout.
- No inline `style=` attributes except for dynamically computed values (e.g. progress bar widths).
- Tailwind class names only — no CSS modules, no custom class names except base reset in `app.css.src`.
- All user-facing text is in English. No i18n infrastructure in v1.

### HTMX

- HTMX endpoints live under `/htmx/`. They return HTML fragments.
- Full page routes are top-level (`/`, `/planner`, `/browse`, `/profile`).
- HTMX `HX-Redirect` header is used (instead of 302) when redirecting from HTMX requests.
- Out-of-band swaps (`hx-swap-oob="true"`) are used sparingly and only when a single action must update multiple independent page regions.

### Database

- All identifiers use `snake_case`.
- Every table has a `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()` column.
- Foreign keys always include `ON DELETE CASCADE` unless there is an explicit reason not to.
- Never modify existing migration files — only append new ones.
- Migration filenames: `NNN_description.sql` where NNN is zero-padded (`001`, `002`, ...).

### Git & Branching

- `main` is always deployable.
- Feature branches: `feature/short-description`.
- Commit messages: imperative mood, present tense — e.g. `Add skill ladder query`.
- PRs require at least one review before merging to `main`.
- No secrets in the repository. All credentials go in `.env`, which is gitignored.

---

## Dependency Reference

### Go

| Package | Version | Purpose |
|---|---|---|
| `github.com/a-h/templ` | v0.3.x | Type-safe HTML templates (SSR) |
| `github.com/go-chi/chi/v5` | v5.1.0 | HTTP router |
| `github.com/jackc/pgx/v5` | v5.6.0 | PostgreSQL driver + connection pool |
| `github.com/golang-migrate/migrate/v4` | v4.17.1 | SQL migration runner |
| `github.com/google/uuid` | v1.6.0 | UUID v4 generation |
| `golang.org/x/crypto` | v0.22.0 | bcrypt password hashing |
| `encoding/json` | stdlib | JSON encode/decode (export + catalog) |
| `net/http` | stdlib | HTTP server |
| `crypto/rand` | stdlib | Session token generation |
| `embed` | stdlib | Embed static assets in binary |

### Frontend (committed static files)

| File | Version | Source |
|---|---|---|
| `static/htmx.min.js` | 2.x | `https://unpkg.com/htmx.org@2.x/dist/htmx.min.js` |
| `static/alpine.min.js` | 3.x | `https://unpkg.com/alpinejs@3.x/dist/cdn.min.js` |
| `static/app.css` | — | Compiled by `tailwindcss` CLI from `static/app.css.src` |

### Build Tools (not in production image)

| Tool | Version | Purpose |
|---|---|---|
| `templ` CLI | v0.3.x | Compile `.templ` files to `_templ.go` |
| Tailwind CSS standalone CLI | 3.x | Compile `app.css` from `.templ` sources |
| `air` (optional) | latest | Live reload during development |

### Infrastructure

| Tool | Version | Purpose |
|---|---|---|
| Docker Engine | 24+ | Container runtime |
| Docker Compose | v2+ | Multi-container orchestration |
| PostgreSQL | 16 (alpine) | Primary database |
| Go | 1.22+ | Backend language + binary serves entire app |
