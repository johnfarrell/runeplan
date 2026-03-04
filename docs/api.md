# API Reference

All endpoints return JSON. All authenticated endpoints require a valid session cookie (`runeplan_session`). Unauthenticated requests to protected routes return `401 Unauthorized`.

Errors always return:
```json
{ "error": "human-readable description" }
```

---

## Authentication

| Method | Path | Auth | Body | Response |
|---|---|---|---|---|
| `POST` | `/api/auth/register` | — | `{ email, password }` | `201 { user }` |
| `POST` | `/api/auth/login` | — | `{ email, password }` | `200 { user }` |
| `POST` | `/api/auth/logout` | ✓ | — | `204` |
| `GET` | `/api/auth/discord` | — | — | `302` redirect to Discord |
| `GET` | `/api/auth/discord/callback` | — | — | `302` redirect on success |

Both `register` and `login` set a `runeplan_session` HTTP-only cookie on success. `logout` clears the cookie and deletes the session row from the database.

---

## User

| Method | Path | Auth | Body / Notes | Response |
|---|---|---|---|---|
| `GET` | `/api/user/me` | ✓ | — | `200 User` |
| `PATCH` | `/api/user/me` | ✓ | Partial: `{ rsn?, skills? }` | `200 User` |
| `POST` | `/api/user/sync` | ✓ | Fetches Hiscores, updates `skills` | `200 { skills }` or `429` |
| `GET` | `/api/user/export` | ✓ | Full data export | `200` JSON blob |

`POST /api/user/sync` returns `429 Too Many Requests` if called within 5 seconds of the previous sync (enforced via `last_hiscores_sync`).

`PATCH /api/user/me` accepts a partial body — only provided fields are updated. To manually set skill levels without syncing:

```json
PATCH /api/user/me
{
  "skills": { "Agility": 70, "Slayer": 67 }
}
```

The backend merges the provided skills into the existing map — it does not replace the entire `skills` object.

### `User` response shape

```json
{
  "id": "uuid",
  "rsn": "Zezima",
  "skills": {
    "Attack": 65,
    "Agility": 63,
    "Slayer": 67
  },
  "last_hiscores_sync": "2024-03-01T12:00:00Z"
}
```

---

## Goals

| Method | Path | Auth | Body / Notes | Response |
|---|---|---|---|---|
| `GET` | `/api/goals` | ✓ | — | `200 Goal[]` |
| `POST` | `/api/goals` | ✓ | See below | `201 Goal` |
| `PATCH` | `/api/goals/:id` | ✓ | Partial `Goal` fields | `200 Goal` |
| `DELETE` | `/api/goals/:id` | ✓ | — | `204` |

### `POST /api/goals` — Creating a goal

**Custom goal:**
```json
{
  "name": "Learn the Inferno",
  "description": "Study and prepare for first Inferno attempt.",
  "category": "custom"
}
```

**Activating a pre-seeded catalog goal:**
```json
{
  "catalog_goal_id": "preset-morytania-hard"
}
```

When `catalog_goal_id` is provided, the handler copies the catalog goal and all its requirements and skill thresholds into user-owned rows in a single transaction. The response is the newly created user-owned `Goal`.

### `Goal` response shape

```json
{
  "id": "uuid",
  "user_id": "uuid",
  "name": "Morytania Hard",
  "description": "Complete all Hard tier tasks in Morytania.",
  "category": "achievement_diary",
  "diary_region": "Morytania",
  "diary_tier": "hard",
  "is_preseeded": true,
  "is_completed": false,
  "sort_order": 0,
  "created_at": "2024-03-01T12:00:00Z"
}
```

Valid `category` values: `achievement_diary`, `quest`, `skill_milestone`, `boss_kill`, `item_obtain`, `custom`.

Valid `diary_tier` values: `easy`, `medium`, `hard`, `elite`.

---

## Skills

| Method | Path | Auth | Body / Notes | Response |
|---|---|---|---|---|
| `GET` | `/api/skills` | ✓ | Aggregates across all active goals | `200 SkillLadder[]` |
| `POST` | `/api/goals/:id/skills` | ✓ | `{ skill, level }` | `201` |
| `DELETE` | `/api/goals/:id/skills/:skill` | ✓ | — | `204` |

### `GET /api/skills` — Skill Ladder aggregation

This is the primary query for rendering skill requirements in the UI. It returns one `SkillLadder` per skill that appears in any active (non-completed) goal, with all thresholds from all goals aggregated and sorted ascending, and `satisfied` computed server-side against the user's current level.

