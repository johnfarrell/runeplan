# Bootstrap Design — RunePlan MVP

**Date:** 2026-03-04
**Branch:** layout
**Approach:** Vertical feature slices (B)

## Objective

Produce a fully runnable RunePlan application using Go + Templ + HTMX. Each feature slice is
end-to-end complete (migration → domain → repo → service → handler → template) before the next
begins. The result is a deployable app with catalog browsing, goal planning, skill tracking, and
hiscores sync.

---

## Section 1: Structure & Domains

The existing DDD layout is preserved and extended with two new domains.

```
domain/
  skill/      — XP, Level value objects, XP table (exists)
  goal/       — Goal, Requirement, SkillThreshold, Type (exists)
  user/       — User entity; RSN ownership model (new)
  catalog/    — CatalogGoal, pre-seeded OSRS content (new)

application/
  skill/      — AggregateThresholds, satisfaction computation (extend)
  goal/       — Activate, List, Complete, ToggleRequirement (extend)
  user/       — GetUser, SyncHiscores (new)
  catalog/    — ListByType, GetByID (new)

infrastructure/
  postgres/   — one file per domain: goal_repo, user_repo, catalog_repo (extend)
  hiscores/   — HTTP client proxying OSRS CSV hiscore API (new)

interfaces/
  handler/    — one file per domain (extend)
  middleware/ — devauth stub injects hardcoded user into context (extend)
  templates/  — base layout + subdirs per page/fragment (extend)
```

### Dev-Auth Stub

`interfaces/middleware/devauth.go` constructs a `domain/user.User` with a fixed ID and injects it
via `user.SetUser(ctx, u)`. No session table, no cookies, no bcrypt. The real session middleware
replaces this file later with zero changes to any handler or service.

---

## Section 2: Database Schema

### Router & Migrations

- **Router:** gorilla/mux
- **Migrations:** golang-migrate with embedded SQL files (`migrations/` via `go:embed`)

### Tables

```sql
-- Runeplan account (no RSN here; RSNs are separate linked accounts)
users (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
)

-- Linked OSRS accounts (one user may have many RSNs)
-- skill_levels is a JSONB map of {skill_name: xp_int}
user_rsns (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  rsn          TEXT NOT NULL,
  skill_levels JSONB NOT NULL DEFAULT '{}',
  synced_at    TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, rsn)
)

-- Pre-seeded canonical OSRS goals (quests, diaries, etc.)
catalog_goals (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  canonical_key TEXT UNIQUE NOT NULL,  -- e.g. "quest.desert_treasure_2"
  title         TEXT NOT NULL,
  type          TEXT NOT NULL,         -- quest | diary | skill | boss_kc | item | custom
  description   TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
)

-- Freeform checklist requirements on a catalog goal
catalog_requirements (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  catalog_goal_id UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  description     TEXT NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
)

-- Skill level requirements on a catalog goal
catalog_skill_requirements (
  catalog_goal_id UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  skill           TEXT NOT NULL,
  level           INT  NOT NULL,
  PRIMARY KEY (catalog_goal_id, skill)
)

-- Item requirements on a catalog goal
catalog_item_requirements (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  catalog_goal_id UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  item_name       TEXT NOT NULL,
  quantity        INT  NOT NULL DEFAULT 1
)

-- Boss KC requirements on a catalog goal
catalog_boss_requirements (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  catalog_goal_id UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  boss_name       TEXT NOT NULL,
  kc              INT  NOT NULL
)

-- Prerequisite DAG within the catalog
catalog_goal_prerequisites (
  goal_id   UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  prereq_id UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  PRIMARY KEY (goal_id, prereq_id)
)

-- Per-RSN goal activations (goals belong to a specific OSRS account, not the runeplan user)
goals (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rsn_id       UUID NOT NULL REFERENCES user_rsns(id) ON DELETE CASCADE,
  catalog_id   UUID REFERENCES catalog_goals(id) ON DELETE SET NULL,
  title        TEXT NOT NULL,
  type         TEXT NOT NULL,
  notes        TEXT,
  completed    BOOLEAN NOT NULL DEFAULT false,
  completed_at TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
)

-- Per-user completion state for catalog freeform requirements (not copied — shared definition)
goal_requirement_progress (
  goal_id        UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  requirement_id UUID NOT NULL REFERENCES catalog_requirements(id) ON DELETE CASCADE,
  completed      BOOLEAN NOT NULL DEFAULT false,
  completed_at   TIMESTAMPTZ,
  PRIMARY KEY (goal_id, requirement_id)
)

-- User-added custom requirements on a goal (not from catalog)
custom_requirements (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  goal_id      UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  description  TEXT NOT NULL,
  completed    BOOLEAN NOT NULL DEFAULT false,
  completed_at TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
)
```

### Key decisions

- `skill_levels` lives on `user_rsns`, not `users`. Each RSN is a separate OSRS account with
  independent skill progression.
