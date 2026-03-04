# Backend

Language: Go 1.22+. Stdlib-first — reach for a library only when the stdlib is genuinely insufficient.

---

## Module & Dependencies

```
// go.mod
module github.com/your-org/runeplan

go 1.22

require (
  github.com/go-chi/chi/v5              v5.1.0   // HTTP router
  github.com/jackc/pgx/v5               v5.6.0   // PostgreSQL driver + pgxpool
  github.com/golang-migrate/migrate/v4  v4.17.1  // Database migrations
  github.com/google/uuid                v1.6.0   // UUID generation
  golang.org/x/crypto                   v0.22.0  // bcrypt
)
```

No ORMs. No HTTP client libraries. No config parsing libraries. All queries are hand-written SQL with `pgx` named parameters.

---

## Entrypoint

`cmd/server/main.go` is responsible for four things in order: load environment, initialize the database pool, run pending migrations, start the HTTP server.

```go
func main() {
    pool := db.NewPool(os.Getenv("DATABASE_URL"))
    defer pool.Close()

    db.RunMigrations(pool) // golang-migrate; safe to call on every startup

    r := buildRouter(pool)
    log.Printf("listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", r))
}
```

---

## Router

All routes are registered in a `buildRouter` function that takes the pool and returns a `chi.Router`. Public routes are registered directly; authenticated routes are wrapped in a group with `auth.SessionMiddleware`.

```go
func buildRouter(pool *pgxpool.Pool) http.Handler {
    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)

    // Public
    r.Post("/api/auth/register",        auth.Register(pool))
    r.Post("/api/auth/login",           auth.Login(pool))
    r.Post("/api/auth/logout",          auth.Logout(pool))
    r.Get("/api/auth/discord",          auth.DiscordRedirect)
    r.Get("/api/auth/discord/callback", auth.DiscordCallback(pool))
    r.Get("/health",                    healthHandler)

    // Catalog (read-only, no auth required)
    r.Get("/api/catalog/diaries", catalog.ListDiaries(pool))
    r.Get("/api/catalog/quests",  catalog.ListQuests(pool))
    r.Get("/api/catalog/skills",  catalog.ListSkills)

    // Authenticated
    r.Group(func(r chi.Router) {
        r.Use(auth.SessionMiddleware(pool))

        r.Get("/api/user/me",    user.GetMe(pool))
        r.Patch("/api/user/me",  user.UpdateMe(pool))
        r.Post("/api/user/sync", user.SyncHiscores(pool))
        r.Get("/api/user/export",user.Export(pool))

        r.Get("/api/goals",             goals.List(pool))
        r.Post("/api/goals",            goals.Create(pool))
        r.Patch("/api/goals/{id}",      goals.Update(pool))
        r.Delete("/api/goals/{id}",     goals.Delete(pool))

        r.Get("/api/skills",                           goals.ListSkillLadders(pool))
        r.Post("/api/goals/{id}/skills",               goals.AddSkillThreshold(pool))
        r.Delete("/api/goals/{id}/skills/{skill}",     goals.RemoveSkillThreshold(pool))

        r.Get("/api/requirements",                             reqs.List(pool))
        r.Post("/api/requirements",                            reqs.Create(pool))
        r.Patch("/api/requirements/{id}",                      reqs.Update(pool))
        r.Delete("/api/requirements/{id}",                     reqs.Delete(pool))
        r.Post("/api/goals/{id}/requirements/{req_id}",        reqs.LinkToGoal(pool))
        r.Delete("/api/goals/{id}/requirements/{req_id}",      reqs.UnlinkFromGoal(pool))
    })

    return r
}
```

---

## Authentication

### Session flow

**Register:** Hash password with bcrypt (cost 12), insert user, create session row, set cookie.

**Login:** Fetch user by email, compare bcrypt hash, create session row, set cookie.

**Cookie settings:**
```go
http.SetCookie(w, &http.Cookie{
    Name:     "runeplan_session",
    Value:    sessionToken,
    Path:     "/",
    HttpOnly: true,
    Secure:   true,  // set to false in local dev if not using HTTPS
    SameSite: http.SameSiteStrictMode,
    MaxAge:   30 * 24 * 60 * 60, // 30 days
})
```

**Session token generation:**
```go
b := make([]byte, 32)
_, err := crypto_rand.Read(b)
token := hex.EncodeToString(b) // 64 hex chars
```

**SessionMiddleware:**
```go
func SessionMiddleware(pool *pgxpool.Pool) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            cookie, err := r.Cookie("runeplan_session")
            if err != nil {
                http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
                return
            }
            user, err := getUserBySession(r.Context(), pool, cookie.Value)
            if err != nil {
                http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
                return
            }
            ctx := context.WithValue(r.Context(), ctxKeyUser, user)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

**Session cleanup:** A goroutine in `main.go` runs every 24 hours:
```go
go func() {
    for range time.Tick(24 * time.Hour) {
        pool.Exec(context.Background(), "DELETE FROM sessions WHERE expires_at < NOW()")
    }
}()
```

### Discord OAuth

Discord OAuth is entirely optional. If `DISCORD_CLIENT_ID` is empty, both OAuth routes return `501 Not Implemented`.

**Redirect flow:**
1. Generate a random CSRF state token, store in a short-lived (10 min) cookie.
2. Redirect to `https://discord.com/api/oauth2/authorize` with `client_id`, `redirect_uri`, `response_type=code`, `scope=identify email`, and `state`.

