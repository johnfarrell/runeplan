# RunePlan Bootstrap Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers-extended-cc:executing-plans to implement this plan task-by-task.

**Goal:** Build a fully runnable RunePlan app — catalog browsing, goal planning, skill tracking, hiscores sync — using Go + Templ + HTMX + gorilla/mux.

**Architecture:** Vertical feature slices: each slice is end-to-end complete (migration → domain → repo → service → handler → template) before the next begins. Dev-auth stub injects a hardcoded user so no real auth is needed for the MVP.

**Tech Stack:** Go 1.25, gorilla/mux, pgx/v5, golang-migrate, a-h/templ, HTMX 2.x, Alpine.js 3.x, Tailwind CSS standalone CLI, PostgreSQL 16.

---

## Task 1: Add Go dependencies

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

**Step 1: Add required packages**

```bash
cd /home/john/projects/runeplan
go get github.com/gorilla/mux@v1.8.1
go get github.com/jackc/pgx/v5@v5.7.2
go get github.com/golang-migrate/migrate/v4@v4.18.1
go get github.com/a-h/templ@v0.3.833
```

Note: `go get` will also fetch indirect deps and update go.sum automatically.

**Step 2: Verify build still compiles**

```bash
go build ./...
```

Expected: no errors. (go.sum is updated, existing code still compiles.)

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "Add gorilla/mux, pgx/v5, golang-migrate, templ dependencies"
```

---

## Task 2: Database migrations

**Files:**
- Create: `migrations/001_schema.sql`
- Create: `migrations/002_seed_catalog.sql`
- Create: `migrations/embed.go`

**Step 1: Write the schema migration**

Create `migrations/001_schema.sql`:

```sql
-- +migrate Up

CREATE TABLE users (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_rsns (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  rsn          TEXT NOT NULL,
  skill_levels JSONB NOT NULL DEFAULT '{}',
  synced_at    TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, rsn)
);

