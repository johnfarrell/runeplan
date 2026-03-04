# Delivery & Conventions

---

## Phased Delivery Plan

### Phase 1 — MVP

The MVP is a fully functional self-hostable planner. All 17 items below must be complete before Phase 2 begins.

| # | Deliverable | Notes |
|---|---|---|
| 1 | Docker Compose — all 3 services healthy | Verify with `docker compose up` + `GET /health` |
| 2 | `001_init.sql` — full schema | All tables including `goal_skill_requirements` |
| 3 | Auth — register, login, logout, session middleware | bcrypt cost 12, HTTP-only cookie, 30-day session |
| 4 | `GET /api/user/me` + `PATCH /api/user/me` | Skills as JSONB; partial update merges, not replaces |
| 5 | `POST /api/user/sync` — Hiscores proxy | Parse CSV in fixed skill order; 429 on rapid re-sync |
| 6 | Goals CRUD — `/api/goals` | Including copy-on-activate transaction for catalog goals |
| 7 | Skill thresholds — `/api/skills` + goal skill endpoints | Full `SkillLadder` aggregation query |
| 8 | Requirements CRUD + deduplication | Upsert on `canonical_key`; `shared_by_goals` in response |
| 9 | `002_seed_diaries.sql` — 48 diary goals | All regions, all tiers, skill thresholds + quest requirements |
| 10 | React app scaffold — Vite + TypeScript + Tailwind | `strict: true`; all types defined in `src/types/index.ts` |
| 11 | API layer — `src/api/` | Typed `apiFetch` wrapper; all endpoints covered |
| 12 | `AppContext` — `useReducer` + all action types | State shape matches API response types exactly |
| 13 | Planner page — 3-panel layout | `GoalSidebar` + `RequirementList` + `RequirementDetail` |
| 14 | `SkillLadderRow` component | Expandable, tick marks on bar, no checkbox, read-only completion |
| 15 | Browse page + `AddGoalModal` | Catalog list + custom goal form + all goal categories |
| 16 | Profile page — skill grid + sync UI | Manual skill entry + Hiscores sync button + last-synced timestamp |
| 17 | `GET /api/user/export` — JSON export | Full dump of user's goals, requirements, skill levels |

### Phase 2 — Depth

| # | Deliverable | Notes |
|---|---|---|
| 1 | `003_seed_quests.sql` — ~200 quests | Skill requirements + quest prerequisites |
| 2 | Discord OAuth | Guard with `DISCORD_CLIENT_ID` env var; 501 if absent |
| 3 | Graph view (D3.js) | Dependency graph for goals + requirements; import D3 only in `GoalGraph.tsx` |
| 4 | Requirement notes — inline editable | Per-requirement free-text notes, auto-saved on blur |

### Phase 3 — Polish

| # | Deliverable | Notes |
|---|---|---|
| 1 | OSRS Wiki deep links | Each requirement and goal links to its wiki page |
| 2 | Shareable read-only plan URLs | Public token-based view of a user's active goals |
| 3 | Kill count progress (manual updates) | Inline counter input on `kill_count` requirements |
| 4 | Self-hosting README | Step-by-step guide with screenshots |

---

## Out of Scope

If a feature is not in the delivery plan above, it is out of scope. Do not implement it. Raise it for discussion before writing any code.

| Feature | Reason excluded |
|---|---|
| Mobile app | Desktop-first. Mobile-responsive is a Phase 3 stretch goal, not a separate app. |
| Analytics / dashboards | Explicitly out of scope by design. No XP tracking, no time estimates. |
| Recommendation engine | Do not add "optimal order" suggestions or any heuristic ranking. |
| Notifications / alerts | No email, push, or in-app notification system. |
| Community features | No sharing of custom goals, no commenting, no social mechanics. |
| Group Ironman | Not in v1. Revisit only if there is demonstrated demand. |
| Runtime data scraping | All seed data is SQL. No wiki API calls at runtime, ever. |
| Redis | Sessions in Postgres. No cache layer needed at this scale. |
| GraphQL | REST only. No GraphQL, tRPC, or other query layers. |
| External component libraries | Tailwind utilities only. No shadcn, MUI, Chakra, or similar. |
| Server-side rendering | Plain SPA served by Nginx. No Next.js or Remix. |
| Ironman mode | Item requirement logic is not mode-aware in v1. |