**Callback flow:**
1. Validate `state` against the cookie.
2. Exchange `code` for access token via `POST https://discord.com/api/oauth2/token`.
3. Fetch user from `GET https://discord.com/api/users/@me`.
4. Upsert user row (match on email; if new, create with null password).
5. Create session, set cookie, redirect to `/`.

---

## Skill Ladder Query

`GET /api/skills` aggregates skill thresholds from all active goals and computes satisfaction against the user's current levels. This is the most complex query in the application.

```sql
SELECT
  gsr.skill,
  gsr.level,
  g.id   AS goal_id,
  g.name AS goal_name
FROM goal_skill_requirements gsr
JOIN goals g ON g.id = gsr.goal_id
WHERE g.user_id = $1
  AND g.is_completed = FALSE
ORDER BY gsr.skill, gsr.level ASC;
```

In Go, iterate the rows and build `SkillLadder` structs:

```go
type SkillLadder struct {
    Skill        string          `json:"skill"`
    CurrentLevel int             `json:"current_level"`
    Notes        string          `json:"notes"`
    Thresholds   []SkillThreshold `json:"thresholds"`
}

type SkillThreshold struct {
    Level     int           `json:"level"`
    Satisfied bool          `json:"satisfied"`
    Goals     []GoalSummary `json:"goals"`
}

// After querying:
for _, row := range rows {
    ladder := findOrCreate(&ladders, row.Skill)
    ladder.CurrentLevel = user.Skills[row.Skill] // from user JSONB
    ladder.Thresholds = append(ladder.Thresholds, SkillThreshold{
        Level:     row.Level,
        Satisfied: user.Skills[row.Skill] >= row.Level,
        Goals:     []GoalSummary{{ID: row.GoalID, Name: row.GoalName}},
    })
}
```

This query will always be fast — a user will never have more than a few dozen active goals, and there are only 23 skills. No pagination or caching needed.

---

## Requirement Deduplication

Non-skill requirements deduplicate via `canonical_key` and a partial unique index. When a user activates a pre-seeded goal, use this upsert pattern:

```sql
-- Upsert the requirement (reuses existing row if canonical_key already exists for this user)
INSERT INTO requirements (id, user_id, label, type, canonical_key, is_preseeded, quest_name, ...)
VALUES ($1, $2, $3, $4, $5, TRUE, $6, ...)
ON CONFLICT (user_id, canonical_key) WHERE canonical_key IS NOT NULL
DO NOTHING
RETURNING id;

-- Then link the (possibly pre-existing) requirement to the new goal:
INSERT INTO goal_requirements (goal_id, requirement_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;
```

`GET /api/requirements` returns each requirement once with `shared_by_goals` populated:

```sql
SELECT
  r.*,
  json_agg(json_build_object('id', g.id, 'name', g.name)) AS shared_by_goals
FROM requirements r
JOIN goal_requirements gr ON gr.requirement_id = r.id
JOIN goals g ON g.id = gr.goal_id
WHERE r.user_id = $1
  AND g.is_completed = FALSE
GROUP BY r.id
ORDER BY r.type, r.label;
```

---

## OSRS Hiscores Proxy

The backend proxies the Hiscores API to avoid CORS and to handle the CSV response format server-side.

**Endpoint:**
```
https://secure.runescape.com/m=hiscore_oldschool/index_lite.ws?player={rsn}
```

**Response format:** Plain text CSV, one skill per line: `rank,level,xp`. Skills appear in a fixed order. Lines beyond the first 23 are minigames/bosses and are ignored.

**Skill order (matches the Hiscores CSV exactly):**
```go
var HiscoresSkillOrder = []string{
    "Attack", "Defence", "Strength", "Hitpoints", "Ranged", "Prayer",
    "Magic", "Cooking", "Woodcutting", "Fletching", "Fishing", "Firemaking",
    "Crafting", "Smithing", "Mining", "Herblore", "Agility", "Thieving",
    "Slayer", "Farming", "Runecrafting", "Hunter", "Construction",
}
```

**Parsing:**
```go
lines := strings.Split(strings.TrimSpace(body), "\n")
skills := make(map[string]int)
for i, skillName := range HiscoresSkillOrder {
    if i >= len(lines) { break }
    parts := strings.Split(lines[i], ",")
    if len(parts) < 2 { continue }
    level, err := strconv.Atoi(parts[1])
    if err != nil || level < 1 { level = 1 } // -1 means not ranked
    skills[skillName] = level
}
```

**Rate limiting:** Return `429 Too Many Requests` if `last_hiscores_sync` is within 5 seconds of the current time.

---

## Conventions

- All HTTP handlers have the signature `func(pool *pgxpool.Pool) http.HandlerFunc`.
- JSON encode/decode uses `encoding/json` from stdlib — no third-party serializer.
- All database queries use `pgx` positional parameters (`$1`, `$2`, ...) — never string interpolation.
- Errors returned to the client are always `{ "error": "description" }` — never raw Go error strings.
- Context is threaded through all DB calls: `pool.QueryRow(r.Context(), ...)`.
- UUIDs generated with `github.com/google/uuid` v4: `uuid.New().String()`.
- Passwords hashed with `golang.org/x/crypto/bcrypt`, cost 12.