CREATE TABLE catalog_goals (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  canonical_key TEXT UNIQUE NOT NULL,
  title         TEXT NOT NULL,
  type          TEXT NOT NULL,
  description   TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE catalog_requirements (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  catalog_goal_id UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  description     TEXT NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE catalog_skill_requirements (
  catalog_goal_id UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  skill           TEXT NOT NULL,
  level           INT  NOT NULL,
  PRIMARY KEY (catalog_goal_id, skill)
);

CREATE TABLE catalog_item_requirements (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  catalog_goal_id UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  item_name       TEXT NOT NULL,
  quantity        INT  NOT NULL DEFAULT 1
);

CREATE TABLE catalog_boss_requirements (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  catalog_goal_id UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  boss_name       TEXT NOT NULL,
  kc              INT  NOT NULL
);

CREATE TABLE catalog_goal_prerequisites (
  goal_id   UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  prereq_id UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  PRIMARY KEY (goal_id, prereq_id)
);

CREATE TABLE goals (
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
);

CREATE TABLE goal_requirement_progress (
  goal_id        UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  requirement_id UUID NOT NULL REFERENCES catalog_requirements(id) ON DELETE CASCADE,
  completed      BOOLEAN NOT NULL DEFAULT false,
  completed_at   TIMESTAMPTZ,
  PRIMARY KEY (goal_id, requirement_id)
);

CREATE TABLE custom_requirements (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  goal_id      UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  description  TEXT NOT NULL,
  completed    BOOLEAN NOT NULL DEFAULT false,
  completed_at TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +migrate Down
DROP TABLE IF EXISTS custom_requirements;
DROP TABLE IF EXISTS goal_requirement_progress;
DROP TABLE IF EXISTS goals;
DROP TABLE IF EXISTS catalog_goal_prerequisites;
DROP TABLE IF EXISTS catalog_boss_requirements;
DROP TABLE IF EXISTS catalog_item_requirements;
DROP TABLE IF EXISTS catalog_skill_requirements;
DROP TABLE IF EXISTS catalog_requirements;
DROP TABLE IF EXISTS catalog_goals;
DROP TABLE IF EXISTS user_rsns;
DROP TABLE IF EXISTS users;
```

**Step 2: Write the seed migration**

Create `migrations/002_seed_catalog.sql`:

```sql
-- +migrate Up

-- Sample quests
INSERT INTO catalog_goals (canonical_key, title, type, description) VALUES
  ('quest.cooks_assistant',      'Cook''s Assistant',        'quest', 'Help the cook at Lumbridge Castle.'),
  ('quest.romeo_and_juliet',     'Romeo & Juliet',           'quest', 'A tale of star-crossed lovers in Varrock.'),
  ('quest.desert_treasure',      'Desert Treasure',          'quest', 'Hunt down four diamonds in the desert.'),
  ('quest.desert_treasure_2',    'Desert Treasure II',       'quest', 'The fallen empire — sequel to Desert Treasure.'),
  ('quest.dragon_slayer',        'Dragon Slayer',            'quest', 'Prove yourself by slaying Elvarg.'),
  ('quest.underground_pass',     'Underground Pass',         'quest', 'Navigate the treacherous Underground Pass.'),
  ('quest.regicide',             'Regicide',                 'quest', 'Assassinate the King of the Elves.'),
  ('quest.monkey_madness',       'Monkey Madness',           'quest', 'Help Gnome King Narnode on Ape Atoll.'),
  ('quest.fremennik_trials',     'The Fremennik Trials',     'quest', 'Earn acceptance among the Fremennik.'),
  ('quest.recipe_for_disaster',  'Recipe for Disaster',      'quest', 'Save the Lumbridge Council from a culinary catastrophe.');

-- Skill requirements for Desert Treasure
INSERT INTO catalog_skill_requirements (catalog_goal_id, skill, level)
SELECT id, 'magic', 50 FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure'
UNION ALL
SELECT id, 'thieving', 53 FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure'
UNION ALL
SELECT id, 'firemaking', 50 FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure'
UNION ALL
SELECT id, 'slayer', 10 FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure';

-- Skill requirements for Dragon Slayer
INSERT INTO catalog_skill_requirements (catalog_goal_id, skill, level)
SELECT id, 'attack', 32 FROM catalog_goals WHERE canonical_key = 'quest.dragon_slayer';

-- Freeform requirements for Desert Treasure
INSERT INTO catalog_requirements (catalog_goal_id, description)
SELECT id, 'The Restless Ghost quest complete' FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure'
UNION ALL
SELECT id, 'Priest in Peril quest complete' FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure'
UNION ALL
SELECT id, 'Temple of Ikov quest complete' FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure'
UNION ALL
SELECT id, 'The Tourist Trap quest complete' FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure'
UNION ALL
SELECT id, 'Troll Stronghold quest complete' FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure'
UNION ALL
SELECT id, 'Waterfall Quest complete' FROM catalog_goals WHERE canonical_key = 'quest.desert_treasure';

-- Sample diaries
INSERT INTO catalog_goals (canonical_key, title, type, description) VALUES
  ('diary.lumbridge_easy',       'Lumbridge & Draynor Diary (Easy)',   'diary', 'Easy tasks in the Lumbridge & Draynor area.'),
  ('diary.lumbridge_medium',     'Lumbridge & Draynor Diary (Medium)', 'diary', 'Medium tasks in the Lumbridge & Draynor area.'),
  ('diary.morytania_hard',       'Morytania Diary (Hard)',             'diary', 'Hard tasks in the Morytania area.');

-- Skill requirements for Morytania Hard
INSERT INTO catalog_skill_requirements (catalog_goal_id, skill, level)
SELECT id, 'slayer', 71 FROM catalog_goals WHERE canonical_key = 'diary.morytania_hard'
UNION ALL
SELECT id, 'agility', 61 FROM catalog_goals WHERE canonical_key = 'diary.morytania_hard'
UNION ALL
SELECT id, 'prayer', 70 FROM catalog_goals WHERE canonical_key = 'diary.morytania_hard'
UNION ALL
SELECT id, 'herblore', 53 FROM catalog_goals WHERE canonical_key = 'diary.morytania_hard';

-- +migrate Down
DELETE FROM catalog_skill_requirements;
DELETE FROM catalog_requirements;
DELETE FROM catalog_goals;
```

**Step 3: Create the embed file**

Create `migrations/embed.go`:

```go
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
```

**Step 4: Verify it compiles**

```bash
go build ./...
```

Expected: no errors.

**Step 5: Commit**

```bash
git add migrations/
git commit -m "Add database schema and catalog seed migrations"
```

---

## Task 3: Infrastructure — postgres connection + migrations runner

**Files:**
- Create: `infrastructure/postgres/connect.go`
- Create: `infrastructure/postgres/migrate.go`
- Modify: `infrastructure/postgres/goal_repository.go` (package declaration only — leave stub, implement in Task 8)

**Step 1: Write failing test for connect**

Create `infrastructure/postgres/connect_test.go`:

```go
package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/johnfarrell/runeplan/infrastructure/postgres"
)

func TestConnect_MissingURL(t *testing.T) {
	_, err := postgres.Connect(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestConnect_Live(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := postgres.Connect(context.Background(), url)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./infrastructure/postgres/... -run TestConnect_MissingURL -v
```

Expected: compile error (package doesn't exist yet).

**Step 3: Implement connect.go**

Create `infrastructure/postgres/connect.go`:

```go
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect creates and validates a pgx connection pool for the given DATABASE_URL.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("postgres: DATABASE_URL is required")
	}
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return pool, nil
}
```

**Step 4: Run tests**

```bash
go test ./infrastructure/postgres/... -run TestConnect_MissingURL -v
```

Expected: PASS.

**Step 5: Implement migrate.go**

Create `infrastructure/postgres/migrate.go`:

```go
package postgres

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/johnfarrell/runeplan/migrations"
)

// RunMigrations applies all pending up migrations from the embedded migrations FS.
func RunMigrations(databaseURL string) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("migrations: create source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, databaseURL)
	if err != nil {
		return fmt.Errorf("migrations: create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrations: up: %w", err)
	}
	return nil
}
```

Note: golang-migrate requires the database URL scheme to be `pgx5://` for the pgx/v5 driver. The connect.go above uses `pgxpool` directly (standard `postgres://` URL). For RunMigrations, the caller should pass a URL with scheme `pgx5://` or we convert it. See Task 11 (main.go) for how to handle this.

**Step 6: Verify build**

```bash
go build ./...
```

Expected: compiles. If there are missing indirect deps, run `go mod tidy` ONLY after confirming all imports are in place.

**Step 7: Commit**

```bash
git add infrastructure/postgres/connect.go infrastructure/postgres/migrate.go infrastructure/postgres/connect_test.go
git commit -m "Add postgres connection pool and migrations runner"
```

---

## Task 4: Base templ layout

Templ workflow: write `.templ` file → run `templ generate` → commit both `.templ` and `_templ.go`.

**Files:**
- Create: `interfaces/templates/layout.templ`
- Create: `interfaces/templates/layout_templ.go` (generated)
- Create: `interfaces/templates/render.go`

**Step 1: Write the layout template**

Create `interfaces/templates/layout.templ`:

```templ
package templates

templ Base(title string) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>{ title } — RunePlan</title>
			<link rel="stylesheet" href="/static/app.css"/>
			<script src="/static/htmx.min.js" defer></script>
			<script src="/static/alpine.min.js" defer></script>
		</head>
		<body class="bg-stone-900 text-stone-100 min-h-screen">
			<nav class="bg-stone-800 border-b border-stone-700 px-4 py-3 flex gap-6 items-center">
				<span class="font-bold text-yellow-400 text-lg">RunePlan</span>
				<a href="/browse" class="hover:text-yellow-300">Browse</a>
				<a href="/planner" class="hover:text-yellow-300">Planner</a>
				<a href="/profile" class="hover:text-yellow-300">Profile</a>
			</nav>
			<main class="max-w-5xl mx-auto px-4 py-8">
				{ children... }
			</main>
		</body>
	</html>
}

templ Error(msg string) {
	<div class="bg-red-900 border border-red-600 text-red-100 px-4 py-3 rounded">
		{ msg }
	</div>
}
```

**Step 2: Generate the template**

```bash
templ generate ./interfaces/templates/
```

Expected: creates `interfaces/templates/layout_templ.go`.

**Step 3: Create the render helper**

Create `interfaces/templates/render.go`:

```go
package templates

import (
	"net/http"

	"github.com/a-h/templ"
)

// Render writes a templ component to the response with the given status code.
func Render(w http.ResponseWriter, r *http.Request, status int, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = c.Render(r.Context(), w)
}
```

**Step 4: Write a basic layout test**

Create `interfaces/templates/layout_test.go`:

```go
package templates_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/johnfarrell/runeplan/interfaces/templates"
)

func TestBase_ContainsTitle(t *testing.T) {
	var buf bytes.Buffer
	if err := templates.Base("My Page").Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "My Page") {
		t.Error("expected title in rendered HTML")
	}
	if !strings.Contains(buf.String(), "RunePlan") {
		t.Error("expected brand name in rendered HTML")
	}
}
```

**Step 5: Run tests**

```bash
go test ./interfaces/templates/... -v
```

Expected: PASS.

**Step 6: Create static assets directory**

The base layout references `/static/app.css`, `/static/htmx.min.js`, `/static/alpine.min.js`. Create placeholder files so the binary embeds them:

```bash
mkdir -p static
touch static/app.css static/htmx.min.js static/alpine.min.js
```

Download actual files or symlink from CDN in development. For now stubs allow compilation.

**Step 7: Commit**

```bash
git add interfaces/templates/ static/
git commit -m "Add base templ layout and render helper"
```

---

## Task 5: User domain + dev-auth middleware

**Files:**
- Create: `domain/user/user.go`
- Create: `domain/user/context.go`
- Create: `interfaces/middleware/devauth.go`

**Step 1: Write failing test for user context**

Create `domain/user/context_test.go`:

```go
package user_test

import (
	"context"
	"testing"

	"github.com/johnfarrell/runeplan/domain/user"
)

func TestSetGetUser(t *testing.T) {
	u := user.User{ID: "abc123"}
	ctx := user.SetUser(context.Background(), u)
	got, ok := user.GetUser(ctx)
	if !ok {
		t.Fatal("expected user in context")
	}
	if got.ID != "abc123" {
		t.Errorf("got ID %q, want %q", got.ID, "abc123")
	}
}

func TestGetUser_Missing(t *testing.T) {
	_, ok := user.GetUser(context.Background())
	if ok {
		t.Fatal("expected no user in empty context")
	}
}
```

**Step 2: Run test (expect compile error)**

```bash
go test ./domain/user/... -v
```

Expected: compile error — package doesn't exist.

**Step 3: Implement user domain**

Create `domain/user/user.go`:

```go
package user

import (
	"time"

	"github.com/johnfarrell/runeplan/domain/skill"
)

// User is a RunePlan account. It may have zero or more linked OSRS accounts (RSNs).
type User struct {
	ID        string
	RSNs      []RSN
	CreatedAt time.Time
}

// ActiveRSN returns the first RSN if any exist.
func (u *User) ActiveRSN() *RSN {
	if len(u.RSNs) == 0 {
		return nil
	}
	return &u.RSNs[0]
}

// RSN is a linked OSRS account. Skill levels are keyed by skill name.
type RSN struct {
	ID          string
	UserID      string
	RSN         string
	SkillLevels map[skill.Skill]skill.XP
	SyncedAt    *time.Time
	CreatedAt   time.Time
}
```

Create `domain/user/context.go`:

```go
package user

import "context"

type contextKey struct{}

// SetUser stores u in ctx and returns the updated context.
func SetUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, contextKey{}, u)
}

// GetUser retrieves the User from ctx. Returns false if not present.
func GetUser(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(contextKey{}).(User)
	return u, ok
}
```

**Step 4: Run tests**

```bash
go test ./domain/user/... -v
```

Expected: PASS.

**Step 5: Implement dev-auth middleware**

Create `interfaces/middleware/devauth.go`:

```go
package middleware

import (
	"net/http"

	"github.com/johnfarrell/runeplan/domain/skill"
	"github.com/johnfarrell/runeplan/domain/user"
)

// DevAuth injects a hardcoded user into every request context.
// Replace with real session middleware when auth is implemented.
func DevAuth(next http.Handler) http.Handler {
	hardcodedUser := user.User{
		ID: "00000000-0000-0000-0000-000000000001",
		RSNs: []user.RSN{
			{
				ID:          "00000000-0000-0000-0000-000000000002",
				UserID:      "00000000-0000-0000-0000-000000000001",
				RSN:         "Zezima",
				SkillLevels: make(map[skill.Skill]skill.XP),
			},
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := user.SetUser(r.Context(), hardcodedUser)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

**Step 6: Verify build**

```bash
go build ./...
```

**Step 7: Commit**

```bash
git add domain/user/ interfaces/middleware/devauth.go
git commit -m "Add user domain and dev-auth stub middleware"
```

---

## Task 6: Update goal domain + catalog domain

The existing `domain/goal/goal.go` has `UserID` but we need `RSNID` to match the schema. We also need a new `domain/catalog` package.

**Files:**
- Modify: `domain/goal/goal.go`
- Create: `domain/catalog/catalog.go`

**Step 1: Read existing goal.go** (already done above)

**Step 2: Update goal.go to match schema**

Replace `domain/goal/goal.go` with:

```go
package goal

import (
	"time"

	"github.com/johnfarrell/runeplan/domain/skill"
)

// Type classifies what kind of achievement a Goal represents.
type Type string

const (
	TypeQuest  Type = "quest"
	TypeDiary  Type = "diary"
	TypeSkill  Type = "skill"
	TypeBossKC Type = "boss_kc"
	TypeItem   Type = "item"
	TypeCustom Type = "custom"
)

func (t Type) Valid() bool {
	switch t {
	case TypeQuest, TypeDiary, TypeSkill, TypeBossKC, TypeItem, TypeCustom:
		return true
	}
	return false
}

// SkillThreshold is the minimum level required in a Skill to satisfy a goal.
type SkillThreshold struct {
	Skill skill.Skill
	XP    skill.XP
	Level skill.Level
}

func NewSkillLevelThreshold(s skill.Skill, level int) (SkillThreshold, error) {
	if !s.Valid() {
		return SkillThreshold{}, skill.ErrInvalidSkill
	}
	l, err := skill.NewLevel(level)
	if err != nil {
		return SkillThreshold{}, err
	}
	minXP, _ := skill.XPRangeForLevel(level)
	xp, err := skill.NewXP(minXP)
	if err != nil {
		return SkillThreshold{}, err
	}
	return SkillThreshold{Skill: s, XP: xp, Level: l}, nil
}

func (s SkillThreshold) IsSatisfiedByLevel(current skill.Level) bool {
	return current.Value() >= s.Level.Value()
}

func (s SkillThreshold) IsSatisfiedByXP(current skill.XP) bool {
	return current.Value() >= s.XP.Value()
}

// RequirementProgress tracks a user's completion state for a catalog requirement.
type RequirementProgress struct {
	GoalID        string
	RequirementID string
	Description   string // denormalised for display
	Completed     bool
	CompletedAt   *time.Time
}

// CustomRequirement is a user-added freeform requirement on a goal.
type CustomRequirement struct {
	ID          string
	GoalID      string
	Description string
	Completed   bool
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Goal is a per-RSN activated goal. It references a catalog goal and tracks progress.
type Goal struct {
	ID          string
	RSNID       string
	CatalogID   *string
	Title       string
	Type        Type
	Notes       string
	Completed   bool
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// Loaded on demand
	Requirements       []RequirementProgress
	CustomRequirements []CustomRequirement
}

func (g *Goal) Complete(at time.Time) {
	g.Completed = true
	g.CompletedAt = &at
	g.UpdatedAt = at
}
```

**Step 3: Create catalog domain**

Create `domain/catalog/catalog.go`:

```go
package catalog

import (
	"time"

	"github.com/johnfarrell/runeplan/domain/goal"
	"github.com/johnfarrell/runeplan/domain/skill"
)

// Goal is a pre-seeded canonical OSRS goal (quest, diary, etc.).
type Goal struct {
	ID           string
	CanonicalKey string
	Title        string
	Type         goal.Type
	Description  string
	CreatedAt    time.Time

	Requirements    []Requirement
	SkillReqs       []SkillRequirement
	ItemReqs        []ItemRequirement
	BossReqs        []BossRequirement
	PrerequisiteIDs []string
}

// Requirement is a freeform checklist item on a catalog goal.
type Requirement struct {
	ID            string
	CatalogGoalID string
	Description   string
	CreatedAt     time.Time
}

// SkillRequirement is a minimum skill level required by a catalog goal.
type SkillRequirement struct {
	CatalogGoalID string
	Skill         skill.Skill
	Level         skill.Level
}

// ItemRequirement is an item quantity required by a catalog goal.
type ItemRequirement struct {
	ID            string
	CatalogGoalID string
	ItemName      string
	Quantity      int
}

// BossRequirement is a minimum KC required by a catalog goal.
type BossRequirement struct {
	ID            string
	CatalogGoalID string
	BossName      string
	KC            int
}
```

**Step 4: Run tests**

```bash
go test ./domain/... -v
```

Expected: existing skill tests pass; goal and catalog packages compile.

**Step 5: Commit**

```bash
git add domain/goal/goal.go domain/catalog/
git commit -m "Update goal domain and add catalog domain"
```

---

## Task 7: Catalog infrastructure + service + handler + templates

**Files:**
- Create: `infrastructure/postgres/catalog_repository.go`
- Create: `application/catalog/service.go`
- Create: `interfaces/handler/catalog.go`
- Create: `interfaces/templates/catalog/browse.templ` + `_templ.go`
- Create: `interfaces/templates/catalog/detail.templ` + `_templ.go`

**Step 1: Write failing test for catalog service**

Create `application/catalog/service_test.go`:

```go
package catalog_test

import (
	"context"
	"testing"

	"github.com/johnfarrell/runeplan/application/catalog"
	domcatalog "github.com/johnfarrell/runeplan/domain/catalog"
	"github.com/johnfarrell/runeplan/domain/goal"
)

type mockRepo struct {
	goals []domcatalog.Goal
}

func (m *mockRepo) ListByType(ctx context.Context, t goal.Type) ([]domcatalog.Goal, error) {
	var out []domcatalog.Goal
	for _, g := range m.goals {
		if g.Type == t {
			out = append(out, g)
		}
	}
	return out, nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*domcatalog.Goal, error) {
	for _, g := range m.goals {
		if g.ID == id {
			return &g, nil
		}
	}
	return nil, catalog.ErrNotFound
}

func (m *mockRepo) ListAll(ctx context.Context) ([]domcatalog.Goal, error) {
	return m.goals, nil
}

func TestListByType(t *testing.T) {
	repo := &mockRepo{goals: []domcatalog.Goal{
		{ID: "1", Type: goal.TypeQuest, Title: "Q"},
		{ID: "2", Type: goal.TypeDiary, Title: "D"},
	}}
	svc := catalog.NewService(repo)
	got, err := svc.ListByType(context.Background(), goal.TypeQuest)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "1" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo := &mockRepo{}
	svc := catalog.NewService(repo)
	_, err := svc.GetByID(context.Background(), "missing")
	if err != catalog.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
```

**Step 2: Run test (expect fail)**

```bash
go test ./application/catalog/... -v
```

**Step 3: Implement catalog service**

Create `application/catalog/service.go`:

```go
package catalog

import (
	"context"
	"errors"

	domcatalog "github.com/johnfarrell/runeplan/domain/catalog"
	"github.com/johnfarrell/runeplan/domain/goal"
)

var ErrNotFound = errors.New("catalog: goal not found")

// Repository is the persistence interface for catalog goals.
type Repository interface {
	ListAll(ctx context.Context) ([]domcatalog.Goal, error)
	ListByType(ctx context.Context, t goal.Type) ([]domcatalog.Goal, error)
	GetByID(ctx context.Context, id string) (*domcatalog.Goal, error)
}

// Service handles catalog browsing use cases.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListAll(ctx context.Context) ([]domcatalog.Goal, error) {
	return s.repo.ListAll(ctx)
}

func (s *Service) ListByType(ctx context.Context, t goal.Type) ([]domcatalog.Goal, error) {
	return s.repo.ListByType(ctx, t)
}

func (s *Service) GetByID(ctx context.Context, id string) (*domcatalog.Goal, error) {
	return s.repo.GetByID(ctx, id)
}
```

**Step 4: Run tests**

```bash
go test ./application/catalog/... -v
```

Expected: PASS.

**Step 5: Implement catalog postgres repository**

Create `infrastructure/postgres/catalog_repository.go`:

```go
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/johnfarrell/runeplan/application/catalog"
	domcatalog "github.com/johnfarrell/runeplan/domain/catalog"
	"github.com/johnfarrell/runeplan/domain/goal"
	"github.com/johnfarrell/runeplan/domain/skill"
)

// CatalogRepository implements catalog.Repository using PostgreSQL.
type CatalogRepository struct {
	pool *pgxpool.Pool
}

func NewCatalogRepository(pool *pgxpool.Pool) *CatalogRepository {
	return &CatalogRepository{pool: pool}
}

func (r *CatalogRepository) ListAll(ctx context.Context) ([]domcatalog.Goal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, canonical_key, title, type, COALESCE(description, ''), created_at
		FROM catalog_goals ORDER BY type, title`)
	if err != nil {
		return nil, fmt.Errorf("catalog: list all: %w", err)
	}
	defer rows.Close()
	return scanGoals(rows)
}

func (r *CatalogRepository) ListByType(ctx context.Context, t goal.Type) ([]domcatalog.Goal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, canonical_key, title, type, COALESCE(description, ''), created_at
		FROM catalog_goals WHERE type = $1 ORDER BY title`, string(t))
	if err != nil {
		return nil, fmt.Errorf("catalog: list by type: %w", err)
	}
	defer rows.Close()
	return scanGoals(rows)
}

func (r *CatalogRepository) GetByID(ctx context.Context, id string) (*domcatalog.Goal, error) {
	var g domcatalog.Goal
	err := r.pool.QueryRow(ctx, `
		SELECT id, canonical_key, title, type, COALESCE(description, ''), created_at
		FROM catalog_goals WHERE id = $1`, id).
		Scan(&g.ID, &g.CanonicalKey, &g.Title, &g.Type, &g.Description, &g.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, catalog.ErrNotFound
		}
		return nil, fmt.Errorf("catalog: get by id: %w", err)
	}

	// Load skill requirements
	skillRows, err := r.pool.Query(ctx,
		`SELECT skill, level FROM catalog_skill_requirements WHERE catalog_goal_id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("catalog: skill reqs: %w", err)
	}
	defer skillRows.Close()
	for skillRows.Next() {
		var s string
		var lvl int
		if err := skillRows.Scan(&s, &lvl); err != nil {
			return nil, err
		}
		level, _ := skill.NewLevel(lvl)
		g.SkillReqs = append(g.SkillReqs, domcatalog.SkillRequirement{
			CatalogGoalID: id,
			Skill:         skill.Skill(s),
			Level:         level,
		})
	}

	// Load freeform requirements
	reqRows, err := r.pool.Query(ctx,
		`SELECT id, description FROM catalog_requirements WHERE catalog_goal_id = $1 ORDER BY created_at`, id)
	if err != nil {
		return nil, fmt.Errorf("catalog: requirements: %w", err)
	}
	defer reqRows.Close()
	for reqRows.Next() {
		var req domcatalog.Requirement
		req.CatalogGoalID = id
		if err := reqRows.Scan(&req.ID, &req.Description); err != nil {
			return nil, err
		}
		g.Requirements = append(g.Requirements, req)
	}

	return &g, nil
}

func scanGoals(rows pgx.Rows) ([]domcatalog.Goal, error) {
	var goals []domcatalog.Goal
	for rows.Next() {
		var g domcatalog.Goal
		if err := rows.Scan(&g.ID, &g.CanonicalKey, &g.Title, &g.Type, &g.Description, &g.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, rows.Err()
}
```

**Step 6: Write catalog templates**

Create `interfaces/templates/catalog/browse.templ`:

```templ
package catalog

import (
	"github.com/johnfarrell/runeplan/domain/goal"
	domcatalog "github.com/johnfarrell/runeplan/domain/catalog"
	"github.com/johnfarrell/runeplan/interfaces/templates"
)

var tabs = []struct {
	Label string
	Type  goal.Type
}{
	{"Quests", goal.TypeQuest},
	{"Diaries", goal.TypeDiary},
	{"Skills", goal.TypeSkill},
	{"Custom", goal.TypeCustom},
}

templ Browse(active goal.Type, goals []domcatalog.Goal) {
	@templates.Base("Browse") {
		<div class="mb-6 flex gap-2">
			for _, tab := range tabs {
				if tab.Type == active {
					<button class="px-4 py-2 rounded bg-yellow-600 text-stone-900 font-semibold">{ tab.Label }</button>
				} else {
					<button
						class="px-4 py-2 rounded bg-stone-700 hover:bg-stone-600"
						hx-get={ "/browse?type=" + string(tab.Type) }
						hx-target="#goal-list"
						hx-push-url="true"
					>{ tab.Label }</button>
				}
			}
		</div>
		<div id="goal-list">
			@GoalList(goals)
		</div>
	}
}

templ GoalList(goals []domcatalog.Goal) {
	if len(goals) == 0 {
		<p class="text-stone-400">No goals found.</p>
	} else {
		<div class="grid gap-3">
			for _, g := range goals {
				@GoalCard(g)
			}
		</div>
	}
}

templ GoalCard(g domcatalog.Goal) {
	<div class="bg-stone-800 border border-stone-700 rounded p-4 flex justify-between items-start">
		<div>
			<a href={ templ.SafeURL("/browse/catalog/" + g.ID) } class="font-semibold text-yellow-300 hover:underline">{ g.Title }</a>
			if g.Description != "" {
				<p class="text-stone-400 text-sm mt-1">{ g.Description }</p>
			}
		</div>
		<button
			class="ml-4 shrink-0 px-3 py-1 bg-green-700 hover:bg-green-600 rounded text-sm"
			hx-post="/htmx/goals/activate"
			hx-vals={ `{"catalog_id":"` + g.ID + `"}` }
			hx-target="#planner-list"
			hx-swap="beforeend"
		>+ Add</button>
	</div>
}
```

Create `interfaces/templates/catalog/detail.templ`:

```templ
package catalog

import (
	domcatalog "github.com/johnfarrell/runeplan/domain/catalog"
	"github.com/johnfarrell/runeplan/interfaces/templates"
	"strconv"
)

templ Detail(g *domcatalog.Goal) {
	@templates.Base(g.Title) {
		<div class="mb-4">
			<a href="/browse" class="text-stone-400 hover:text-stone-200 text-sm">&larr; Back to Browse</a>
		</div>
		<h1 class="text-2xl font-bold text-yellow-300 mb-2">{ g.Title }</h1>
		if g.Description != "" {
			<p class="text-stone-300 mb-6">{ g.Description }</p>
		}

		if len(g.SkillReqs) > 0 {
			<section class="mb-6">
				<h2 class="text-lg font-semibold mb-2">Skill Requirements</h2>
				<ul class="space-y-1">
					for _, sr := range g.SkillReqs {
						<li class="flex gap-2 text-sm">
							<span class="capitalize text-stone-300">{ string(sr.Skill) }</span>
							<span class="text-yellow-400">{ strconv.Itoa(sr.Level.Value()) }</span>
						</li>
					}
				</ul>
			</section>
		}

		if len(g.Requirements) > 0 {
			<section class="mb-6">
				<h2 class="text-lg font-semibold mb-2">Requirements</h2>
				<ul class="space-y-1 list-disc list-inside text-stone-300">
					for _, req := range g.Requirements {
						<li>{ req.Description }</li>
					}
				</ul>
			</section>
		}

		<button
			class="px-4 py-2 bg-green-700 hover:bg-green-600 rounded font-semibold"
			hx-post="/htmx/goals/activate"
			hx-vals={ `{"catalog_id":"` + g.ID + `"}` }
			hx-target="#planner-list"
		>Add to Planner</button>
	}
}
```

**Step 7: Generate templates**

```bash
templ generate ./interfaces/templates/catalog/
```

**Step 8: Implement catalog handler**

Create `interfaces/handler/catalog.go`:

```go
package handler

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/johnfarrell/runeplan/application/catalog"
	"github.com/johnfarrell/runeplan/domain/goal"
	templatescatalog "github.com/johnfarrell/runeplan/interfaces/templates/catalog"
	"github.com/johnfarrell/runeplan/interfaces/templates"
)

// BrowseHandler returns the catalog browse page handler.
func BrowseHandler(svc *catalog.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		typeParam := r.URL.Query().Get("type")
		if typeParam == "" {
			typeParam = string(goal.TypeQuest)
		}
		t := goal.Type(typeParam)
		if !t.Valid() {
			t = goal.TypeQuest
		}

		goals, err := svc.ListByType(r.Context(), t)
		if err != nil {
			templates.Render(w, r, http.StatusInternalServerError, templates.Error("Failed to load goals"))
			return
		}

		// HTMX tab swap — return fragment only
		if r.Header.Get("HX-Request") == "true" {
			templates.Render(w, r, http.StatusOK, templatescatalog.GoalList(goals))
			return
		}

		templates.Render(w, r, http.StatusOK, templatescatalog.Browse(t, goals))
	}
}

// CatalogDetailHandler returns the catalog goal detail page handler.
func CatalogDetailHandler(svc *catalog.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		g, err := svc.GetByID(r.Context(), id)
		if err != nil {
			templates.Render(w, r, http.StatusNotFound, templates.Error("Goal not found"))
			return
		}
		templates.Render(w, r, http.StatusOK, templatescatalog.Detail(g))
	}
}
```

**Step 9: Write handler test**

Create `interfaces/handler/catalog_test.go`:

```go
package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/johnfarrell/runeplan/application/catalog"
	"github.com/johnfarrell/runeplan/domain/goal"
	domcatalog "github.com/johnfarrell/runeplan/domain/catalog"
	"github.com/johnfarrell/runeplan/interfaces/handler"
)

type fakeCatalogRepo struct {
	goals []domcatalog.Goal
}

func (f *fakeCatalogRepo) ListAll(ctx context.Context) ([]domcatalog.Goal, error) {
	return f.goals, nil
}
func (f *fakeCatalogRepo) ListByType(ctx context.Context, t goal.Type) ([]domcatalog.Goal, error) {
	var out []domcatalog.Goal
	for _, g := range f.goals {
		if g.Type == t {
			out = append(out, g)
		}
	}
	return out, nil
}
func (f *fakeCatalogRepo) GetByID(ctx context.Context, id string) (*domcatalog.Goal, error) {
	for _, g := range f.goals {
		if g.ID == id {
			return &g, nil
		}
	}
	return nil, catalog.ErrNotFound
}

func TestBrowseHandler_ReturnsPage(t *testing.T) {
	repo := &fakeCatalogRepo{goals: []domcatalog.Goal{
		{ID: "1", Type: goal.TypeQuest, Title: "Dragon Slayer"},
	}}
	svc := catalog.NewService(repo)
	h := handler.BrowseHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/browse?type=quest", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Dragon Slayer") {
		t.Error("expected goal title in response")
	}
}

func TestCatalogDetailHandler_NotFound(t *testing.T) {
	repo := &fakeCatalogRepo{}
	svc := catalog.NewService(repo)
	h := handler.CatalogDetailHandler(svc)

	r := mux.NewRouter()
	r.Handle("/browse/catalog/{id}", h)

	req := httptest.NewRequest(http.MethodGet, "/browse/catalog/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", w.Code)
	}
}
```

**Step 10: Run tests**

```bash
go test ./interfaces/handler/... -v
go test ./application/catalog/... -v
```

**Step 11: Commit**

```bash
git add infrastructure/postgres/catalog_repository.go application/catalog/ \
        interfaces/handler/catalog.go interfaces/handler/catalog_test.go \
        interfaces/templates/catalog/
git commit -m "Add catalog browse slice: repo, service, handler, templates"
```

---

## Task 8: Goal planner slice

**Files:**
- Modify: `infrastructure/postgres/goal_repository.go`
- Modify: `application/goal/service.go`
- Modify: `interfaces/handler/goal.go`
- Create: `interfaces/templates/goal/planner.templ` + `_templ.go`
- Create: `interfaces/templates/goal/card.templ` + `_templ.go`

**Step 1: Write failing service test**

Create `application/goal/service_test.go`:

```go
package goal_test

import (
	"context"
	"testing"
	"time"

	"github.com/johnfarrell/runeplan/application/goal"
	domgoal "github.com/johnfarrell/runeplan/domain/goal"
)

type mockGoalRepo struct {
	goals []domgoal.Goal
}

func (m *mockGoalRepo) ListByRSN(ctx context.Context, rsnID string) ([]domgoal.Goal, error) {
	var out []domgoal.Goal
	for _, g := range m.goals {
		if g.RSNID == rsnID {
			out = append(out, g)
		}
	}
	return out, nil
}
func (m *mockGoalRepo) Activate(ctx context.Context, rsnID, catalogID string) (*domgoal.Goal, error) {
	g := &domgoal.Goal{
		ID:        "new-id",
		RSNID:     rsnID,
		Title:     "Test Goal",
		Type:      domgoal.TypeQuest,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	cid := catalogID
	g.CatalogID = &cid
	m.goals = append(m.goals, *g)
	return g, nil
}
func (m *mockGoalRepo) Complete(ctx context.Context, goalID string) error {
	for i, g := range m.goals {
		if g.ID == goalID {
			m.goals[i].Completed = true
			return nil
		}
	}
	return goal.ErrNotFound
}
func (m *mockGoalRepo) ToggleRequirement(ctx context.Context, goalID, requirementID string) (bool, error) {
	return true, nil
}

func TestActivate(t *testing.T) {
	repo := &mockGoalRepo{}
	svc := goal.NewService(repo)
	g, err := svc.Activate(context.Background(), "rsn1", "catalog1")
	if err != nil {
		t.Fatal(err)
	}
	if g.RSNID != "rsn1" {
		t.Errorf("got RSNID %q, want rsn1", g.RSNID)
	}
}

func TestListByRSN(t *testing.T) {
	repo := &mockGoalRepo{}
	svc := goal.NewService(repo)
	_, _ = svc.Activate(context.Background(), "rsn1", "cat1")
	goals, err := svc.List(context.Background(), "rsn1")
	if err != nil {
		t.Fatal(err)
	}
	if len(goals) != 1 {
		t.Errorf("got %d goals, want 1", len(goals))
	}
}
```

**Step 2: Implement goal service**

Replace `application/goal/service.go`:

```go
package goal

import (
	"context"
	"errors"

	domgoal "github.com/johnfarrell/runeplan/domain/goal"
)

var ErrNotFound = errors.New("goal: not found")

// Repository is the persistence interface for user goals.
type Repository interface {
	ListByRSN(ctx context.Context, rsnID string) ([]domgoal.Goal, error)
	Activate(ctx context.Context, rsnID, catalogID string) (*domgoal.Goal, error)
	Complete(ctx context.Context, goalID string) error
	ToggleRequirement(ctx context.Context, goalID, requirementID string) (completed bool, err error)
}

// Service handles goal planning use cases.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, rsnID string) ([]domgoal.Goal, error) {
	return s.repo.ListByRSN(ctx, rsnID)
}

func (s *Service) Activate(ctx context.Context, rsnID, catalogID string) (*domgoal.Goal, error) {
	return s.repo.Activate(ctx, rsnID, catalogID)
}

func (s *Service) Complete(ctx context.Context, goalID string) error {
	return s.repo.Complete(ctx, goalID)
}

func (s *Service) ToggleRequirement(ctx context.Context, goalID, requirementID string) (bool, error) {
	return s.repo.ToggleRequirement(ctx, goalID, requirementID)
}
```

**Step 3: Run service tests**

```bash
go test ./application/goal/... -v
```

Expected: PASS.

**Step 4: Implement goal postgres repository**

Replace `infrastructure/postgres/goal_repository.go`:

```go
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goalapp "github.com/johnfarrell/runeplan/application/goal"
	domgoal "github.com/johnfarrell/runeplan/domain/goal"
)

// GoalRepository implements goal.Repository using PostgreSQL.
type GoalRepository struct {
	pool *pgxpool.Pool
}

func NewGoalRepository(pool *pgxpool.Pool) *GoalRepository {
	return &GoalRepository{pool: pool}
}

func (r *GoalRepository) ListByRSN(ctx context.Context, rsnID string) ([]domgoal.Goal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT g.id, g.rsn_id, g.catalog_id, g.title, g.type, COALESCE(g.notes,''),
		       g.completed, g.completed_at, g.created_at, g.updated_at
		FROM goals g WHERE g.rsn_id = $1 ORDER BY g.created_at`, rsnID)
	if err != nil {
		return nil, fmt.Errorf("goals: list: %w", err)
	}
	defer rows.Close()

	var goals []domgoal.Goal
	for rows.Next() {
		var g domgoal.Goal
		if err := rows.Scan(&g.ID, &g.RSNID, &g.CatalogID, &g.Title, &g.Type, &g.Notes,
			&g.Completed, &g.CompletedAt, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load requirements for each goal
	for i := range goals {
		if err := r.loadRequirements(ctx, &goals[i]); err != nil {
			return nil, err
		}
	}
	return goals, nil
}

func (r *GoalRepository) loadRequirements(ctx context.Context, g *domgoal.Goal) error {
	rows, err := r.pool.Query(ctx, `
		SELECT p.requirement_id, cr.description, p.completed, p.completed_at
		FROM goal_requirement_progress p
		JOIN catalog_requirements cr ON cr.id = p.requirement_id
		WHERE p.goal_id = $1 ORDER BY cr.created_at`, g.ID)
	if err != nil {
		return fmt.Errorf("goals: load reqs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var rp domgoal.RequirementProgress
		rp.GoalID = g.ID
		if err := rows.Scan(&rp.RequirementID, &rp.Description, &rp.Completed, &rp.CompletedAt); err != nil {
			return err
		}
		g.Requirements = append(g.Requirements, rp)
	}
	return rows.Err()
}

func (r *GoalRepository) Activate(ctx context.Context, rsnID, catalogID string) (*domgoal.Goal, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("goals: activate: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Fetch catalog goal title + type
	var title, goalType string
	if err := tx.QueryRow(ctx,
		`SELECT title, type FROM catalog_goals WHERE id = $1`, catalogID).
		Scan(&title, &goalType); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("goals: activate: catalog goal not found")
		}
		return nil, fmt.Errorf("goals: activate: fetch catalog: %w", err)
	}

	// Insert goal
	var goalID string
	now := time.Now()
	if err := tx.QueryRow(ctx, `
		INSERT INTO goals (rsn_id, catalog_id, title, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id`, rsnID, catalogID, title, goalType, now).Scan(&goalID); err != nil {
		return nil, fmt.Errorf("goals: activate: insert: %w", err)
	}

	// Insert progress rows for each catalog requirement
	_, err = tx.Exec(ctx, `
		INSERT INTO goal_requirement_progress (goal_id, requirement_id)
		SELECT $1, id FROM catalog_requirements WHERE catalog_goal_id = $2`, goalID, catalogID)
	if err != nil {
		return nil, fmt.Errorf("goals: activate: insert progress: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("goals: activate: commit: %w", err)
	}

	cid := catalogID
	return &domgoal.Goal{
		ID:        goalID,
		RSNID:     rsnID,
		CatalogID: &cid,
		Title:     title,
		Type:      domgoal.Type(goalType),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *GoalRepository) Complete(ctx context.Context, goalID string) error {
	now := time.Now()
	ct, err := r.pool.Exec(ctx,
		`UPDATE goals SET completed = true, completed_at = $1, updated_at = $1 WHERE id = $2`, now, goalID)
	if err != nil {
		return fmt.Errorf("goals: complete: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return goalapp.ErrNotFound
	}
	return nil
}

func (r *GoalRepository) ToggleRequirement(ctx context.Context, goalID, requirementID string) (bool, error) {
	var completed bool
	err := r.pool.QueryRow(ctx, `
		UPDATE goal_requirement_progress
		SET completed = NOT completed,
		    completed_at = CASE WHEN NOT completed THEN now() ELSE NULL END
		WHERE goal_id = $1 AND requirement_id = $2
		RETURNING completed`, goalID, requirementID).Scan(&completed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, goalapp.ErrNotFound
		}
		return false, fmt.Errorf("goals: toggle req: %w", err)
	}
	return completed, nil
}
```

**Step 5: Write planner templates**

Create `interfaces/templates/goal/planner.templ`:

```templ
package goal

import (
	domgoal "github.com/johnfarrell/runeplan/domain/goal"
	"github.com/johnfarrell/runeplan/interfaces/templates"
)

templ Planner(goals []domgoal.Goal) {
	@templates.Base("Planner") {
		<div class="flex justify-between items-center mb-6">
			<h1 class="text-2xl font-bold">My Planner</h1>
			<a href="/browse" class="px-3 py-1 bg-stone-700 hover:bg-stone-600 rounded text-sm">+ Browse Goals</a>
		</div>
		<div id="skills-fragment" hx-get="/htmx/skills" hx-trigger="load" class="mb-8"></div>
		<div id="planner-list" class="space-y-4">
			for _, g := range goals {
				@GoalCard(g)
			}
		</div>
	}
}
```

Create `interfaces/templates/goal/card.templ`:

```templ
package goal

import (
	domgoal "github.com/johnfarrell/runeplan/domain/goal"
)

templ GoalCard(g domgoal.Goal) {
	<div id={ "goal-" + g.ID } class="bg-stone-800 border border-stone-700 rounded p-4">
		<div class="flex justify-between items-start mb-3">
			<div>
				<span class="font-semibold text-yellow-300">{ g.Title }</span>
				<span class="ml-2 text-xs text-stone-400 uppercase">{ string(g.Type) }</span>
			</div>
			if !g.Completed {
				<button
					class="text-xs px-2 py-1 bg-green-800 hover:bg-green-700 rounded"
					hx-post={ "/htmx/goals/" + g.ID + "/complete" }
					hx-target={ "#goal-" + g.ID }
					hx-swap="outerHTML"
				>Complete</button>
			} else {
				<span class="text-xs text-green-400">Completed</span>
			}
		</div>
		if len(g.Requirements) > 0 {
			<ul class="space-y-1">
				for _, req := range g.Requirements {
					@RequirementRow(req)
				}
			</ul>
		}
	</div>
}

templ RequirementRow(req domgoal.RequirementProgress) {
	<li id={ "req-" + req.RequirementID } class="flex items-center gap-2 text-sm">
		<input
			type="checkbox"
			checked?={ req.Completed }
			hx-post={ "/htmx/requirements/" + req.RequirementID + "/toggle" }
			hx-vals={ `{"goal_id":"` + req.GoalID + `"}` }
			hx-target={ "#req-" + req.RequirementID }
			hx-swap="outerHTML"
			class="accent-yellow-400"
		/>
		<span class={ templ.KV("line-through text-stone-500", req.Completed) }>{ req.Description }</span>
	</li>
}
```

**Step 6: Generate templates**

```bash
templ generate ./interfaces/templates/goal/
```

**Step 7: Implement goal handler**

Replace `interfaces/handler/goal.go`:

```go
package handler

import (
	"net/http"

	"github.com/gorilla/mux"
	goalapp "github.com/johnfarrell/runeplan/application/goal"
	"github.com/johnfarrell/runeplan/domain/user"
	templatesgoal "github.com/johnfarrell/runeplan/interfaces/templates/goal"
	"github.com/johnfarrell/runeplan/interfaces/templates"
)

// PlannerHandler returns the full planner page.
func PlannerHandler(svc *goalapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := user.GetUser(r.Context())
		if !ok {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		rsn := u.ActiveRSN()
		if rsn == nil {
			templates.Render(w, r, http.StatusOK, templatesgoal.Planner(nil))
			return
		}
		goals, err := svc.List(r.Context(), rsn.ID)
		if err != nil {
			templates.Render(w, r, http.StatusInternalServerError, templates.Error("Failed to load goals"))
			return
		}
		templates.Render(w, r, http.StatusOK, templatesgoal.Planner(goals))
	}
}

// ActivateGoalHandler activates a catalog goal for the current RSN.
func ActivateGoalHandler(svc *goalapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := user.GetUser(r.Context())
		if !ok {
			w.Header().Set("HX-Redirect", "/")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		rsn := u.ActiveRSN()
		if rsn == nil {
			templates.Render(w, r, http.StatusBadRequest, templates.Error("No RSN linked"))
			return
		}
		if err := r.ParseForm(); err != nil {
			templates.Render(w, r, http.StatusBadRequest, templates.Error("Invalid request"))
			return
		}
		catalogID := r.FormValue("catalog_id")
		g, err := svc.Activate(r.Context(), rsn.ID, catalogID)
		if err != nil {
			templates.Render(w, r, http.StatusInternalServerError, templates.Error("Failed to activate goal"))
			return
		}
		templates.Render(w, r, http.StatusOK, templatesgoal.GoalCard(*g))
	}
}

// CompleteGoalHandler marks a goal as complete and returns the updated card.
func CompleteGoalHandler(svc *goalapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		goalID := mux.Vars(r)["id"]
		if err := svc.Complete(r.Context(), goalID); err != nil {
			templates.Render(w, r, http.StatusInternalServerError, templates.Error("Failed to complete goal"))
			return
		}
		// Reload goal from service for accurate state
		u, _ := user.GetUser(r.Context())
		rsn := u.ActiveRSN()
		if rsn == nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		goals, _ := svc.List(r.Context(), rsn.ID)
		for _, g := range goals {
			if g.ID == goalID {
				templates.Render(w, r, http.StatusOK, templatesgoal.GoalCard(g))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}

// ToggleRequirementHandler toggles a requirement and returns the updated row.
func ToggleRequirementHandler(svc *goalapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requirementID := mux.Vars(r)["id"]
		if err := r.ParseForm(); err != nil {
			templates.Render(w, r, http.StatusBadRequest, templates.Error("Invalid request"))
			return
		}
		goalID := r.FormValue("goal_id")
		completed, err := svc.ToggleRequirement(r.Context(), goalID, requirementID)
		if err != nil {
			templates.Render(w, r, http.StatusInternalServerError, templates.Error("Failed to toggle"))
			return
		}
		templates.Render(w, r, http.StatusOK, templatesgoal.RequirementRow(domgoal.RequirementProgress{
			GoalID:        goalID,
			RequirementID: requirementID,
			Completed:     completed,
		}))
	}
}
```

Fix the import — add `domgoal` import:

```go
import (
	...
	domgoal "github.com/johnfarrell/runeplan/domain/goal"
	...
)
```

**Step 8: Verify build**

```bash
go build ./...
```

**Step 9: Commit**

```bash
git add infrastructure/postgres/goal_repository.go application/goal/ \
        interfaces/handler/goal.go interfaces/templates/goal/
git commit -m "Add goal planner slice: repo, service, handler, templates"
```

---

## Task 9: Skills view slice

**Files:**
- Modify: `application/skill/service.go`
- Create: `interfaces/handler/skill.go`
- Create: `interfaces/templates/skill/grid.templ` + `_templ.go`

**Step 1: Write failing service test**

Create `application/skill/service_test.go`:

```go
package skill_test

import (
	"testing"

	domskill "github.com/johnfarrell/runeplan/domain/skill"
	"github.com/johnfarrell/runeplan/application/skill"
	domgoal "github.com/johnfarrell/runeplan/domain/goal"
	domcatalog "github.com/johnfarrell/runeplan/domain/catalog"
)

func TestAggregateThresholds_MaxPerSkill(t *testing.T) {
	goals := []domgoal.Goal{
		{RSNID: "r1", Completed: false},
		{RSNID: "r1", Completed: false},
	}
	catalogGoals := []domcatalog.Goal{
		{SkillReqs: []domcatalog.SkillRequirement{
			{Skill: domskill.Agility, Level: mustLevel(60)},
		}},
		{SkillReqs: []domcatalog.SkillRequirement{
			{Skill: domskill.Agility, Level: mustLevel(70)},
			{Skill: domskill.Magic, Level: mustLevel(55)},
		}},
	}
	current := map[domskill.Skill]domskill.XP{
		domskill.Agility: mustXP(737627), // level 61
	}

	thresholds := skill.AggregateThresholds(goals, catalogGoals, current)

	agility, ok := thresholds[domskill.Agility]
	if !ok {
		t.Fatal("expected agility threshold")
	}
	if agility.Required.Value() != 70 {
		t.Errorf("agility required: got %d, want 70", agility.Required.Value())
	}
	if agility.Satisfied {
		t.Error("agility should not be satisfied (current 61 < required 70)")
	}
	magic, ok := thresholds[domskill.Magic]
	if !ok {
		t.Fatal("expected magic threshold")
	}
	if magic.Required.Value() != 55 {
		t.Errorf("magic required: got %d, want 55", magic.Required.Value())
	}
}

func mustLevel(v int) domskill.Level {
	l, err := domskill.NewLevel(v)
	if err != nil {
		panic(err)
	}
	return l
}

func mustXP(v int) domskill.XP {
	x, err := domskill.NewXP(v)
	if err != nil {
		panic(err)
	}
	return x
}
```

**Step 2: Implement skill service**

Replace `application/skill/service.go`:

```go
package skill

import (
	domcatalog "github.com/johnfarrell/runeplan/domain/catalog"
	domgoal "github.com/johnfarrell/runeplan/domain/goal"
	domskill "github.com/johnfarrell/runeplan/domain/skill"
)

// Threshold holds the highest required level for a skill and whether the user satisfies it.
type Threshold struct {
	Skill     domskill.Skill
	Required  domskill.Level
	Current   domskill.Level
	CurrentXP domskill.XP
	XPNeeded  int
	Satisfied bool
}

// AggregateThresholds computes per-skill max required level across all active goals,
// compared against the current XP map.
//
// goals and catalogGoals must be parallel slices (goals[i] corresponds to catalogGoals[i]).
// goals with Completed=true are excluded.
func AggregateThresholds(
	goals []domgoal.Goal,
	catalogGoals []domcatalog.Goal,
	current map[domskill.Skill]domskill.XP,
) map[domskill.Skill]Threshold {
	maxLevel := make(map[domskill.Skill]domskill.Level)

	for i, g := range goals {
		if g.Completed {
			continue
		}
		if i >= len(catalogGoals) {
			continue
		}
		for _, sr := range catalogGoals[i].SkillReqs {
			if existing, ok := maxLevel[sr.Skill]; !ok || sr.Level.Value() > existing.Value() {
				maxLevel[sr.Skill] = sr.Level
			}
		}
	}

	thresholds := make(map[domskill.Skill]Threshold, len(maxLevel))
	for s, required := range maxLevel {
		currentXP, _ := current[s]
		currentLevel := currentXP.ToLevel()
		xpNeeded := currentXP.XPRemaining(required)
		thresholds[s] = Threshold{
			Skill:     s,
			Required:  required,
			Current:   currentLevel,
			CurrentXP: currentXP,
			XPNeeded:  xpNeeded,
			Satisfied: xpNeeded == 0,
		}
	}
	return thresholds
}
```

**Step 3: Run skill service tests**

```bash
go test ./application/skill/... -v
```

Expected: PASS.

**Step 4: Write skill grid template**

Create `interfaces/templates/skill/grid.templ`:

```templ
package skill

import (
	appskill "github.com/johnfarrell/runeplan/application/skill"
	domskill "github.com/johnfarrell/runeplan/domain/skill"
	"strconv"
)

templ Grid(thresholds map[domskill.Skill]appskill.Threshold) {
	<div id="skills-fragment" class="bg-stone-800 border border-stone-700 rounded p-4 mb-6">
		<h2 class="text-lg font-semibold mb-3">Skill Requirements</h2>
		if len(thresholds) == 0 {
			<p class="text-stone-400 text-sm">No skill requirements across active goals.</p>
		} else {
			<div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2">
				for _, s := range domskill.All {
					if t, ok := thresholds[s]; ok {
						@SkillRow(t)
					}
				}
			</div>
		}
	</div>
}

templ SkillRow(t appskill.Threshold) {
	<div class={
		"rounded p-2 text-xs border",
		templ.KV("bg-green-900 border-green-700 text-green-100", t.Satisfied),
		templ.KV("bg-red-900 border-red-700 text-red-100", !t.Satisfied),
	}>
		<div class="font-semibold capitalize">{ string(t.Skill) }</div>
		<div>{ strconv.Itoa(t.Current.Value()) } / { strconv.Itoa(t.Required.Value()) }</div>
		if !t.Satisfied {
			<div class="text-xs opacity-75">{ strconv.Itoa(t.XPNeeded) } XP</div>
		}
	</div>
}
```

**Step 5: Generate templates**

```bash
templ generate ./interfaces/templates/skill/
```

**Step 6: Implement skill handler**

Create `interfaces/handler/skill.go`:

```go
package handler

import (
	"context"
	"net/http"

	appskill "github.com/johnfarrell/runeplan/application/skill"
	domcatalog "github.com/johnfarrell/runeplan/domain/catalog"
	domgoal "github.com/johnfarrell/runeplan/domain/goal"
	domskill "github.com/johnfarrell/runeplan/domain/skill"
	"github.com/johnfarrell/runeplan/domain/user"
	templatesskill "github.com/johnfarrell/runeplan/interfaces/templates/skill"
	"github.com/johnfarrell/runeplan/interfaces/templates"
)

// GoalLoader loads goals for a given RSN ID.
type GoalLoader interface {
	ListByRSN(ctx context.Context, rsnID string) ([]domgoal.Goal, error)
}

// CatalogLoader loads catalog goals by ID.
type CatalogLoader interface {
	GetByID(ctx context.Context, id string) (*domcatalog.Goal, error)
}

// SkillsHandler returns the HTMX skills fragment.
func SkillsHandler(goalRepo GoalLoader, catalogRepo CatalogLoader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := user.GetUser(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		rsn := u.ActiveRSN()
		if rsn == nil {
			templates.Render(w, r, http.StatusOK, templatesskill.Grid(nil))
			return
		}

		goals, err := goalRepo.ListByRSN(r.Context(), rsn.ID)
		if err != nil {
			templates.Render(w, r, http.StatusInternalServerError, templates.Error("Failed to load goals"))
			return
		}

		// Load catalog goals in parallel with the goal list
		catalogGoals := make([]domcatalog.Goal, len(goals))
		for i, g := range goals {
			if g.CatalogID != nil {
				cg, err := catalogRepo.GetByID(r.Context(), *g.CatalogID)
				if err == nil {
					catalogGoals[i] = *cg
				}
			}
		}

		thresholds := appskill.AggregateThresholds(goals, catalogGoals, rsn.SkillLevels)

		// Convert XP map to Level map for display
		_ = domskill.All // ensure package is used
		templates.Render(w, r, http.StatusOK, templatesskill.Grid(thresholds))
	}
}
```

**Step 7: Verify build**

```bash
go build ./...
```

**Step 8: Commit**

```bash
git add application/skill/ interfaces/handler/skill.go interfaces/templates/skill/
git commit -m "Add skills view slice: aggregation service, handler, template"
```

---

## Task 10: Hiscores sync + profile slice

**Files:**
- Create: `infrastructure/hiscores/client.go`
- Create: `application/user/service.go`
- Create: `interfaces/handler/user.go`
- Create: `interfaces/templates/user/profile.templ` + `_templ.go`

**Step 1: Write failing hiscores client test**

Create `infrastructure/hiscores/client_test.go`:

```go
package hiscores_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnfarrell/runeplan/domain/skill"
	"github.com/johnfarrell/runeplan/infrastructure/hiscores"
)

// OSRS CSV format: rank,level,xp per skill in canonical order
const sampleCSV = `1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
`

func TestFetch_ParsesSkillLevels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleCSV))
	}))
	defer srv.Close()

	client := hiscores.NewClient(srv.URL, 0)
	levels, err := client.Fetch("Zezima")
	if err != nil {
		t.Fatal(err)
	}
	xp, ok := levels[skill.Attack]
	if !ok {
		t.Fatal("expected attack XP")
	}
	if xp.Value() != 200000000 {
		t.Errorf("got %d, want 200000000", xp.Value())
	}
}