---

## Conventions

### Backend (Go)

- All HTTP handlers have the signature `func(pool *pgxpool.Pool) http.HandlerFunc`.
- JSON encoding/decoding uses `encoding/json` from stdlib — no third-party serializer.
- All database queries use pgx positional parameters (`$1`, `$2`, ...) — never string interpolation.
- Errors returned to clients are always `{ "error": "description" }` — never raw Go error strings.
- Context is threaded through all DB calls: `pool.QueryRow(r.Context(), ...)`.
- UUIDs generated with `github.com/google/uuid` v4.
- Passwords hashed with `golang.org/x/crypto/bcrypt`, cost 12.

### Frontend (TypeScript)

- `strict: true` in tsconfig. No `any`. No implicit returns.
- All API functions are in `src/api/` and return typed `Promise<T>`.
- Components accept typed props — no prop drilling through untyped objects.
- No inline styles except where Tailwind cannot express a dynamic runtime value (e.g. progress bar widths).
- Tailwind class names only — no CSS modules, no styled-components.
- All user-facing text is in English. No i18n infrastructure in v1.

### Database

- All identifiers use `snake_case`.
- Every table has a `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()` column.
- Foreign keys always include `ON DELETE CASCADE` unless there is an explicit reason not to.
- Never modify existing migration files — only append new ones.
- Migration filenames: `NNN_description.sql` where NNN is zero-padded (`001`, `002`, ...).

### Git & Branching

- `main` is always deployable.
- Feature branches: `feature/short-description`.
- Commit messages: imperative mood, present tense — e.g. `Add skill ladder query`, not `Added skill ladder query`.
- PRs require at least one review before merging to `main`.
- No secrets in the repository. All credentials go in `.env`, which is gitignored.

---

## Dependency Reference

### Backend

| Package | Version | Purpose |
|---|---|---|
| `github.com/go-chi/chi/v5` | v5.1.0 | HTTP router |
| `github.com/jackc/pgx/v5` | v5.6.0 | PostgreSQL driver + connection pool |
| `github.com/golang-migrate/migrate/v4` | v4.17.1 | SQL migration runner |
| `github.com/google/uuid` | v1.6.0 | UUID v4 generation |
| `golang.org/x/crypto` | v0.22.0 | bcrypt password hashing |
| `encoding/json` | stdlib | JSON encode/decode |
| `net/http` | stdlib | HTTP server |
| `crypto/rand` | stdlib | Session token generation |

### Frontend

| Package | Version | Purpose |
|---|---|---|
| `react` | 18.x | UI framework |
| `react-dom` | 18.x | DOM renderer |
| `typescript` | 5.x | Type system (`strict: true`) |
| `vite` | 5.x | Build tool + dev server |
| `tailwindcss` | 3.x | Utility CSS — no component library on top |
| `d3` | 7.x | Dependency graph (Phase 2 — import only in `GoalGraph.tsx`) |
| `@types/react` | 18.x | React TypeScript types |
| `@types/d3` | 7.x | D3 TypeScript types (Phase 2) |

### Infrastructure

| Tool | Version | Purpose |
|---|---|---|
| Docker Engine | 24+ | Container runtime |
| Docker Compose | v2+ | Multi-container orchestration |
| PostgreSQL | 16 (alpine) | Primary database |
| Nginx | latest alpine | Static file serving + API proxy |
| Go | 1.22+ | Backend language |
| Node.js | 20 (alpine) | Frontend build only — not present in production image |
