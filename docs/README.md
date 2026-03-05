# RunePlan — Technical Documentation

> OSRS Goal Planner · Go · Templ · HTMX · Alpine.js · PostgreSQL · Docker

This directory contains the full technical implementation brief for the RunePlan engineering team. Each document covers one concern area and can be read independently.

---

## Documents

| File | Contents |
|---|---|
| [architecture.md](./architecture.md) | System overview, stack decisions, DDD repo structure, environment variables |
| [database.md](./database.md) | Full schema DDL, migration strategy, seed data, skills storage rationale |
| [api.md](./api.md) | Route reference — page routes, HTMX fragment endpoints, JSON API endpoints |
| [backend.md](./backend.md) | Go module, router wiring, auth, response helpers, Hiscores proxy, deduplication |
| [frontend.md](./frontend.md) | Templ components, HTMX patterns, Alpine.js usage, Tailwind CSS setup |
| [deployment.md](./deployment.md) | Docker Compose, Dockerfile, dev workflow, self-hosting guide |
| [delivery.md](./delivery.md) | Phased task list, out-of-scope features, conventions and standards |

---

## Quick Start (Self-Hosting)

```bash
git clone https://github.com/johnfarrell/runeplan
cd runeplan
cp .env.example .env        # fill in DB_PASSWORD
docker compose up -d
# App available at http://localhost:8080
```

Migrations and seed data run automatically on first backend startup. No manual database setup required.

---

## Stack Summary

| Layer | Technology | Details |
|---|---|---|
| Backend | Go 1.22+ | chi v5 router · pgx/v5 · golang-migrate · google/uuid |
| Templates | a-h/templ | Compiled to Go — type-safe SSR, no runtime parsing |
| Interactivity | HTMX 2.x + Alpine.js 3.x | HTMX for server updates · Alpine for client-side UI state |
| Styling | Tailwind CSS 3.x | Standalone CLI · compiled `app.css` committed to repo |
| Database | PostgreSQL 16 | Sessions in DB (no Redis) · migrations on startup |
| Auth | Session cookies + optional Discord OAuth | HTTP-only · Secure · SameSite=Strict · 30-day sessions |
| Infrastructure | Docker Compose | 2 services: backend + db · Go binary serves entire app |

---

## Design Mandate

**Simple over clever.** No analytics dashboards, no recommendation engine, no scope creep. Every decision should make the tool easier to maintain and self-host, not more impressive on paper. If a feature is not in [delivery.md](./delivery.md), raise it for discussion before writing any code.