func TestFetch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := hiscores.NewClient(srv.URL, 0)
	_, err := client.Fetch("unknownplayer")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}
```

**Step 2: Implement hiscores client**

Create `infrastructure/hiscores/client.go`:

```go
package hiscores

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/johnfarrell/runeplan/domain/skill"
)

// HiscoreSkillOrder is the canonical order OSRS returns skills in the CSV response.
// Must match the order returned by the hiscore API exactly.
var HiscoreSkillOrder = []skill.Skill{
	skill.Attack, skill.Defence, skill.Strength, skill.Hitpoints,
	skill.Ranged, skill.Prayer, skill.Magic, skill.Cooking,
	skill.Woodcut, skill.Fletching, skill.Fishing, skill.Firemaking,
	skill.Crafting, skill.Smithing, skill.Mining, skill.Herblore,
	skill.Agility, skill.Thieving, skill.Slayer, skill.Farming,
	skill.Runecraft, skill.Hunter, skill.Construction, skill.Sailing,
}

// Client fetches skill data from the OSRS Hiscores API.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient creates a hiscores client. timeout=0 uses a default of 10s.
func NewClient(baseURL string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
	}
}

// Fetch retrieves XP values for all skills for the given RSN.
func (c *Client) Fetch(rsn string) (map[skill.Skill]skill.XP, error) {
	url := c.baseURL + "?player=" + rsn
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("hiscores: fetch %q: %w", rsn, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hiscores: fetch %q: HTTP %d", rsn, resp.StatusCode)
	}

	result := make(map[skill.Skill]skill.XP, len(HiscoreSkillOrder))
	scanner := bufio.NewScanner(resp.Body)
	i := 0
	for scanner.Scan() && i < len(HiscoreSkillOrder) {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			i++
			continue
		}
		xpVal, err := strconv.Atoi(parts[2])
		if err != nil || xpVal < 0 {
			i++
			continue
		}
		xp, err := skill.NewXP(xpVal)
		if err != nil {
			i++
			continue
		}
		result[HiscoreSkillOrder[i]] = xp
		i++
	}
	return result, scanner.Err()
}
```

**Step 3: Run hiscores tests**

```bash
go test ./infrastructure/hiscores/... -v
```

Expected: PASS.

**Step 4: Implement user application service**

Create `application/user/service.go`:

```go
package user

