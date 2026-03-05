# Frontend

The frontend is server-rendered HTML using [a-h/templ](https://templ.guide). Interactivity is handled by [HTMX 2.x](https://htmx.org) (server-driven partial updates) and [Alpine.js 3.x](https://alpinejs.dev) (client-side UI state). Styling uses [Tailwind CSS](https://tailwindcss.com) utility classes only.

There is no JavaScript build step, no bundler, and no Node.js in production. Static files (`htmx.min.js`, `alpine.min.js`, `app.css`) are committed to the repository and embedded in the Go binary via `go:embed`.

---

## Templ

### What it is

Templ compiles `.templ` files into Go functions. Each component is a typed Go function — no string templates, no runtime parsing.

```templ
// internal/goals/templates/goal_item.templ
package templates

import "github.com/johnfarrell/runeplan/internal/goals"

templ GoalItem(goal goals.Goal) {
    <li id={ "goal-" + goal.ID } class="flex items-center gap-2 p-2 rounded hover:bg-slate-700">
        <span class="flex-1 text-sm">{ goal.Name }</span>
        <span class="text-xs text-slate-400">{ goal.Category }</span>
        <button
            hx-delete={ "/htmx/goals/" + goal.ID }
            hx-target={ "#goal-" + goal.ID }
            hx-swap="outerHTML"
            class="text-red-400 hover:text-red-300 text-xs"
        >×</button>
    </li>
}
```

### Code generation

Run `templ generate` before `go build`. This produces `*_templ.go` files alongside each `.templ` file. Both files are committed.

```bash
# Generate all templates
templ generate

# Watch mode during development
templ generate --watch
```

### Rendering from handlers

Use `httputil.Render` to write a component to the response:

```go
// internal/httputil/render.go
func Render(w http.ResponseWriter, r *http.Request, status int, component templ.Component) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(status)
    component.Render(r.Context(), w)
}
```

Never use `templ.Handler` directly. Always use `httputil.Render` so status codes and content-type headers are set consistently.

### Layout

Full pages use the base layout which includes all script and style tags:

```templ
// internal/templates/layout/base.templ
package layout

templ Base(title string, content templ.Component) {
    <!DOCTYPE html>
    <html lang="en" class="h-full bg-slate-900 text-slate-100">
    <head>
        <meta charset="UTF-8"/>
        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
        <title>{ title } — RunePlan</title>
        <link rel="stylesheet" href="/static/app.css"/>
        <script src="/static/htmx.min.js"></script>
        <script defer src="/static/alpine.min.js"></script>
    </head>
    <body class="h-full" hx-boost="true">
        @nav()
        <main>
            @content
        </main>
    </body>
    </html>
}
```

HTMX fragment endpoints (returning partial HTML for swapping) do **not** use the base layout — they return only the component markup.

---

## HTMX

HTMX handles all data mutations and partial page updates without writing JavaScript.

### Core attributes used

| Attribute | Purpose |
|---|---|
| `hx-get="/path"` | Issue GET on trigger (default: click) |
| `hx-post="/path"` | Issue POST on trigger |
| `hx-patch="/path"` | Issue PATCH on trigger |
| `hx-delete="/path"` | Issue DELETE on trigger |
| `hx-target="#selector"` | Element to swap into |
| `hx-swap="outerHTML"` | Replace entire target element |
| `hx-swap="innerHTML"` | Replace inner content of target |
| `hx-trigger="change"` | Custom trigger (input change, etc.) |
| `hx-include="#form"` | Include another element's inputs in request |
| `hx-indicator="#spinner"` | Show element while request is in-flight |
| `hx-push-url="true"` | Update browser URL bar on navigation |

### Endpoint conventions

HTMX endpoints live under `/htmx/`. They return HTML fragments (no base layout wrapper). Full page routes live at top-level paths (`/`, `/planner`, `/browse`, `/profile`).

```
GET  /planner                     → Full HTML page (base layout + planner panels)
GET  /htmx/goals                  → <ul> of goal items (GoalSidebar fragment)
POST /htmx/goals                  → <li> of new goal (GoalItem fragment, swapped into list)
DELETE /htmx/goals/{id}           → Empty string (HTMX removes the target element)
GET  /htmx/requirements           → RequirementList fragment
PATCH /htmx/requirements/{id}     → Updated RequirementRow fragment
GET  /htmx/skills                 → SkillLadder list fragment
POST /htmx/user/sync              → Updated SkillGrid fragment after Hiscores sync
```

### Example: goal creation

```templ
// Template: form that posts and swaps the result into the list
templ AddGoalForm() {
    <form
        hx-post="/htmx/goals"
        hx-target="#goal-list"
        hx-swap="beforeend"
        class="flex gap-2"
    >
        <input type="text" name="name" placeholder="Goal name" class="input" required/>
        <button type="submit" class="btn-primary">Add</button>
    </form>
}
```

```go
// Handler: returns only the new <li> fragment
func Create(pool *pgxpool.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ... parse form, insert into db ...
        httputil.Render(w, r, http.StatusCreated, templates.GoalItem(goal))
    }
}
```

### `hx-boost`

`hx-boost="true"` on `<body>` upgrades all same-origin `<a>` clicks to HTMX fetches, swapping only `<main>` content and updating the browser URL. This gives SPA-like navigation without JavaScript routing.

### Out-of-band swaps

Use `hx-swap-oob="true"` on elements returned in a response when a single action needs to update multiple parts of the page (e.g. updating both the goal sidebar count and the requirement list after activating a catalog goal).

---

## Alpine.js

Alpine handles UI state that doesn't require a server round-trip.

### When to use Alpine vs HTMX

| Situation | Use |
|---|---|
| Toggle a modal open/closed | Alpine (`x-show`, `x-data`) |
| Fetch and display server data | HTMX |
| Form field validation feedback | Alpine |
| Submit a form and update DOM | HTMX |
| Accordion expand/collapse | Alpine |
| Skill ladder expand/collapse | Alpine (no server call needed) |

### Skill ladder expand/collapse

The skill ladder row toggles expanded state client-side — Alpine is the right tool because no data changes:

```templ
templ SkillLadderRow(ladder goals.SkillLadder) {
    <div x-data="{ open: false }" class={ ladderBorderColor(ladder) }>
        <button @click="open = !open" class="w-full flex items-center gap-2 p-2">
            <span>{ ladder.Skill }</span>
            <span>{ fmt.Sprintf("%d", ladder.CurrentLevel) }</span>
            <span x-show="!open">▶</span>
            <span x-show="open">▼</span>
        </button>
        <div x-show="open" x-collapse>
            for _, threshold := range ladder.Thresholds {
                @SkillThresholdRung(threshold, ladder.CurrentLevel)
            }
        </div>
    </div>
}
```

### Modal pattern

```templ
templ AddGoalModal() {
    <div x-data="{ open: false }">
        <button @click="open = true" class="btn-primary">+ Add Goal</button>

        <div x-show="open" x-cloak class="fixed inset-0 bg-black/50 flex items-center justify-center">
            <div @click.outside="open = false" class="bg-slate-800 rounded-lg p-6 w-full max-w-lg">
                <h2 class="text-lg font-semibold mb-4">Add Goal</h2>
                <!-- catalog browser loaded via HTMX when modal opens -->
                <div hx-get="/htmx/catalog/diaries" hx-trigger="revealed">
                    Loading...
                </div>
                <button @click="open = false" class="btn-secondary">Cancel</button>
            </div>
        </div>
    </div>
}
```

---

## Tailwind CSS

### Setup

Tailwind standalone CLI is used — no Node.js required.

```bash
# Development (watch mode)
tailwindcss --input static/app.css.src --output static/app.css --watch --content "./internal/**/*.templ"

# Production (minified)
tailwindcss --input static/app.css.src --output static/app.css --minify --content "./internal/**/*.templ"
```

The `tailwind.config.js` content paths point at `.templ` files:

```js
// tailwind.config.js
module.exports = {
  content: ["./internal/**/*.templ"],
  theme: { extend: {} },
  plugins: [],
}
```

The compiled `static/app.css` is committed to the repository. The Tailwind CLI is only needed when changing template markup — not for running the app.

### Conventions

- Tailwind utility classes only — no CSS modules, no custom component classes except in `app.css.src` where needed for base reset.
- No inline `style=` attributes except for dynamically computed values (e.g. progress bar width: `style={ "width: " + pct + "%" }`).

---

## Component Structure

```
internal/
  templates/
    layout/
      base.templ            # HTML shell, loads CSS + HTMX + Alpine
      nav.templ             # Top navigation, shows logged-in user
    components/
      error.templ           # Error message component
      planner.templ         # 3-panel planner page (wraps domain fragments)
      browse.templ          # Catalog browse page layout

  auth/templates/
    login.templ             # Login form page
    register.templ          # Registration form page

  goals/templates/
    goal_sidebar.templ      # Left panel: active goals list
    goal_item.templ         # Single goal row (HTMX swap target)
    skill_ladder_row.templ  # Expandable skill ladder (Alpine for expand)
    add_goal_modal.templ    # Alpine modal + HTMX catalog loader

  requirements/templates/
    requirement_list.templ  # Center panel: skill ladders + non-skill requirements
    requirement_row.templ   # Single requirement row (HTMX swap target)
    requirement_detail.templ # Right panel: selected item detail

  user/templates/
    profile.templ           # Profile page
    skill_grid.templ        # 23-skill grid with levels + Hiscores sync button
```

### SkillLadderRow spec

This is the most visually complex component.

**Collapsed (default):**
- Skill name + emoji icon
- Progress bar: `current_level` → highest threshold level
- Tick marks at each intermediate threshold
- Badge: `N/M goals` or `✓ all done`

**Expanded (Alpine `open = true`):**
- One rung per threshold, sorted ascending
- Satisfied rung: green filled circle + strikethrough level + green goal pill
- Unsatisfied rung: grey outline circle + `"N levels to go"` subtitle

**Left border:**
- Green: all thresholds satisfied
- Orange: any unsatisfied
- (Alpine: no click-based color change — color is server-rendered based on satisfaction state)

---

## Static Assets

```
static/
  htmx.min.js      # HTMX 2.x — download from https://unpkg.com/htmx.org@2.x
  alpine.min.js    # Alpine.js 3.x — download from https://unpkg.com/alpinejs@3.x
  app.css          # Compiled Tailwind output
```

All three files are committed to the repository and embedded in the Go binary:

```go
//go:embed static
var staticFiles embed.FS
```

When upgrading HTMX or Alpine, replace the file and regenerate `app.css` (only needed if new Tailwind classes are used).
