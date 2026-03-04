# Frontend

Framework: React 18 + TypeScript. Tooling: Vite. Styling: Tailwind CSS utility classes only — no component library. State: `useReducer` + Context — no external state library. HTTP: native `fetch` — no Axios.

---

## TypeScript Configuration

```json
{
  "compilerOptions": {
    "strict": true,
    "target": "ES2020",
    "lib": ["ES2020", "DOM"],
    "module": "ESNext",
    "moduleResolution": "bundler",
    "jsx": "react-jsx",
    "baseUrl": "src"
  }
}
```

`strict: true` is non-negotiable. No `any`, no implicit returns, no unchecked index access.

---

## Type Definitions

All types live in `src/types/index.ts`. This file mirrors the Go structs exactly and is the single source of truth for type shapes on the frontend. Do not define ad-hoc types inline in components.

```typescript
// src/types/index.ts

export type GoalCategory =
  | 'achievement_diary'
  | 'quest'
  | 'skill_milestone'
  | 'boss_kill'
  | 'item_obtain'
  | 'custom';

export type DiaryTier = 'easy' | 'medium' | 'hard' | 'elite';

export type RequirementType = 'quest' | 'kill_count' | 'item_obtain' | 'freeform';

export interface GoalSummary {
  id: string;
  name: string;
}

export interface Goal {
  id: string;
  user_id: string;
  name: string;
  description: string;
  category: GoalCategory;
  diary_region?: string;
  diary_tier?: DiaryTier;
  is_preseeded: boolean;
  is_completed: boolean;
  sort_order: number;
  created_at: string;
}

export interface Requirement {
  id: string;
  label: string;
  type: RequirementType;
  is_preseeded: boolean;
  is_completed: boolean;
  notes: string;
  canonical_key?: string;
  quest_name?: string;
  boss_name?: string;
  kill_target?: number;
  kill_current?: number;
  item_name?: string;
  shared_by_goals: GoalSummary[];
}

export interface SkillThreshold {
  level: number;
  satisfied: boolean;
  goals: GoalSummary[];
}

export interface SkillLadder {
  skill: string;
  current_level: number;
  notes: string;
  thresholds: SkillThreshold[];
}

export interface User {
  id: string;
  rsn: string;
  skills: Record<string, number>;
  last_hiscores_sync?: string;
}

export interface CatalogGoal {
  id: string;
  name: string;
  description: string;
  category: GoalCategory;
  diary_region?: string;
  diary_tier?: DiaryTier;
  skill_thresholds: Array<{ skill: string; level: number }>;
  requirements: Array<{ label: string; type: RequirementType; canonical_key: string }>;
}
```

---

## API Layer

Each file in `src/api/` exports typed async functions. Always include `credentials: 'include'` so the session cookie is sent. Throw on non-OK responses so callers can handle errors uniformly.

```typescript
// src/api/client.ts — shared fetch wrapper
const BASE = '/api';

export async function apiFetch<T>(
  path: string,
  init?: RequestInit
): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    ...init,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error ?? 'Unknown error');
  }
  return res.json() as Promise<T>;
}
```

```typescript
// src/api/goals.ts
import { apiFetch } from './client';
import type { Goal } from '../types';

export const listGoals = () =>
  apiFetch<Goal[]>('/goals');

export const createGoal = (body: Partial<Goal> & { catalog_goal_id?: string }) =>
  apiFetch<Goal>('/goals', { method: 'POST', body: JSON.stringify(body) });

export const updateGoal = (id: string, patch: Partial<Goal>) =>
  apiFetch<Goal>(`/goals/${id}`, { method: 'PATCH', body: JSON.stringify(patch) });

export const deleteGoal = (id: string) =>
  apiFetch<void>(`/goals/${id}`, { method: 'DELETE' });
```

```typescript
// src/api/skills.ts
import { apiFetch } from './client';
import type { SkillLadder } from '../types';

export const listSkillLadders = () =>
  apiFetch<SkillLadder[]>('/skills');

export const addSkillThreshold = (goalId: string, skill: string, level: number) =>
  apiFetch<void>(`/goals/${goalId}/skills`, {
    method: 'POST',
    body: JSON.stringify({ skill, level }),
  });

export const removeSkillThreshold = (goalId: string, skill: string) =>
  apiFetch<void>(`/goals/${goalId}/skills/${skill}`, { method: 'DELETE' });
```

---

## State Management

Global state lives in a single `useReducer` in `src/context/AppContext.tsx`. The state shape mirrors the API responses directly. No derived state is stored — completion percentages, shared counts, and similar computations happen in components or pure utility functions.