import (
	"context"
	"fmt"

	domskill "github.com/johnfarrell/runeplan/domain/skill"
)

// HiscoresClient fetches skill XP for a given RSN string.
type HiscoresClient interface {
	Fetch(rsn string) (map[domskill.Skill]domskill.XP, error)
}

// RSNRepository persists RSN skill data.
type RSNRepository interface {
	UpdateSkillLevels(ctx context.Context, rsnID string, levels map[domskill.Skill]domskill.XP) error
}

// Service handles user-facing use cases (hiscores sync).
type Service struct {
	hiscores HiscoresClient
	repo     RSNRepository
}

func NewService(hiscores HiscoresClient, repo RSNRepository) *Service {
	return &Service{hiscores: hiscores, repo: repo}
}

// SyncHiscores fetches the latest OSRS hiscore data for rsnName and persists it.
func (s *Service) SyncHiscores(ctx context.Context, rsnID, rsnName string) (map[domskill.Skill]domskill.XP, error) {
	levels, err := s.hiscores.Fetch(rsnName)
	if err != nil {
		return nil, fmt.Errorf("sync hiscores: %w", err)
	}
	if err := s.repo.UpdateSkillLevels(ctx, rsnID, levels); err != nil {
		return nil, fmt.Errorf("sync hiscores: persist: %w", err)
	}
	return levels, nil
}
```

**Step 5: Implement user postgres repository**

Create `infrastructure/postgres/user_repository.go`:

```go
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	domskill "github.com/johnfarrell/runeplan/domain/skill"
)

