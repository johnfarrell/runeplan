# Route Reference

RunePlan has three kinds of routes:

- **Page routes** (`GET /planner`, `GET /browse`, etc.) — return full HTML pages with the base layout. Require a valid session; redirect to `/login` if not authenticated.
- **HTMX fragment routes** (`/htmx/*`) — return partial HTML for in-place DOM swapping. Used by all CRUD operations and partial page refreshes.
- **JSON API routes** (`/api/*`) — return `application/json`. Used for catalog data (public), user data export, and the health check.

**Error handling:**
- HTML routes: render the `Error` component with appropriate HTTP status.
- HTMX routes: same — return an HTML error fragment that HTMX can swap into the page.
- JSON routes: always return `{ "error": "human-readable description" }`.

**Authentication:**
- Unauthenticated requests to protected page routes → `302` redirect to `/login`.
- Unauthenticated HTMX requests (detected via `HX-Request: true` header) → `401` with `HX-Redirect: /login` response header.

---

## Page Routes

| Method | Path | Notes |
|---|---|---|
| `GET` | `/login` | Login form page |
| `GET` | `/register` | Registration form page |
| `GET` | `/` | Redirects to `/planner` |
| `GET` | `/planner` | 3-panel planner (auth required) |
| `GET` | `/browse` | Catalog browser (auth required) |
| `GET` | `/profile` | User profile + skill sync (auth required) |

---

## Auth Form Handlers

| Method | Path | Body | Response |
|---|---|---|---|
| `POST` | `/auth/register` | `email`, `password` (form) | `303` redirect to `/planner` on success; re-render form with error on failure |
| `POST` | `/auth/login` | `email`, `password` (form) | `303` redirect to `/planner` on success; re-render form with error on failure |
| `POST` | `/auth/logout` | — | `303` redirect to `/login` |
| `GET` | `/auth/discord` | — | `302` redirect to Discord |
| `GET` | `/auth/discord/callback` | — | `303` redirect to `/planner` on success |

---

## HTMX Fragment Routes

All `/htmx/*` routes require a valid session. They return HTML fragments (no base layout). Use these as `hx-get`, `hx-post`, `hx-patch`, or `hx-delete` targets in templates.

### Goals

| Method | Path | Body | Returns |
|---|---|---|---|
| `GET` | `/htmx/goals` | — | `<ul>` of all active goals (GoalSidebar fragment) |
| `POST` | `/htmx/goals` | `name`, `category`, `catalog_goal_id?` (form) | New `<li>` GoalItem fragment |
| `PATCH` | `/htmx/goals/{id}` | Partial goal fields (form) | Updated GoalItem fragment |
| `DELETE` | `/htmx/goals/{id}` | — | Empty response (HTMX deletes the element) |

**POST `/htmx/goals` — creating a goal**

Custom goal (form fields):
```
name=Learn the Inferno&category=custom
```

Activating a pre-seeded catalog goal:
```
catalog_goal_id=preset-morytania-hard
```

When `catalog_goal_id` is provided, the handler copies the catalog goal and all its requirements and skill thresholds into user-owned rows in a single transaction.

### Skill Thresholds

| Method | Path | Body | Returns |
|---|---|---|---|
| `GET` | `/htmx/skills` | — | Full SkillLadder list fragment |
| `POST` | `/htmx/goals/{id}/skills` | `skill`, `level` (form) | Updated SkillLadder list fragment |
| `DELETE` | `/htmx/goals/{id}/skills/{skill}` | — | Updated SkillLadder list fragment |

### Requirements

