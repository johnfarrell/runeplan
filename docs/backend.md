# Backend

Language: Go 1.22+. Stdlib-first — reach for a library only when the stdlib is genuinely insufficient.

---

## Module & Dependencies

```
// go.mod
module github.com/johnfarrell/runeplan

go 1.22

require (
  github.com/a-h/templ                  v0.3.0   // Type-safe HTML templates
  github.com/go-chi/chi/v5              v5.1.0   // HTTP router
  github.com/jackc/pgx/v5               v5.6.0   // PostgreSQL driver + pgxpool
  github.com/golang-migrate/migrate/v4  v4.17.1  // Database migrations
  github.com/google/uuid                v1.6.0   // UUID generation
  golang.org/x/crypto                   v0.22.0  // bcrypt
)
```

No ORMs. No HTTP client libraries. No config parsing libraries. All queries are hand-written SQL with `pgx` positional parameters.

---

## Entrypoint

`cmd/server/main.go` is responsible for four things in order: load environment, initialize the database pool, run pending migrations, start the HTTP server.

```go
func main() {
    pool := db.NewPool(os.Getenv("DATABASE_URL"))
    defer pool.Close()
    db.RunMigrations(pool)

    shutdownCtx, cancel := context.WithCancel(context.Background())
    defer cancel()

    r := buildRouter(pool)
    startSessionCleanup(pool, shutdownCtx)

    srv := &http.Server{Addr: ":8080", Handler: r}

    go func() {
        quit := make(chan os.Signal, 1)
        signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
        <-quit
        cancel()
        ctx, done := context.WithTimeout(context.Background(), 10*time.Second)
        defer done()
        srv.Shutdown(ctx)
    }()

    log.Printf("listening on :8080")
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatal(err)
    }
}
```

---

## Router

All routes are registered in a `buildRouter` function. The router serves three kinds of responses:

1. **Full HTML pages** — `GET /planner`, `GET /browse`, `GET /profile` — rendered with the base layout
2. **HTMX fragments** — `GET|POST|PATCH|DELETE /htmx/*` — return partial HTML for HTMX swapping
3. **JSON data endpoints** — `GET /api/*` — catalog data, export, health check

```go
func buildRouter(pool *pgxpool.Pool) http.Handler {
    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)

    // Static assets (embedded)
    r.Handle("/static/*", http.FileServer(http.FS(staticFiles)))

    // Health check
    r.Get("/health", healthHandler)

    // Auth pages + form handlers (public)
    r.Get("/login",                     auth.LoginPage(pool))
    r.Post("/auth/register",            auth.Register(pool))
    r.Post("/auth/login",               auth.Login(pool))
    r.Post("/auth/logout",              auth.Logout(pool))
    r.Get("/auth/discord",              auth.DiscordRedirect)
    r.Get("/auth/discord/callback",     auth.DiscordCallback(pool))

    // Catalog JSON API (public — no auth required)
    r.Get("/api/catalog/diaries",       catalog.ListDiaries(pool))
    r.Get("/api/catalog/quests",        catalog.ListQuests(pool))
    r.Get("/api/catalog/skills",        catalog.ListSkills)

    // Authenticated routes
    r.Group(func(r chi.Router) {
        r.Use(auth.SessionMiddleware(pool))

        // Full page routes
        r.Get("/",          goals.PlannerPage(pool))
        r.Get("/planner",   goals.PlannerPage(pool))
        r.Get("/browse",    catalog.BrowsePage(pool))
        r.Get("/profile",   user.ProfilePage(pool))

        // HTMX fragment routes — goals
        r.Get("/htmx/goals",                        goals.ListFragment(pool))
        r.Post("/htmx/goals",                       goals.CreateFragment(pool))
        r.Patch("/htmx/goals/{id}",                 goals.UpdateFragment(pool))
        r.Delete("/htmx/goals/{id}",                goals.DeleteFragment(pool))

        // HTMX fragment routes — skill thresholds
        r.Get("/htmx/skills",                               goals.SkillLaddersFragment(pool))
        r.Post("/htmx/goals/{id}/skills",                   goals.AddSkillFragment(pool))
        r.Delete("/htmx/goals/{id}/skills/{skill}",         goals.RemoveSkillFragment(pool))

        // HTMX fragment routes — requirements
        r.Get("/htmx/requirements",                         reqs.ListFragment(pool))
        r.Post("/htmx/requirements",                        reqs.CreateFragment(pool))
        r.Patch("/htmx/requirements/{id}",                  reqs.UpdateFragment(pool))
        r.Delete("/htmx/requirements/{id}",                 reqs.DeleteFragment(pool))
        r.Post("/htmx/goals/{id}/requirements/{req_id}",    reqs.LinkFragment(pool))
        r.Delete("/htmx/goals/{id}/requirements/{req_id}",  reqs.UnlinkFragment(pool))

        // HTMX fragment routes — user / profile
        r.Get("/htmx/user/me",      user.ProfileFragment(pool))
        r.Patch("/htmx/user/me",    user.UpdateFragment(pool))
        r.Post("/htmx/user/sync",   user.SyncHiscoresFragment(pool))

        // JSON data endpoints
        r.Get("/api/user/export",   user.Export(pool))
    })

    return r
}
```