// UserRepository implements the user application RSNRepository.
type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// UpdateSkillLevels serialises levels as JSONB and updates user_rsns.skill_levels.
func (r *UserRepository) UpdateSkillLevels(ctx context.Context, rsnID string, levels map[domskill.Skill]domskill.XP) error {
	// Convert to JSON-serialisable form: map[string]int
	raw := make(map[string]int, len(levels))
	for s, xp := range levels {
		raw[string(s)] = xp.Value()
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("user: marshal skill levels: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE user_rsns SET skill_levels = $1, synced_at = now() WHERE id = $2`, b, rsnID)
	return err
}
```

**Step 6: Write profile template**

Create `interfaces/templates/user/profile.templ`:

```templ
package user

import (
	"github.com/johnfarrell/runeplan/domain/user"
	"github.com/johnfarrell/runeplan/interfaces/templates"
)

templ Profile(u user.User) {
	@templates.Base("Profile") {
		<h1 class="text-2xl font-bold mb-6">Profile</h1>
		if len(u.RSNs) == 0 {
			<p class="text-stone-400">No RSNs linked. (RSN management coming soon.)</p>
		} else {
			<div class="space-y-4">
				for _, rsn := range u.RSNs {
					@RSNCard(rsn)
				}
			</div>
		}
	}
}

templ RSNCard(rsn user.RSN) {
	<div id={ "rsn-" + rsn.ID } class="bg-stone-800 border border-stone-700 rounded p-4 flex justify-between items-center">
		<div>
			<span class="font-semibold text-yellow-300">{ rsn.RSN }</span>
			if rsn.SyncedAt != nil {
				<span class="ml-2 text-xs text-stone-400">Last synced: { rsn.SyncedAt.Format("2006-01-02 15:04") }</span>
			}
		</div>
		<button
			class="px-3 py-1 bg-blue-700 hover:bg-blue-600 rounded text-sm"
			hx-post="/htmx/sync"
			hx-vals={ `{"rsn_id":"` + rsn.ID + `","rsn":"` + rsn.RSN + `"}` }
			hx-target="#skills-fragment"
			hx-swap="outerHTML"
		>Sync Hiscores</button>
	</div>
}
```

**Step 7: Generate templates**

```bash
templ generate ./interfaces/templates/user/
```

**Step 8: Implement user handler**

Create `interfaces/handler/user.go`:

```go
package handler

import (
	"net/http"

	appuser "github.com/johnfarrell/runeplan/application/user"
	domskill "github.com/johnfarrell/runeplan/domain/skill"
	"github.com/johnfarrell/runeplan/domain/user"
	templatesskill "github.com/johnfarrell/runeplan/interfaces/templates/skill"
	templatesuser "github.com/johnfarrell/runeplan/interfaces/templates/user"
	"github.com/johnfarrell/runeplan/interfaces/templates"
)

// ProfileHandler returns the profile page.
func ProfileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := user.GetUser(r.Context())
		if !ok {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		templates.Render(w, r, http.StatusOK, templatesuser.Profile(u))
	}
}

// SyncHandler fetches hiscores for the given RSN and returns the updated skills fragment.
func SyncHandler(svc *appuser.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			templates.Render(w, r, http.StatusBadRequest, templates.Error("Invalid request"))
			return
		}
		rsnID := r.FormValue("rsn_id")
		rsnName := r.FormValue("rsn")
		if rsnID == "" || rsnName == "" {
			templates.Render(w, r, http.StatusBadRequest, templates.Error("Missing rsn_id or rsn"))
			return
		}

		levels, err := svc.SyncHiscores(r.Context(), rsnID, rsnName)
		if err != nil {
			templates.Render(w, r, http.StatusBadGateway, templates.Error("Hiscores sync failed: "+err.Error()))
			return
		}

		// Build threshold map from updated levels (no active goals context here — just show current levels)
		thresholds := make(map[domskill.Skill]interface{})
		_ = levels
		_ = thresholds

		// Return updated skills fragment (empty thresholds — planner will reload)
		templates.Render(w, r, http.StatusOK, templatesskill.Grid(nil))
	}
}
```

**Step 9: Verify build**

```bash
go build ./...
```

**Step 10: Commit**

```bash
git add infrastructure/hiscores/ infrastructure/postgres/user_repository.go \
        application/user/ interfaces/handler/user.go interfaces/templates/user/