| Method | Path | Body | Returns |
|---|---|---|---|
| `GET` | `/htmx/requirements` | — | RequirementList fragment (deduplicated, with `shared_by_goals`) |
| `POST` | `/htmx/requirements` | See below | New RequirementRow fragment |
| `PATCH` | `/htmx/requirements/{id}` | `notes?`, `kill_current?`, `is_completed?` (form) | Updated RequirementRow fragment |
| `DELETE` | `/htmx/requirements/{id}` | — | Empty response |
| `POST` | `/htmx/goals/{id}/requirements/{req_id}` | — | Updated RequirementRow fragment (now shared) |
| `DELETE` | `/htmx/goals/{id}/requirements/{req_id}` | — | Updated RequirementRow fragment (unlinked) |

**POST `/htmx/requirements` — creating a requirement**

Provide fields appropriate to the `type`:

Quest: `label=Priest+in+Peril&type=quest&quest_name=Priest+in+Peril`

Kill count: `label=Kill+Zulrah+x100&type=kill_count&boss_name=Zulrah&kill_target=100`

Item obtain: `label=Obtain+Twisted+Bow&type=item_obtain&item_name=Twisted+Bow`

Freeform: `label=Research+Jad+phases&type=freeform`

### User / Profile

| Method | Path | Body | Returns |
|---|---|---|---|
| `GET` | `/htmx/user/me` | — | ProfileFragment (skill grid + account info) |
| `PATCH` | `/htmx/user/me` | Partial: `rsn?`, `skills?` (form / JSON) | Updated ProfileFragment |
| `POST` | `/htmx/user/sync` | — | Updated SkillGrid fragment, or `429` HTML error if synced within 5 seconds |

`PATCH /htmx/user/me` accepts skill levels as form fields: `skills[Agility]=70&skills[Slayer]=67`. The backend merges the provided skills into the existing map — it does not replace the entire `skills` object.

---

## JSON API Routes

### Catalog (public — no authentication required)

| Method | Path | Response |
|---|---|---|
| `GET` | `/api/catalog/diaries` | `200 CatalogGoal[]` — all 48 diary goals with thresholds |
| `GET` | `/api/catalog/quests` | `200 CatalogGoal[]` — all quests (Phase 2) |
| `GET` | `/api/catalog/skills` | `200 string[]` — ordered list of all 23 skill names |

**`CatalogGoal` shape:**
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

### User Export (authenticated)

| Method | Path | Response |
|---|---|---|
| `GET` | `/api/user/export` | `200` JSON blob — full dump of user's goals, requirements, skill levels |

### Health Check

```
GET /health
```

Returns `200 { "status": "ok" }`. No authentication required. Used by Docker Compose health checks.

---

## Data Shapes (for template context, not HTTP responses)

These are the Go structs that handlers build and pass to templates. They mirror the database schema and inform how templates render data.

### `Goal`
```go
type Goal struct {
    ID          string
    UserID      string
    Name        string
    Description string
    Category    string    // "achievement_diary" | "quest" | "skill_milestone" | "boss_kill" | "item_obtain" | "custom"
    DiaryRegion string
    DiaryTier   string    // "easy" | "medium" | "hard" | "elite"
    IsPreseeded bool
    IsCompleted bool
    SortOrder   int
    CreatedAt   time.Time
}
```

### `Requirement`
```go
type Requirement struct {
    ID           string
    Label        string
    Type         string    // "quest" | "kill_count" | "item_obtain" | "freeform"
    IsPreseeded  bool
    IsCompleted  bool
    Notes        string
    CanonicalKey string
    QuestName    string
    BossName     string
    KillTarget   int
    KillCurrent  int
    ItemName     string
    SharedByGoals []GoalSummary
}
```

### `SkillLadder`
```go
type SkillLadder struct {
    Skill        string
    CurrentLevel int
    Thresholds   []SkillThreshold
}

type SkillThreshold struct {
    Level     int
    Satisfied bool           // CurrentLevel >= Level
    Goals     []GoalSummary
}

type GoalSummary struct {
    ID   string
    Name string
}
```

### `User`
```go
type User struct {
    ID               string
    RSN              string
    Skills           map[string]int
    LastHiscoresSync *time.Time
}
```
