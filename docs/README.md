# RunePlan — Technical Documentation

> OSRS Goal Planner · Go · React 18 · TypeScript · PostgreSQL · Docker

This directory contains the full technical implementation brief for the RunePlan engineering team. Each document covers one concern area and can be read independently.

---

## Documents

| File | Contents |
|---|---|
| [architecture.md](./architecture.md) | System overview, stack decisions, repo structure, environment variables |
| [database.md](./database.md) | Full schema DDL, migration strategy, seed data, skills storage rationale |
| [api.md](./api.md) | Every endpoint — method, path, request body, response shape, JSON examples |
| [backend.md](./backend.md) | Go module, router wiring, auth, Hiscores proxy, deduplication logic |
| [frontend.md](./frontend.md) | TypeScript types, API layer, state management, component contracts, Nginx config |
| [deployment.md](./deployment.md) | Docker Compose, Dockerfiles, environment setup, self-hosting guide |
| [delivery.md](./delivery.md) | Phased task list, out-of-scope features, conventions and standards |

---

## Quick Start (Self-Hosting)

```bash
git clone https://github.com/your-org/runeplan
cd runeplan
cp .env.example .env        # fill in DB_PASSWORD
docker compose up -d
# App available at http://localhost:3000
```

Migrations and seed data run automatically on first backend startup. No manual database setup required.

---

## Stack Summary

| Layer | Technology | Key Libraries |
|---|---|---|
| Backend | Go 1.22+ | chi v5, pgx/v5, golang-migrate, google/uuid |
| Frontend | React 18 + TypeScript | Tailwind CSS, D3.js (graph view only), native fetch |
| Database | PostgreSQL 16 | Hosted in Docker; sessions in DB (no Redis) |
| Auth | Session cookies + optional Discord OAuth | HTTP-only, Secure, SameSite=Strict |
| Infrastructure | Docker Compose | 3 services: frontend (Nginx), backend, db |

---

## Design Mandate

**Simple over clever.** No analytics dashboards, no recommendation engine, no scope creep. Every decision should make the tool easier to maintain and self-host, not more impressive on paper. If a feature is not in [delivery.md](./delivery.md), raise it for discussion before writing any code.