---

## Response Helpers

All response writing goes through `internal/httputil`.

### HTML responses — `httputil.Render`

```go
// internal/httputil/render.go
func Render(w http.ResponseWriter, r *http.Request, status int, component templ.Component) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(status)
    if err := component.Render(r.Context(), w); err != nil {
        log.Printf("render error: %v", err)
    }
}
```

Use `httputil.Render` for every HTML response — both full pages and HTMX fragments:

```go
// Full page
httputil.Render(w, r, http.StatusOK, layout.Base("Planner", planner.Page(goals, ladders, reqs)))

// HTMX fragment
httputil.Render(w, r, http.StatusOK, goalTemplates.GoalItem(goal))
```

### JSON responses — `httputil.WriteJSON`

```go
// internal/httputil/json.go
func WriteJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(v); err != nil {
        log.Printf("json encode error: %v", err)
    }
}
```

Use `httputil.WriteJSON` only for the catalog, export, and health endpoints. All interactive endpoints return HTML.

### Error responses

For HTML routes, render the shared Error component:

```go
httputil.Render(w, r, http.StatusUnauthorized, components.Error("You must be logged in."))
```

For JSON-only routes (`/api/*`), write a JSON error:

```go
httputil.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid input"})
```

---

## Authentication

### Session flow

**Register:** Hash password with bcrypt (cost 12), insert user, create session row, set cookie, redirect to `/planner`.

**Login:** Fetch user by email, compare bcrypt hash, create session row, set cookie, redirect to `/planner`.

**Logout:** Delete session row, clear cookie, redirect to `/login`.

**Cookie settings:**
```go
http.SetCookie(w, &http.Cookie{
    Name:     "runeplan_session",
    Value:    sessionToken,
    Path:     "/",
    HttpOnly: true,
    Secure:   true,  // set false in local dev without HTTPS
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

**Context key type:** Use an unexported integer type to avoid collisions:
```go
// internal/auth/model.go
type contextKey int
const ctxKeyUser contextKey = iota
```

**SessionMiddleware:**
```go
func SessionMiddleware(pool *pgxpool.Pool) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            cookie, err := r.Cookie("runeplan_session")
            if err != nil {
                http.Redirect(w, r, "/login", http.StatusSeeOther)
                return
            }
            user, err := getUserBySession(r.Context(), pool, cookie.Value)
            if err != nil {
                http.Redirect(w, r, "/login", http.StatusSeeOther)
                return
            }
            ctx := context.WithValue(r.Context(), ctxKeyUser, user)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

Note: for HTMX requests (detected via `HX-Request: true` header), return a `HX-Redirect` header instead of a 302 so HTMX handles the redirect:
```go
if r.Header.Get("HX-Request") == "true" {
    w.Header().Set("HX-Redirect", "/login")
    w.WriteHeader(http.StatusUnauthorized)
    return
}
http.Redirect(w, r, "/login", http.StatusSeeOther)
```

**Get user from context:**
```go
// internal/auth/model.go
func GetUser(ctx context.Context) *User {
    u, _ := ctx.Value(ctxKeyUser).(*User)
    return u
}
```

