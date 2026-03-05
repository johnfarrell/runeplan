# Deployment

RunePlan is designed to be self-hosted with a single command. Everything needed to run the application is in the repository.

---

## Quick Start

```bash
git clone https://github.com/johnfarrell/runeplan
cd runeplan
cp .env.example .env       # edit DB_PASSWORD
docker compose up -d
```

The app is available at `http://localhost:8080`. The Go backend serves HTML, static assets, and all API endpoints directly — no Nginx proxy is needed.

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
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U runeplan"]
      interval: 5s
      timeout: 5s
      retries: 5

  backend:
    build: .
    environment:
      DATABASE_URL:          postgres://runeplan:${DB_PASSWORD}@db:5432/runeplan
      DISCORD_CLIENT_ID:     ${DISCORD_CLIENT_ID:-}
      DISCORD_CLIENT_SECRET: ${DISCORD_CLIENT_SECRET:-}
    depends_on:
      db:
        condition: service_healthy
    ports:
      - "8080:8080"
    restart: unless-stopped

volumes:
  pgdata:
```

There are only two services. The Go binary serves the entire application — HTML pages, HTMX fragments, static assets, and API endpoints — from port 8080.

The `db` service uses a named volume (`pgdata`) so data persists across `docker compose down` / `up` cycles. To fully wipe the database: `docker compose down -v`.

---

## Dockerfile

Multi-stage build:
1. `templ generate` compiles `.templ` files to `_templ.go`
2. `tailwindcss` compiles `static/app.css` from template sources
3. `go build` compiles the static binary with all assets embedded
4. Final stage: minimal Alpine image with only the binary

```dockerfile
# Dockerfile

# Stage 1: Generate templates + CSS + build binary
FROM golang:1.22-alpine AS build
WORKDIR /app

# Install templ and tailwindcss
RUN go install github.com/a-h/templ/cmd/templ@latest
RUN wget -q https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 \
    -O /usr/local/bin/tailwindcss && chmod +x /usr/local/bin/tailwindcss

# Download Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Generate templ files, compile CSS, build binary
RUN templ generate
RUN tailwindcss --input static/app.css.src --output static/app.css --minify --content "./internal/**/*.templ"
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Stage 2: Minimal runtime image
FROM alpine:3.19
WORKDIR /app
COPY --from=build /app/server .
EXPOSE 8080
CMD ["./server"]
```

`CGO_ENABLED=0` produces a fully static binary. Migrations are embedded in the binary (not copied separately). Static assets (`htmx.min.js`, `alpine.min.js`, `app.css`) are embedded via `go:embed` and do not need to be present in the runtime image. The final image is ~20MB.

---

## Development Workflow

```bash
# Terminal 1: Start the database
docker compose up db

# Terminal 2: Watch and recompile templates
templ generate --watch

# Terminal 3: Watch and recompile Tailwind CSS
tailwindcss --input static/app.css.src --output static/app.css --watch --content "./internal/**/*.templ"

# Terminal 4: Run the Go server (restart manually after code changes, or use air)
DATABASE_URL=postgres://runeplan:changeme@localhost:5432/runeplan go run ./cmd/server
```

Or use [air](https://github.com/air-verse/air) for live reload:

```bash
# Install air
go install github.com/air-verse/air@latest

# .air.toml: run templ generate before go build
air
```

---

## Deploying to a VPS

```bash
# On the server:
git clone https://github.com/johnfarrell/runeplan
cd runeplan
cp .env.example .env
nano .env                  # set real DB_PASSWORD
docker compose up -d
```

To expose the app on a real domain with HTTPS, place a reverse proxy in front of port 8080. Example Caddy config:

```
runeplan.example.com {
    reverse_proxy localhost:8080
}
```

Caddy handles TLS certificate provisioning automatically via Let's Encrypt.

---

## Data Backup

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
