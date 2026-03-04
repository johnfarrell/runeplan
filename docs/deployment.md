# Deployment

RunePlan is designed to be self-hosted with a single command. Everything needed to run the application is in the repository.

---

## Quick Start

```bash
git clone https://github.com/your-org/runeplan
cd runeplan
cp .env.example .env       # edit DB_PASSWORD and SESSION_SECRET
docker compose up -d
```

The app is available at `http://localhost:3000`. The backend API is at `http://localhost:8080` (also proxied via Nginx at `http://localhost:3000/api/`).

Migrations and seed data run automatically on first backend startup. No manual database initialization is required.

---

## docker-compose.yml

```yaml
services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB:       runeplan
      POSTGRES_USER:     runeplan
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U runeplan"]
      interval: 5s
      timeout: 5s
      retries: 5

  backend:
    build: ./backend
    environment:
      DATABASE_URL:          postgres://runeplan:${DB_PASSWORD}@db:5432/runeplan
      SESSION_SECRET:        ${SESSION_SECRET}
      DISCORD_CLIENT_ID:     ${DISCORD_CLIENT_ID:-}
      DISCORD_CLIENT_SECRET: ${DISCORD_CLIENT_SECRET:-}
    depends_on:
      db:
        condition: service_healthy
    ports:
      - "8080:8080"
    restart: unless-stopped

  frontend:
    build: ./frontend
    ports:
      - "3000:80"
    depends_on:
      - backend
    restart: unless-stopped

volumes:
  pgdata:
```

The `db` service uses a named volume (`pgdata`) so data persists across `docker compose down` / `up` cycles. To fully wipe the database: `docker compose down -v`.

The `backend` waits for the `db` health check to pass before starting, which guarantees Postgres is accepting connections before migrations run.

---

## Backend Dockerfile

Multi-stage build: the first stage compiles the Go binary, the second stage copies only the binary and migrations into a minimal Alpine image.

```dockerfile
# backend/Dockerfile

FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM alpine:3.19
WORKDIR /app
COPY --from=build /app/server .
COPY --from=build /app/migrations ./migrations
EXPOSE 8080
CMD ["./server"]
```

`CGO_ENABLED=0` produces a fully static binary that runs in Alpine without glibc. The final image is ~20MB.

---

## Frontend Dockerfile

Multi-stage build: the first stage runs `npm run build` (Vite), the second stage serves the static output with Nginx.

```dockerfile
# frontend/Dockerfile

FROM node:20-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build           # output in /app/dist

FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

The final image is the stock Nginx Alpine image (~10MB) plus the compiled static assets. There is no Node.js runtime in the production image.

---

## Deploying to a VPS

The same `docker-compose.yml` works on any server with Docker installed.

```bash
# On the server:
git clone https://github.com/your-org/runeplan
cd runeplan
cp .env.example .env
nano .env                  # set real values
docker compose up -d
```

To put the app on a real domain with HTTPS, place a reverse proxy (Nginx or Caddy) in front of the `frontend` container on port 3000. Example Caddy config:

```
runeplan.example.com {
    reverse_proxy localhost:3000
}
```

Caddy handles TLS certificate provisioning automatically via Let's Encrypt.

---

## Deploying to Fly.io or Railway

Both platforms support Docker Compose or individual Dockerfiles. The recommended approach is to deploy the backend and database on Fly.io/Railway and host the frontend on any static hosting (Vercel, Cloudflare Pages, Netlify) — update the Nginx proxy target accordingly.

For fully managed self-hosting, the single-VPS approach above is simpler and keeps everything together.

---

## Data Backup

The database volume can be backed up with:

```bash
docker exec runeplan-db-1 pg_dump -U runeplan runeplan > backup.sql
```

To restore:

```bash
docker exec -i runeplan-db-1 psql -U runeplan runeplan < backup.sql
```

Users can also export their own data via `GET /api/user/export`, which returns a full JSON dump of their goals, requirements, and skill levels.

---

## Updating

```bash
git pull
docker compose build
docker compose up -d
```

New migrations run automatically on backend startup. Existing data is preserved.