The frontend renders this response directly — **no client-side computation is required**.

```json
[
  {
    "skill": "Agility",
    "current_level": 63,
    "notes": "",
    "thresholds": [
      {
        "level": 70,
        "satisfied": false,
        "goals": [
          { "id": "uuid", "name": "Morytania Hard" }
        ]
      },
      {
        "level": 72,
        "satisfied": false,
        "goals": [
          { "id": "uuid", "name": "Fremennik Hard" }
        ]
      }
    ]
  },
  {
    "skill": "Slayer",
    "current_level": 67,
    "notes": "",
    "thresholds": [
      {
        "level": 70,
        "satisfied": false,
        "goals": [
          { "id": "uuid", "name": "Morytania Hard" }
        ]
      }
    ]
  }
]
```

When the user reaches Agility 72, both thresholds in the Agility ladder flip to `satisfied: true` on the next call to this endpoint. No write is required to record skill completion.

---

## Requirements

Requirements in this API are **non-skill requirements only** (quest, kill count, item obtain, freeform). Skill level requirements are managed via the Skills endpoints above.

| Method | Path | Auth | Body / Notes | Response |
|---|---|---|---|---|
| `GET` | `/api/requirements` | ✓ | Deduplicated across all active goals | `200 Requirement[]` |
| `POST` | `/api/requirements` | ✓ | See below | `201 Requirement` |
| `PATCH` | `/api/requirements/:id` | ✓ | Partial: `{ notes?, kill_current?, is_completed? }` | `200 Requirement` |
| `DELETE` | `/api/requirements/:id` | ✓ | Also removes all `goal_requirements` rows | `204` |
| `POST` | `/api/goals/:id/requirements/:req_id` | ✓ | Links an existing requirement to a goal | `201` |
| `DELETE` | `/api/goals/:id/requirements/:req_id` | ✓ | Unlinks (does not delete the requirement) | `204` |

### `POST /api/requirements` — Creating a requirement

Provide fields appropriate to the `type`. All unrelated fields are ignored.

**Quest:**
```json
{ "label": "Priest in Peril", "type": "quest", "quest_name": "Priest in Peril" }
```

**Kill count:**
```json
{ "label": "Kill Zulrah x100", "type": "kill_count", "boss_name": "Zulrah", "kill_target": 100 }
```

**Item obtain:**
```json
{ "label": "Obtain Twisted Bow", "type": "item_obtain", "item_name": "Twisted Bow" }
```

**Freeform:**
```json
{ "label": "Research Jad phases", "type": "freeform" }
```

For pre-seeded requirements, the backend sets `canonical_key` automatically. For user-created requirements, `canonical_key` is null and deduplication does not apply.

### `Requirement` response shape

```json
{
  "id": "uuid",
  "label": "Priest in Peril",
  "type": "quest",
  "is_preseeded": true,
  "is_completed": false,
  "notes": "",
  "canonical_key": "quest:Priest in Peril",
  "quest_name": "Priest in Peril",
  "shared_by_goals": [
    { "id": "uuid", "name": "Morytania Easy" },
    { "id": "uuid", "name": "Morytania Medium" },
    { "id": "uuid", "name": "Morytania Hard" }
  ]
}
```

`shared_by_goals` is always present and may be empty. When it contains more than one entry, the UI shows a shared badge.

---

## Catalog

Catalog endpoints return pre-seeded read-only game data. No authentication required (catalog data is not user-specific).

| Method | Path | Response | Notes |
|---|---|---|---|
| `GET` | `/api/catalog/diaries` | `200 CatalogGoal[]` | All 48 diary goals with thresholds |
| `GET` | `/api/catalog/quests` | `200 CatalogGoal[]` | All quests (Phase 2) |
| `GET` | `/api/catalog/skills` | `200 string[]` | Ordered list of all 23 skill names |

### `CatalogGoal` shape

```json
{
  "id": "preset-morytania-hard",
  "name": "Morytania Hard",
  "description": "Complete all Hard tier tasks in Morytania.",
  "category": "achievement_diary",
  "diary_region": "Morytania",
  "diary_tier": "hard",
  "skill_thresholds": [
    { "skill": "Agility", "level": 70 },
    { "skill": "Slayer",  "level": 70 }
  ],
  "requirements": [
    { "label": "Priest in Peril", "type": "quest", "canonical_key": "quest:Priest in Peril" }
  ]
}
```

---

## Health Check

```
GET /health
```

Returns `200 { "status": "ok" }`. No authentication required. Used by Docker Compose health checks and any external uptime monitors.
