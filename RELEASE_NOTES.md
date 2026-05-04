# Aria CRM — Release notes

## Version 0.2.0 — 4 May 2026

### Summary

This release adds a **Next.js CRM frontend** (login, dashboard, AI chat with SSE, admin surface), **demo JWT login** for local agent testing, an **idempotent demo database seed** so the AI has realistic leads, tasks, bookings, and activities to query, plus configuration and documentation updates. Earlier milestones on this line include the **Go API** with AI-assisted SQL, streaming chat, conversations, feedback, and admin endpoints.

### Frontend

- Next.js app with dashboard layout, theme support, and shared UI components.
- Chat experience wired to the Aria API with streaming responses and conversation context.
- Login flow including demo-token path for rapid local evaluation.
- Admin dashboard page scaffold for operational views.

### API & backend (`aria/`)

- Demo token authentication path (non-production) alongside Google OAuth.
- Config extensions for demo and related environment variables.
- AI query pipeline: text-to-SQL, validation, execution against live Postgres, SSE streaming.
- Handlers for chat, conversations, feedback, and admin utilities.

### Database & operations

- **`db/demo_agent_seed.sql`**: safe-to-re-run seed for the standard demo agent UUID — partners, property, leads, tasks, bookings, and sample `lead_activities`.
- **`.env.example`**: documents variables needed for API, OAuth, Redis, Postgres, and demo mode.

### Requirements & docs

- **`requirements.md`**: expanded product and technical requirements reference.

### Upgrade notes

1. Apply schema from `db/init.sql` on fresh databases (or your existing migration process).
2. Optional: run `db/demo_agent_seed.sql` against Postgres so demo chat answers return non-empty CRM data (see script header for the agent UUID).
3. Configure root `.env` and `frontend/.env` from the respective `.env.example` files.

### Acknowledgements

Built for **Aria CRM** — live SQL-backed answers over your CRM schema.

---

## Legal

**Copyright © 2026 Deonkar. All rights reserved.**

This release notes document and the Aria CRM software described herein are the property of the copyright holder unless otherwise stated. Unauthorized reproduction or distribution may be prohibited by law.

Third-party libraries and frameworks used in this project remain subject to their respective licenses.