- Requirements are **not copied** on activation. `catalog_requirements` holds the canonical list;
  `goal_requirement_progress` holds per-user completion state. No duplication between users.
- Skill requirements are **never tracked as progress** — satisfaction is computed live by comparing
  `user_rsns.skill_levels` against `catalog_skill_requirements` for active goals.
- Quest prerequisites are satisfied when the user has a `goals` row with `completed = true` for
  the prereq's `catalog_id`.
- No `sessions` table in this milestone (dev-auth stub only).

---

## Section 3: Feature Slices

### Slice 1 — Server Bootstrap

- `main.go`: init pgx pool, run migrations, wire gorilla/mux router, graceful shutdown
- `infrastructure/postgres/connect.go`: pool setup
- `infrastructure/postgres/migrate.go`: golang-migrate runner with embedded SQL
- `migrations/001_schema.sql`: full schema above
- `migrations/002_seed_catalog.sql`: all OSRS quests and diaries
- Base templ layout: `<html>`, nav bar, HTMX 2.x + Alpine.js 3.x script tags
- `GET /` → redirect to `/browse`

### Slice 2 — Dev-Auth Stub

- `domain/user/user.go`: `User{ID string, RSNs []RSN}`, `RSN{ID, UserID, RSN string, SkillLevels map[skill.Skill]skill.XP}`
- `domain/user/context.go`: `SetUser` / `GetUser` with unexported context key
- `interfaces/middleware/devauth.go`: hardcoded User injected on every request; active RSN is the first RSN if present
- No DB interaction — fully in-memory for now

### Slice 3 — Catalog Browse

- `infrastructure/postgres/catalog_repository.go`: list by type, get by ID with requirements
- `application/catalog/service.go`: `ListByType(t goal.Type)`, `GetByID(id string)`
- `interfaces/handler/catalog.go`: `GET /browse`, `GET /browse/catalog/{id}`
- Templates: browse page with type filter tabs, goal cards; detail page with requirements list
- HTMX: tab filter swap (`hx-get`, `hx-target`) replaces goal list without full reload

### Slice 4 — Goal Planner

- `infrastructure/postgres/goal_repository.go`: activate, list by RSN, complete, toggle requirement
- `application/goal/service.go`: `Activate(rsnID, catalogID)`, `List(rsnID)`, `Complete(goalID)`, `ToggleRequirement(goalID, reqID)`
- `interfaces/handler/goal.go`: full page + HTMX fragments
- Activation: inserts `goals` row + one `goal_requirement_progress` row per `catalog_requirement` (completed=false)
- Templates: planner page, goal card fragment, requirement checklist fragment

Routes:
```
GET  /planner
POST /htmx/goals/activate          → goal card fragment
POST /htmx/goals/{id}/complete     → updated goal card fragment
POST /htmx/requirements/{id}/toggle → updated requirement row fragment
```

### Slice 5 — Skills View

- `application/skill/service.go`: `AggregateThresholds(rsnID)` — for each skill, finds the max
  required level across all active goals for this RSN; returns satisfaction status vs current XP
- `interfaces/handler/skill.go`: `GET /htmx/skills`
- Template: skill grid fragment — 24 skills, required level / current level / XP remaining,
  colour-coded satisfied (green) / unsatisfied (red) / not required (grey)

### Slice 6 — Hiscores Sync

- `infrastructure/hiscores/client.go`: GET OSRS CSV hiscore API, parse into `map[skill.Skill]skill.XP`
- `application/user/service.go`: `SyncHiscores(rsnID)` — fetch hiscores for RSN's name, persist to `user_rsns.skill_levels`
- `interfaces/handler/user.go`: profile page + sync endpoint
- Templates: profile page with RSN list, sync button per RSN; on sync returns updated skill grid fragment

Routes:
```
GET  /profile
POST /htmx/sync    → updated skill grid fragment
```

---

## Full Routing Table

| Method | Path                          | Response type     |
|--------|-------------------------------|-------------------|
| GET    | `/`                           | redirect /browse  |
| GET    | `/browse`                     | full page         |
| GET    | `/browse/catalog/{id}`        | full page         |
| GET    | `/planner`                    | full page         |
| GET    | `/profile`                    | full page         |
| POST   | `/htmx/goals/activate`        | HTMX fragment     |
| POST   | `/htmx/goals/{id}/complete`   | HTMX fragment     |
| POST   | `/htmx/requirements/{id}/toggle` | HTMX fragment  |
| GET    | `/htmx/skills`                | HTMX fragment     |
| POST   | `/htmx/sync`                  | HTMX fragment     |

---

## Out of Scope (follow-on)

- DAG graph view
- Real user auth (registration, login, sessions, bcrypt)
- Multiple RSN switcher UI (schema supports it; UI is one RSN only)
- Discord OAuth