**Session cleanup:**
```go
func startSessionCleanup(pool *pgxpool.Pool, ctx context.Context) {
    go func() {
        ticker := time.NewTicker(24 * time.Hour)
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                pool.Exec(ctx, "DELETE FROM sessions WHERE expires_at < NOW()")
            case <-ctx.Done():
                return
            }
        }
    }()
}
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
5. Create session, set cookie, redirect to `/planner`.

---

## User Profile

### Skill Level Update (`PATCH /htmx/user/me`)

`skills` is a JSONB column. Merge incoming skill levels atomically using PostgreSQL's `||` operator:

```sql
UPDATE users
SET skills = skills || $1::jsonb
WHERE id = $2
RETURNING *;
```

Pass only the skills to update; unmentioned skills are preserved.

---

## Skill Ladder Query

`GET /htmx/skills` aggregates skill thresholds from all active goals and computes satisfaction against the user's current levels.

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

In Go, build `SkillLadder` structs from the rows and pass them to the template:

```go
type SkillLadder struct {
    Skill        string           `json:"skill"`
    CurrentLevel int              `json:"current_level"`
    Thresholds   []SkillThreshold `json:"thresholds"`
}

type SkillThreshold struct {
    Level     int           `json:"level"`
    Satisfied bool          `json:"satisfied"`
    Goals     []GoalSummary `json:"goals"`
}
```

---

## Requirement Deduplication

Non-skill requirements deduplicate via `canonical_key` and a partial unique index. When a user activates a pre-seeded goal:

```sql
INSERT INTO requirements (id, user_id, label, type, canonical_key, is_preseeded, quest_name, ...)
VALUES ($1, $2, $3, $4, $5, TRUE, $6, ...)
ON CONFLICT (user_id, canonical_key) WHERE canonical_key IS NOT NULL
DO NOTHING
RETURNING id;

INSERT INTO goal_requirements (goal_id, requirement_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;
```

---

## OSRS Hiscores Proxy

The backend proxies the Hiscores API to avoid CORS and to handle the CSV response format server-side.

**Endpoint:**
```
https://secure.runescape.com/m=hiscore_oldschool/index_lite.ws?player={rsn}
```

**Skill order (matches the Hiscores CSV exactly):**
```go
var HiscoresSkillOrder = []string{
    "Attack", "Defence", "Strength", "Hitpoints", "Ranged", "Prayer",
    "Magic", "Cooking", "Woodcutting", "Fletching", "Fishing", "Firemaking",
    "Crafting", "Smithing", "Mining", "Herblore", "Agility", "Thieving",
    "Slayer", "Farming", "Runecrafting", "Hunter", "Construction",
}
```

**HTTP client:** Use a package-level client with a timeout:
```go
var hiscoresClient = &http.Client{Timeout: 10 * time.Second}
```

**Parsing:** Skip the Overall row at index 0:
```go
lines := strings.Split(strings.TrimSpace(body), "\n")
if len(lines) > 0 {
    lines = lines[1:] // skip "Overall" row
}
skills := make(map[string]int)
for i, skillName := range HiscoresSkillOrder {
    if i >= len(lines) { break }
    parts := strings.Split(lines[i], ",")
    if len(parts) < 2 { continue }
    level, err := strconv.Atoi(parts[1])
    if err != nil || level < 1 { level = 1 }
    skills[skillName] = level
}
```

**Rate limiting:** Return `429 Too Many Requests` if `last_hiscores_sync` is within 5 seconds.

After a successful sync, render the updated `SkillGrid` component and return it as an HTMX fragment.

---

## Conventions

- All HTTP handlers have the signature `func Name(pool *pgxpool.Pool) http.HandlerFunc`.
- HTML responses: `httputil.Render(w, r, status, component)` — never `templ.Handler` directly.
- JSON responses: `httputil.WriteJSON(w, status, v)` — only for `/api/*` endpoints.
- HTML errors: `httputil.Render(w, r, status, components.Error("msg"))` — for interactive routes.
- JSON errors: `httputil.WriteJSON(w, status, map[string]string{"error": "..."})` — for `/api/*` only.
- Context user: `auth.GetUser(ctx)` — unexported contextKey type.
- All DB queries use pgx positional parameters (`$1`, `$2`, ...) — never string interpolation.
- Context is threaded through all DB calls: `pool.QueryRow(r.Context(), ...)`.
- UUIDs generated with `github.com/google/uuid` v4: `uuid.New().String()`.
- Passwords hashed with `golang.org/x/crypto/bcrypt`, cost 12.
- `HiscoresSkillOrder` is defined in `catalog/catalog.go` (or `catalog/model.go`).
- Run `templ generate` before `go build`. Generated `_templ.go` files are committed.
