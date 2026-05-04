# Aria CRM

Aria CRM is a **student-housing style CRM** with a **Go API** and **Next.js** web app. The standout feature is an **AI assistant** that answers questions by generating and running **read-only SQL** against your live Postgres database (leads, tasks, bookings, partners, properties, and related tables), streaming results over **SSE** chat.

## Repository layout

| Path | Purpose |
|------|---------|
| `aria/` | Go API (`cmd/api`), auth, AI query service, handlers, DB repos |
| `frontend/` | Next.js UI (login, dashboard, chat, admin) |
| `db/` | `init.sql` schema, `demo_agent_seed.sql` idempotent demo data |
| `schema_pipeline/` | Optional tooling to refresh schema / intent metadata for the AI |
| `docker-compose.yml` | Postgres (pgvector), Redis, API, optional frontend image |

## Prerequisites

- **Docker** and **Docker Compose** (recommended for API + DB + Redis)
- For **local frontend dev**: **Node.js** and **pnpm** (see `frontend/package.json`)
- **OpenAI** (or compatible) API key for the chat / text-to-SQL path

## Quick start (Docker)

1. **Copy environment file** from the template and edit secrets:

   ```bash
   cp .env.example .env
   ```

   Set at least `OPENAI_API_KEY`, Google OAuth values if you use Google login, and a strong `JWT_SECRET`. For local-only agent login without Google, you can set `ALLOW_DEMO_AUTH=true` (never enable in production).

2. **Start the stack**:

   ```bash
   docker compose up --build
   ```

   - API: `http://localhost:8080` (health: `GET /health`)
   - Postgres: `localhost:5432`
   - Redis: `localhost:6379`
   - Frontend container (if you use the `frontend` service): mapped per `docker-compose.yml` (e.g. port **3000**)

3. **Demo CRM data** (optional but recommended for the AI chat):

   ```bash
   docker compose exec -T postgres psql -U aria -d aria -f /db/demo_agent_seed.sql
   ```

   This attaches leads, tasks, bookings, and sample activities to the demo user id in `.env.example` (`DEMO_USER_ID`).

## Frontend (local development)

```bash
cd frontend
cp .env.example .env.local   # set NEXT_PUBLIC_API_URL etc. as documented there
pnpm install
pnpm dev
```

The dev server typically runs at `http://localhost:3000` with the API at `http://localhost:8080` unless you override URLs.

## Makefile targets

| Target | Description |
|--------|-------------|
| `make run` | `docker compose up` |
| `make build` | `docker compose build` |
| `make test` | Run Go tests inside the API container |
| `make lint` | `go vet` in the API container |
| `make schema-pipeline` | Run schema pipeline container (see compose file) |

The `make seed` target re-applies `init.sql` and `db/seed.sql`; the full bulk seed may be heavy or fragile. Prefer **`db/demo_agent_seed.sql`** for a small, repeatable dataset.

## Documentation

- **[Release notes](RELEASE_NOTES.md)** — version history and upgrade notes
- **[Requirements](requirements.md)** — product and technical requirements

## License and copyright

Copyright © 2026 Deonkar. All rights reserved.

This README is part of the Aria CRM project. Third-party dependencies remain under their respective licenses.