```typescript
// src/context/AppContext.tsx

interface AppState {
  user: User | null;
  goals: Goal[];
  requirements: Requirement[];
  skillLadders: SkillLadder[];
  loading: boolean;
  error: string | null;
}

type Action =
  | { type: 'SET_USER';          payload: User }
  | { type: 'SET_GOALS';         payload: Goal[] }
  | { type: 'ADD_GOAL';          payload: Goal }
  | { type: 'UPDATE_GOAL';       payload: Goal }
  | { type: 'REMOVE_GOAL';       id: string }
  | { type: 'SET_REQUIREMENTS';  payload: Requirement[] }
  | { type: 'UPDATE_REQUIREMENT';payload: Requirement }
  | { type: 'SET_SKILL_LADDERS'; payload: SkillLadder[] }
  | { type: 'SET_LOADING';       payload: boolean }
  | { type: 'SET_ERROR';         payload: string | null };

function reducer(state: AppState, action: Action): AppState {
  switch (action.type) {
    case 'SET_GOALS':         return { ...state, goals: action.payload };
    case 'ADD_GOAL':          return { ...state, goals: [...state.goals, action.payload] };
    case 'UPDATE_GOAL':       return { ...state, goals: state.goals.map(g => g.id === action.payload.id ? action.payload : g) };
    case 'REMOVE_GOAL':       return { ...state, goals: state.goals.filter(g => g.id !== action.id) };
    case 'SET_REQUIREMENTS':  return { ...state, requirements: action.payload };
    case 'UPDATE_REQUIREMENT':return { ...state, requirements: state.requirements.map(r => r.id === action.payload.id ? action.payload : r) };
    case 'SET_SKILL_LADDERS': return { ...state, skillLadders: action.payload };
    // ... etc
    default: return state;
  }
}
```

---

## Component Structure

```
src/components/
  GoalSidebar.tsx        # Left panel: list of active goals with completion % bars
  RequirementList.tsx    # Center panel: skill ladders + non-skill requirements
  RequirementRow.tsx     # Single non-skill requirement row (checkbox, badge, notes)
  SkillLadderRow.tsx     # Single skill ladder row (see spec below)
  RequirementDetail.tsx  # Right panel: detail view for selected req or skill
  AddGoalModal.tsx       # Modal: catalog browser + custom goal creation
  AddRequirementModal.tsx# Modal: typed requirement creation form
  ProfilePanel.tsx       # Skill grid + Hiscores sync UI

src/pages/
  Planner.tsx            # 3-panel layout — composes sidebar, list, detail
  Browse.tsx             # Catalog browser
  Profile.tsx            # Player profile + sync
```

### `SkillLadderRow` component spec

This is the most behaviorally complex component in the application.

**Props:**
```typescript
interface SkillLadderRowProps {
  ladder: SkillLadder;
  onSelect: (skill: string) => void;
  selected: boolean;
}
```

**Collapsed state (default):**
- Skill name with an emoji icon (map defined in the component)
- A single progress bar from `current_level` to the highest threshold level
- Small tick marks on the bar at each intermediate threshold position
- A badge: `"N/M goals"` (satisfied thresholds / total thresholds) or `"✓ all done"` if all satisfied
- `current_level → highest_threshold` label

**Expanded state (click to toggle):**
- Vertical ladder rendered below the header row
- One rung per threshold, sorted ascending (thresholds arrive pre-sorted from the API)
- Each rung: circular node + level number + goal name pills
- Satisfied rungs: green filled node, strikethrough level number, green goal pill
- Unsatisfied rungs: grey outline node, `"N levels away"` subtitle
- No checkbox — completion is read-only, driven solely by `current_level`

**Left border color:**
- Green if all thresholds satisfied
- Orange if any unsatisfied
- Gold if this ladder is the currently selected one

**Important:** `SkillLadderRow` dispatches no state actions. It is purely presentational. Clicking the row calls `onSelect(ladder.skill)`, which the parent uses to open the detail panel.

---

## Utility Functions

Pure functions in `src/utils/` — no side effects, no imports from context.

```typescript
// src/utils/goals.ts

export function goalCompletionPct(
  goal: Goal,
  requirements: Requirement[],
  skillLadders: SkillLadder[]
): number {
  const reqs = requirements.filter(r =>
    r.shared_by_goals.some(g => g.id === goal.id)
  );
  const ladder = skillLadders.filter(l =>
    l.thresholds.some(t => t.goals.some(g => g.id === goal.id))
  );

  const totalThresholds = ladder.reduce((n, l) =>
    n + l.thresholds.filter(t => t.goals.some(g => g.id === goal.id)).length, 0
  );
  const satisfiedThresholds = ladder.reduce((n, l) =>
    n + l.thresholds.filter(t => t.satisfied && t.goals.some(g => g.id === goal.id)).length, 0
  );

  const total = reqs.length + totalThresholds;
  if (total === 0) return 0;
  const done = reqs.filter(r => r.is_completed).length + satisfiedThresholds;
  return Math.round((done / total) * 100);
}
```

---

## Nginx Configuration

```nginx
# frontend/nginx.conf
server {
  listen 80;

  # Serve the React SPA
  location / {
    root   /usr/share/nginx/html;
    index  index.html;
    try_files $uri $uri/ /index.html;  # SPA client-side routing fallback
  }

  # Proxy all API requests to the Go backend
  location /api/ {
    proxy_pass         http://backend:8080;
    proxy_set_header   Host $host;
    proxy_set_header   X-Real-IP $remote_addr;
    proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
  }

  location /health {
    proxy_pass http://backend:8080;
  }
}
```

Because Nginx proxies `/api/` to the backend, the frontend makes all requests to its own origin (`/api/...`). No CORS configuration is needed anywhere.

---

## Conventions

- `strict: true` in tsconfig — no `any`, no implicit returns.
- All API functions are in `src/api/` and return typed `Promise<T>`.
- Components accept typed props — no prop drilling through `unknown` or `Record<string, any>`.
- No inline styles except where Tailwind cannot express a dynamic value (e.g. a progress bar width computed from a percentage at runtime).
- Tailwind class names only — no CSS modules, no styled-components.
- All user-facing text is in English. No i18n infrastructure in v1.