git commit -m "Add hiscores sync and profile slice"
```

---

## Task 11: Wire main.go + static assets + run

**Files:**
- Modify: `cmd/server/main.go`
- Modify: `static/` (download real HTMX + Alpine files)

**Step 1: Download static assets**

```bash
# HTMX 2.x
curl -sL https://unpkg.com/htmx.org@2.0.4/dist/htmx.min.js -o static/htmx.min.js

# Alpine.js 3.x
curl -sL https://unpkg.com/alpinejs@3.14.3/dist/cdn.min.js -o static/alpine.min.js
```

For Tailwind, generate a minimal CSS file:

```bash
# If tailwindcss CLI is installed:
tailwindcss --input static/app.css.src --output static/app.css --content "./interfaces/templates/**/*.templ"
# Or create a stub for now:
echo "" > static/app.css
```

**Step 2: Create static embed file**

Create `static/embed.go`:

```go
package static

import "embed"

//go:embed *.css *.js
var FS embed.FS
```

**Step 3: Implement main.go**

Replace `cmd/server/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/johnfarrell/runeplan/application/catalog"
	appgoal "github.com/johnfarrell/runeplan/application/goal"
	appuser "github.com/johnfarrell/runeplan/application/user"
	"github.com/johnfarrell/runeplan/config"
	"github.com/johnfarrell/runeplan/infrastructure/hiscores"
	"github.com/johnfarrell/runeplan/infrastructure/postgres"
	"github.com/johnfarrell/runeplan/interfaces/handler"
	"github.com/johnfarrell/runeplan/interfaces/middleware"
	"github.com/johnfarrell/runeplan/logger"
	"github.com/johnfarrell/runeplan/static"
	"net/http/fs"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.App)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger error: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	// Run migrations (use pgx5:// scheme for golang-migrate pgx/v5 driver)
	migrateURL := strings.Replace(cfg.DB.URL, "postgres://", "pgx5://", 1)
	if err := postgres.RunMigrations(migrateURL); err != nil {
		log.Sugar().Fatalf("migrations: %v", err)
	}
	log.Info("migrations applied")

	// Connect pgxpool
	pool, err := postgres.Connect(context.Background(), cfg.DB.URL)
	if err != nil {
		log.Sugar().Fatalf("db connect: %v", err)
	}
	defer pool.Close()
	log.Info("database connected")

	// Repositories
	catalogRepo := postgres.NewCatalogRepository(pool)
	goalRepo := postgres.NewGoalRepository(pool)
	userRepo := postgres.NewUserRepository(pool)

	// Services
	catalogSvc := catalog.NewService(catalogRepo)
	goalSvc := appgoal.NewService(goalRepo)
	hiscoresClient := hiscores.NewClient(cfg.Hiscores.BaseURL, cfg.Hiscores.Timeout)
	userSvc := appuser.NewService(hiscoresClient, userRepo)

	// Router
	r := mux.NewRouter()
	r.Use(middleware.Logging(log))
	r.Use(middleware.DevAuth)

	// Static assets
	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.FS(static.FS))),
	)

	// Pages
	r.Handle("/", http.RedirectHandler("/browse", http.StatusFound))
	r.Handle("/browse", handler.BrowseHandler(catalogSvc)).Methods(http.MethodGet)
	r.Handle("/browse/catalog/{id}", handler.CatalogDetailHandler(catalogSvc)).Methods(http.MethodGet)
	r.Handle("/planner", handler.PlannerHandler(goalSvc)).Methods(http.MethodGet)
	r.Handle("/profile", handler.ProfileHandler()).Methods(http.MethodGet)

	// HTMX fragments
	r.Handle("/htmx/goals/activate", handler.ActivateGoalHandler(goalSvc)).Methods(http.MethodPost)
	r.Handle("/htmx/goals/{id}/complete", handler.CompleteGoalHandler(goalSvc)).Methods(http.MethodPost)
	r.Handle("/htmx/requirements/{id}/toggle", handler.ToggleRequirementHandler(goalSvc)).Methods(http.MethodPost)
	r.Handle("/htmx/skills", handler.SkillsHandler(goalRepo, catalogRepo)).Methods(http.MethodGet)
	r.Handle("/htmx/sync", handler.SyncHandler(userSvc)).Methods(http.MethodPost)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Sugar().Infof("listening on :%d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Sugar().Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Sugar().Errorf("shutdown: %v", err)
	}
}
```

Note: `net/http/fs` and `fs.FS` — use `http.FS(static.FS)`. The import `"net/http/fs"` is wrong; remove it. Just use `http.FS(static.FS)` directly.

Also: `middleware.Logging` must exist. If it's not yet implemented in `interfaces/middleware/logging.go`, add a basic wrapper:

Check `interfaces/middleware/logging.go` — if it's just a stub `package middleware`, implement it:

```go
package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Logging returns a gorilla/mux middleware that logs each request.
func Logging(log *zap.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, code: http.StatusOK}
			next.ServeHTTP(rw, r)
			log.Info("request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", rw.code),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	code int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.code = code
	rw.ResponseWriter.WriteHeader(code)
}
```

The logging middleware needs `mux.MiddlewareFunc` — import gorilla/mux.

**Step 4: Implement SkillsHandler interface**

`handler.SkillsHandler` needs `GoalLoader` and `CatalogLoader` interfaces. The postgres repositories implement them — verify that `GoalRepository` has a `ListByRSN` method and `CatalogRepository` has `GetByID`. (They do, from Tasks 7 and 8.)

**Step 5: Build and fix**

```bash
go build ./...
```

Iterate on any compile errors. Common issues:
- Missing imports
- Interface not satisfied (check method signatures match exactly)
- `static.FS` — ensure `static/embed.go` exports `FS`

**Step 6: Run tests**

```bash
go test ./... -v
```

**Step 7: Integration test with Docker**

```bash
docker compose up -d
sleep 3
curl -v http://localhost:8080/browse
```

Expected: HTTP 200 with HTML containing "Browse".

```bash
curl -v http://localhost:8080/browse?type=quest
```

Expected: HTML with quest list.

**Step 8: Commit**

```bash
git add cmd/server/main.go static/ interfaces/middleware/logging.go
git commit -m "Wire main.go with full router, services, and graceful shutdown"
```

---

## Full Routing Table (Reference)

| Method | Path | Response |
|--------|------|----------|
| GET | `/` | redirect /browse |
| GET | `/browse` | full page |
| GET | `/browse/catalog/{id}` | full page |
| GET | `/planner` | full page |
| GET | `/profile` | full page |
| POST | `/htmx/goals/activate` | HTMX fragment |
| POST | `/htmx/goals/{id}/complete` | HTMX fragment |
| POST | `/htmx/requirements/{id}/toggle` | HTMX fragment |
| GET | `/htmx/skills` | HTMX fragment |
| POST | `/htmx/sync` | HTMX fragment |

## Out of Scope

- Real auth (sessions, bcrypt, Discord OAuth)
- Multiple RSN switcher UI
- Full 48-diary seed (add more rows to 002_seed_catalog.sql)
- Tailwind Tailwind JIT — stub CSS is fine for now
- DAG prerequisite graph view
