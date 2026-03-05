# Database

## Migration Strategy

All schema changes are managed by `golang-migrate` and run automatically on backend startup. Migration files live in `migrations/` and are numbered sequentially. They are embedded in the Go binary via `go:embed` so no external files are needed at runtime.

**Never modify existing migration files.** If a change is needed, add a new migration file. The runner tracks which migrations have been applied and will only execute new ones.

Migration filenames follow the pattern: `NNN_description.sql` where `NNN` is zero-padded (e.g. `001`, `002`).

---

## Schema

### `users`

```sql
CREATE TABLE users (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email      TEXT UNIQUE NOT NULL,
  password   TEXT,                -- bcrypt hash; NULL for OAuth-only accounts
  rsn        TEXT NOT NULL DEFAULT '',
  skills     JSONB NOT NULL DEFAULT '{}', -- {"Agility": 63, "Slayer": 67, ...}
  last_hiscores_sync TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Skill levels are stored as a JSONB map on the user row — not as separate rows in a skills table. This is intentional. See [Skills Storage Rationale](#skills-storage-rationale) below.

---

### `sessions`

```sql
CREATE TABLE sessions (
  id         TEXT PRIMARY KEY,        -- random token (crypto/rand, 32 bytes, hex-encoded)
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON sessions(user_id);
CREATE INDEX ON sessions(expires_at); -- for cleanup job
```

Sessions are stored in Postgres — no Redis required. A background goroutine runs every 24 hours to `DELETE FROM sessions WHERE expires_at < NOW()`.

---

### `goals`

```sql
CREATE TABLE goals (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID REFERENCES users(id) ON DELETE CASCADE, -- NULL for pre-seeded catalog goals
  name         TEXT NOT NULL,
  description  TEXT NOT NULL DEFAULT '',
  category     TEXT NOT NULL,
    -- 'achievement_diary' | 'quest' | 'skill_milestone'
    -- | 'boss_kill' | 'item_obtain' | 'custom'
  diary_region TEXT,   -- NULL for non-diary goals
  diary_tier   TEXT,   -- 'easy' | 'medium' | 'hard' | 'elite' | NULL
  is_preseeded BOOLEAN NOT NULL DEFAULT FALSE,
  is_completed BOOLEAN NOT NULL DEFAULT FALSE,
  sort_order   INT NOT NULL DEFAULT 0,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON goals(user_id);
```

Pre-seeded catalog goals have `user_id = NULL` and `is_preseeded = TRUE`. When a user activates a catalog goal, it is copied into a user-owned row in the same transaction that copies its requirements and skill thresholds.

---

### `requirements` (non-skill only)

```sql
CREATE TABLE requirements (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID REFERENCES users(id) ON DELETE CASCADE, -- NULL for pre-seeded
  label         TEXT NOT NULL,
  type          TEXT NOT NULL,
    -- 'quest' | 'kill_count' | 'item_obtain' | 'freeform'
  is_preseeded  BOOLEAN NOT NULL DEFAULT FALSE,
  is_completed  BOOLEAN NOT NULL DEFAULT FALSE,
  notes         TEXT NOT NULL DEFAULT '',
  canonical_key TEXT,  -- e.g. 'quest:Priest in Peril'; NULL for freeform requirements

  -- Type-specific fields (all nullable; populate based on type):
  quest_name    TEXT,
  boss_name     TEXT,
  kill_target   INT,
  kill_current  INT NOT NULL DEFAULT 0,
  item_name     TEXT,

  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Prevents duplicate canonical requirements per user:
CREATE UNIQUE INDEX ON requirements(user_id, canonical_key)
  WHERE canonical_key IS NOT NULL;
```

Skill level requirements are **not** stored in this table. See [Skills Storage Rationale](#skills-storage-rationale).

---

### `goal_requirements` (join table — non-skill requirements)

```sql
CREATE TABLE goal_requirements (
  goal_id        UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  requirement_id UUID NOT NULL REFERENCES requirements(id) ON DELETE CASCADE,
  PRIMARY KEY (goal_id, requirement_id)
);

-- Covers the requirement_id lookup in GET /api/requirements (the PK is (goal_id, requirement_id)
-- and does not help when filtering by requirement_id alone):
CREATE INDEX ON goal_requirements(requirement_id);
```

A requirement can be linked to many goals. This join table is the mechanism for shared requirement deduplication. When `GET /api/requirements` is called, each requirement row is returned once with a `shared_by_goals` array containing all goals it is linked to.

---

### `goal_skill_requirements`

```sql
-- One threshold row per skill per goal.
-- Completion is NEVER stored here — always computed as user.skills[skill] >= level.
CREATE TABLE goal_skill_requirements (
  goal_id UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  skill   TEXT NOT NULL,  -- 'Agility', 'Slayer', etc.
  level   INT  NOT NULL,  -- the level threshold this goal requires
  PRIMARY KEY (goal_id, skill)  -- one threshold per skill per goal
);
```

---

## Skills Storage Rationale

Skill requirements are handled differently from all other requirement types because **skill levels are cumulative** — reaching level 72 implicitly satisfies any requirement for 70 or below. The two-table approach (current level on user, thresholds on goals) makes this correct by construction.

**Why not store skill requirements as rows in `requirements`?**

If "70 Agility" and "72 Agility" were stored as separate checkable rows, a user could manually check off "70 Agility" while sitting at level 68. That's wrong. Completion must derive from the user's actual level, not from a manual toggle.

**How it works instead:**

- `users.skills` holds the current level per skill as a JSONB map. This is the single source of truth.
- `goal_skill_requirements` holds the level threshold each goal needs.
- At query time: `satisfied = users.skills[skill] >= threshold.level`
- Updating a user's skill level (via Hiscores sync or manual entry) instantly refreshes the satisfied state of every threshold across all their goals — no extra writes required.

**Example:**

User has Agility 63. Two active goals:
- Morytania Hard needs Agility 70 → `satisfied: false`
- Fremennik Hard needs Agility 72 → `satisfied: false`

User trains to 72. Next API call:
- Morytania Hard needs Agility 70 → `satisfied: true` (72 >= 70)
- Fremennik Hard needs Agility 72 → `satisfied: true` (72 >= 72)

No rows were written to record completion. It just works.

---

## Deduplication Summary

| Requirement type | Storage | Completion |
|---|---|---|
| Skill level | `goal_skill_requirements` (threshold per goal) + `users.skills` (current level) | `current >= threshold` — automatic |
| Quest | `requirements` row, shared via `goal_requirements` | Manual checkbox |
| Kill count | `requirements` row with counter | `kill_current >= kill_target` |
| Item obtain | `requirements` row | Manual checkbox |
| Freeform | `requirements` row | Manual checkbox |

Non-skill requirements deduplicate via the `canonical_key` partial unique index. Skill requirements deduplicate by design — one ladder per skill across all active goals, many thresholds.

---

## Seed Data

All pre-seeded game data (Achievement Diaries, quests, skill requirements) is bundled as SQL migration files. There is no runtime scraping and no external data fetching on startup.

**Source of truth:** The [OSRS Wiki](https://oldschool.runescape.wiki). All seed data is manually authored from the wiki. If the game updates a diary requirement, update the SQL and ship a new migration.

### Coverage

| Dataset | Count | Migration | Notes |
|---|---|---|---|
| Achievement Diary goals | 48 | `002_seed_diaries.sql` | 12 regions × 4 tiers |
| Diary skill thresholds | ~120 rows | `002_seed_diaries.sql` | Varies by diary; many shared |
| Diary quest requirements | ~60 rows | `002_seed_diaries.sql` | Canonical keys prevent duplicates |
| Quests | ~200 | `003_seed_quests.sql` | Phase 2 — not in MVP |
| Skill names | 23 | Hardcoded in catalog handler | Fixed list; never changes |

### Seed Data Pattern

Pre-seeded goals have `user_id = NULL`. The `id` column is `UUID PRIMARY KEY` — seed files must use valid UUID literals, not text aliases. Generate stable IDs once with `uuidgen | tr '[:upper:]' '[:lower:]'` and never change them after a migration ships.

```sql
-- 002_seed_diaries.sql
--
-- Stable hardcoded UUIDs for pre-seeded content.
-- Generate once with: uuidgen | tr '[:upper:]' '[:lower:]'
-- Never change these after the migration is shipped.

INSERT INTO goals (id, user_id, name, description, category, diary_region, diary_tier, is_preseeded)
VALUES (
  'd4e5f6a7-b8c9-4d0e-1f2a-3b4c5d6e7f80',
  NULL,
  'Morytania Hard',
  'Complete all Hard tier tasks in Morytania.',
  'achievement_diary',
  'Morytania',
  'hard',
  TRUE
);

-- Skill thresholds
INSERT INTO goal_skill_requirements (goal_id, skill, level) VALUES
  ('d4e5f6a7-b8c9-4d0e-1f2a-3b4c5d6e7f80', 'Agility',      70),
  ('d4e5f6a7-b8c9-4d0e-1f2a-3b4c5d6e7f80', 'Slayer',        70),
  ('d4e5f6a7-b8c9-4d0e-1f2a-3b4c5d6e7f80', 'Construction',  50);

-- Non-skill requirement (shared across all Morytania tiers)
INSERT INTO requirements (id, user_id, label, type, canonical_key, is_preseeded, quest_name)
VALUES (
  'a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d',
  NULL,
  'Priest in Peril',
  'quest',
  'quest:Priest in Peril',
  TRUE,
  'Priest in Peril'
);

INSERT INTO goal_requirements (goal_id, requirement_id)
VALUES (
  'd4e5f6a7-b8c9-4d0e-1f2a-3b4c5d6e7f80',
  'a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d'
);
```

### Activating a Pre-Seeded Goal

When a user adds a catalog goal to their planner, the handler runs a single transaction that:

1. Copies the catalog `goal` row into a new user-owned row.
2. For each `goal_skill_requirement` on the catalog goal, inserts a corresponding row for the new goal.
3. For each `requirement` linked via `goal_requirements`, upserts the requirement for this user (using `ON CONFLICT (user_id, canonical_key) DO NOTHING`) and links it to the new goal.

This ensures the user owns their own copy of the goal and its requirements, which they can then modify (add notes, mark complete) without affecting other users.
