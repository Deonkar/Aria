# requirements.md
# Aria — CRM AI Assistant
### Natural language intelligence layer for CRM agents

---

## 1. Project Overview

Aria is a production-grade AI assistant embedded in a CRM system. Agents log in
with their Google account and chat with Aria in natural language. Aria translates
their questions into SQL, executes queries against the CRM database, and streams
grounded, accurate answers in real time. No agent needs SQL knowledge. No agent
ever sees data that is not theirs.

**Core value proposition:**
> An agent asks "Who are my most overdue leads this week?" — Aria queries the
> real database, returns real names, real contact dates, real statuses. Not a
> hallucination. A grounded answer from live data, streamed token by token.

**What this is NOT:**
- Not a chatbot answering from LLM memory
- Not a dashboard or BI tool
- Not a write system — read-only always
- Not a general assistant — scoped to CRM data only

---

## 2. Actors

| Actor | Description |
|---|---|
| **Agent** | Pre-sales agent. Owns assigned leads. Logs in via Google. Only sees their own data. |
| **Admin** | Full data access across all agents. Same UI, broader SQL scope. |
| **System** | Background jobs — schema pipeline, feedback clustering, cache eviction. |

---

## 3. Functional Requirements

### 3.1 Authentication — Google OAuth 2.0

- FR-AUTH-01: Users MUST sign in exclusively via Google OAuth 2.0. No passwords.
- FR-AUTH-02: On first login, system MUST create a `users` record for that Google ID if none exists.
- FR-AUTH-03: Backend MUST issue a signed JWT (HS256) after callback: `user_id`, `email`, `role`, `exp`.
- FR-AUTH-04: JWT expires in 8 hours. Refresh token stored in httpOnly cookie, valid 30 days.
- FR-AUTH-05: All API endpoints EXCEPT `/auth/google` and `/auth/callback` require valid JWT.
- FR-AUTH-06: Go middleware MUST extract `user_id` from JWT and attach to every request context.
- FR-AUTH-07: Logout MUST blacklist the JWT in Redis until its natural expiry (prevents reuse).
- FR-AUTH-08: Google OAuth scopes required: `openid`, `email`, `profile`.

### 3.2 CRM Data — Read-Only Access

- FR-CRM-01: DB MUST be seeded with: 10 agents, 500 leads, 200 bookings, 150 payments, 300 tasks, 50 properties, 20 partners, 1000 lead activities.
- FR-CRM-02: All seeded data MUST be relationally consistent (no orphaned foreign keys).
- FR-CRM-03: Application MUST connect via a PostgreSQL read-only role. No INSERT/UPDATE/DELETE at DB level.
- FR-CRM-04: Every AI-generated query MUST have `WHERE assigned_agent_id = $agent_id` unless role = admin.
- FR-CRM-05: Every SQL query MUST be validated as SELECT-only before execution. Non-SELECT = reject immediately.

### 3.3 AI Chat — Core Feature

- FR-AI-01: Agents type any natural language question about their CRM work.
- FR-AI-02: System converts question to SQL using GPT-4o tool calling (not prompt-stuffed SQL generation).
- FR-AI-03: Generated SQL is executed against PostgreSQL. Real rows returned.
- FR-AI-04: GPT-4o formats the SQL results into a clear, natural language answer.
- FR-AI-05: Answer MUST stream token-by-token via Server-Sent Events (SSE).
- FR-AI-06: If AI cannot generate valid SQL, it MUST explain why and suggest rephrasing.
- FR-AI-07: Every response MUST expose the generated SQL via a collapsible "View Query" panel.
- FR-AI-08: Zero SQL results → AI says "No results found" — does NOT fabricate data.
- FR-AI-09: SQL execution MUST timeout at 5 seconds. Timeout → graceful error to agent.
- FR-AI-10: If generated SQL fails, system MUST retry once with a self-correction prompt before erroring.

### 3.4 Conversation Memory

- FR-MEM-01: Each agent session maintains last 10 message pairs (question + answer) in Redis.
- FR-MEM-02: Redis key: `session:{user_id}:{session_id}`, TTL 2 hours.
- FR-MEM-03: Full conversation history is passed to GPT-4o on every call — enables follow-ups.
- FR-MEM-04: "New Chat" button clears session history and starts fresh.
- FR-MEM-05: All conversations and messages are persisted to PostgreSQL for audit + training.

### 3.5 Feedback Loop

- FR-FEED-01: Every AI response has thumbs-up / thumbs-down buttons.
- FR-FEED-02: Thumbs-up stores Q→SQL pair in `query_feedback` with `is_helpful = true`.
- FR-FEED-03: Thumbs-down prompts agent for correction note, stores with `is_helpful = false`.
- FR-FEED-04: Q→SQL pairs with 3+ upvotes are auto-promoted to `intent_examples`.
- FR-FEED-05: Nightly background job clusters negative feedback → flags to `intent_gaps` table.

### 3.6 Schema Intelligence Pipeline (Python)

- FR-SCHEMA-01: Python script introspects PostgreSQL schema: all tables, columns, types, FK relationships.
- FR-SCHEMA-02: GPT-4o generates business-friendly documentation for each table and column.
- FR-SCHEMA-03: Documentation is embedded via `text-embedding-3-small` → stored in `schema_embeddings` (pgvector).
- FR-SCHEMA-04: Pipeline is incremental: hash each table DDL, only re-process changed tables.
- FR-SCHEMA-05: Pipeline generates 20–30 example Q→SQL pairs per table group → stored in `intent_examples`.
- FR-SCHEMA-06: Run via: `python schema_pipeline/run.py`

### 3.7 Query Caching

- FR-CACHE-01: Before LLM call, check Redis: `query_cache:{user_id}:{sha256(normalised_question)}`.
- FR-CACHE-02: Cache TTL: 5 minutes. Cached hit → response in <100ms with `cached: true` flag shown.
- FR-CACHE-03: Thumbs-down on a response MUST invalidate its Redis cache entry.

### 3.8 Observability

- FR-OBS-01: Every AI query logged: question, SQL, execution_ms, row_count, tokens, cache_hit.
- FR-OBS-02: Structured JSON logs for: every HTTP request, LLM call, SQL execution, auth event.
- FR-OBS-03: `GET /admin/metrics` returns: queries today, avg response time, cache hit rate, top question types, feedback scores.

---

## 4. Non-Functional Requirements

| ID | Category | Requirement |
|---|---|---|
| NFR-01 | Performance | SSE first token within 2s for cache miss |
| NFR-02 | Performance | Cached response within 100ms |
| NFR-03 | Performance | SQL execution hard limit: 5 seconds |
| NFR-04 | Performance | API handles 100 concurrent agents without degradation |
| NFR-05 | Security | Read-only PostgreSQL role — app NEVER writes |
| NFR-06 | Security | All JWTs verified on every request |
| NFR-07 | Security | Agent data isolation enforced at Go layer, not trusted to LLM |
| NFR-08 | Security | OpenAI key never exposed to frontend |
| NFR-09 | Security | All SQL uses parameterised queries — no string concatenation |
| NFR-10 | Security | Rate limit: 30 AI queries per agent per minute (Redis) |
| NFR-11 | Reliability | OpenAI unavailable → graceful error, not 500 crash |
| NFR-12 | Reliability | Redis unavailable → degrade gracefully, skip cache, still answer |
| NFR-13 | DX | `docker compose up` starts everything, no manual steps |
| NFR-14 | DX | `make seed` populates all tables with realistic dummy data |
| NFR-15 | DX | `make schema-pipeline` runs Python doc + embedding generation |

---

## 5. Database Schema — All Flat Tables, Production Grade

> Rule: NO nested columns. NO JSONB for queryable data. Every field that an agent
> might ask about is a proper typed column. Foreign keys are explicit. Indexes are
> declared below each table.

---

### 5.1 teams
```sql
CREATE TABLE teams (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL,
    department   TEXT NOT NULL,  -- 'pre_sales','partnerships','supply','finance'
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

### 5.2 users
```sql
CREATE TABLE users (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    google_id           TEXT UNIQUE NOT NULL,
    email               TEXT UNIQUE NOT NULL,
    full_name           TEXT NOT NULL,
    avatar_url          TEXT,
    role                TEXT NOT NULL DEFAULT 'agent',  -- 'agent','admin'
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    department          TEXT,   -- 'pre_sales','partnerships','supply','finance'
    team_id             UUID REFERENCES teams(id),
    manager_id          UUID REFERENCES users(id),
    timezone            TEXT NOT NULL DEFAULT 'Asia/Kolkata',
    last_login_at       TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_users_google_id ON users(google_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_team_id ON users(team_id);
```

---

### 5.3 partners
```sql
CREATE TABLE partners (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             TEXT NOT NULL,
    partner_type     TEXT NOT NULL,  -- 'university','portal','agency','direct'
    country          TEXT,
    contact_email    TEXT,
    contact_phone    TEXT,
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    commission_rate  NUMERIC(5,2),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

### 5.4 properties
```sql
CREATE TABLE properties (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                      TEXT NOT NULL,
    city                      TEXT NOT NULL,
    country                   TEXT NOT NULL,
    address_line1             TEXT,
    address_line2             TEXT,
    postcode                  TEXT,
    property_type             TEXT,  -- 'student_hall','studio','shared_house'
    total_units               INT,
    available_units           INT,
    price_per_week            NUMERIC(10,2),
    currency                  TEXT NOT NULL DEFAULT 'GBP',
    university_proximity_km   NUMERIC(5,2),
    rating                    NUMERIC(3,2),
    is_active                 BOOLEAN NOT NULL DEFAULT TRUE,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_properties_city ON properties(city);
CREATE INDEX idx_properties_country ON properties(country);
```

---

### 5.5 leads
```sql
CREATE TABLE leads (
    id                           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id                  TEXT UNIQUE,
    assigned_agent_id            UUID REFERENCES users(id),
    partner_id                   UUID REFERENCES partners(id),
    first_name                   TEXT NOT NULL,
    last_name                    TEXT NOT NULL,
    email                        TEXT,
    phone                        TEXT,
    whatsapp_number              TEXT,
    alternate_phone              TEXT,
    alternate_email              TEXT,
    source_channel               TEXT,  -- 'partner','organic','paid','referral','direct'
    utm_source                   TEXT,
    utm_medium                   TEXT,
    utm_campaign                 TEXT,
    state                        TEXT NOT NULL DEFAULT 'new',
    -- 'new','contacted','interested','qualified','booked','lost','junk'
    priority                     TEXT NOT NULL DEFAULT 'medium',
    -- 'low','medium','high','urgent'
    source_country               TEXT,
    destination_country          TEXT,
    destination_city             TEXT,
    preferred_university         TEXT,
    preferred_room_type          TEXT,  -- 'studio','ensuite','shared'
    preferred_move_in_month      INT,   -- 1–12
    preferred_move_in_year       INT,
    budget_min                   NUMERIC(10,2),
    budget_max                   NUMERIC(10,2),
    budget_currency              TEXT DEFAULT 'GBP',
    duration_weeks               INT,
    is_phone_valid               BOOLEAN DEFAULT FALSE,
    is_ai_eligible               BOOLEAN DEFAULT FALSE,
    is_ai_qualified              BOOLEAN,
    ai_call_status               TEXT,
    -- 'pending','completed','ineligible','failed'
    ai_ineligible_reason         TEXT,
    is_bulk_upload               BOOLEAN DEFAULT FALSE,
    retry_count                  INT DEFAULT 0,
    next_retry_at                TIMESTAMPTZ,
    lost_reason                  TEXT,
    notification_sent_at         TIMESTAMPTZ,
    notification_delivery_status TEXT,  -- 'success','failure','pending'
    last_activity_at             TIMESTAMPTZ,
    assigned_at                  TIMESTAMPTZ,
    state_changed_at             TIMESTAMPTZ,
    created_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_leads_assigned_agent ON leads(assigned_agent_id);
CREATE INDEX idx_leads_state ON leads(state);
CREATE INDEX idx_leads_priority ON leads(priority);
CREATE INDEX idx_leads_partner ON leads(partner_id);
CREATE INDEX idx_leads_destination_country ON leads(destination_country);
CREATE INDEX idx_leads_last_activity ON leads(last_activity_at);
CREATE INDEX idx_leads_next_retry ON leads(next_retry_at);
CREATE INDEX idx_leads_created_at ON leads(created_at);
```

---

### 5.6 lead_activities
```sql
CREATE TABLE lead_activities (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id          UUID NOT NULL REFERENCES leads(id),
    agent_id         UUID REFERENCES users(id),
    activity_type    TEXT NOT NULL,
    -- 'call','email','whatsapp','note','state_change','assignment','sms'
    subject          TEXT,
    body             TEXT,
    direction        TEXT,        -- 'inbound','outbound'
    duration_seconds INT,
    outcome          TEXT,        -- 'connected','no_answer','voicemail','bounced'
    old_state        TEXT,
    new_state        TEXT,
    occurred_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_lead_activities_lead_id ON lead_activities(lead_id);
CREATE INDEX idx_lead_activities_agent_id ON lead_activities(agent_id);
CREATE INDEX idx_lead_activities_occurred_at ON lead_activities(occurred_at);
CREATE INDEX idx_lead_activities_type ON lead_activities(activity_type);
```

---

### 5.7 tasks
```sql
CREATE TABLE tasks (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id              UUID NOT NULL REFERENCES leads(id),
    assigned_agent_id    UUID NOT NULL REFERENCES users(id),
    created_by_agent_id  UUID REFERENCES users(id),
    task_type            TEXT NOT NULL,
    -- 'follow_up_call','send_email','send_whatsapp','schedule_viewing',
    -- 'send_proposal','other'
    title                TEXT NOT NULL,
    description          TEXT,
    priority             TEXT NOT NULL DEFAULT 'medium',
    status               TEXT NOT NULL DEFAULT 'pending',
    -- 'pending','in_progress','completed','cancelled','snoozed'
    due_date             DATE NOT NULL,
    due_time             TIME,
    completed_at         TIMESTAMPTZ,
    snoozed_until        TIMESTAMPTZ,
    reminder_sent_at     TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_tasks_assigned_agent ON tasks(assigned_agent_id);
CREATE INDEX idx_tasks_lead_id ON tasks(lead_id);
CREATE INDEX idx_tasks_due_date ON tasks(due_date);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_priority ON tasks(priority);
```

---

### 5.8 bookings
```sql
CREATE TABLE bookings (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id               UUID NOT NULL REFERENCES leads(id),
    property_id           UUID NOT NULL REFERENCES properties(id),
    agent_id              UUID NOT NULL REFERENCES users(id),
    external_booking_ref  TEXT UNIQUE,
    status                TEXT NOT NULL DEFAULT 'pending',
    -- 'pending','confirmed','cancelled','completed','refunded'
    room_type             TEXT,
    move_in_date          DATE,
    move_out_date         DATE,
    duration_weeks        INT,
    weekly_rent           NUMERIC(10,2) NOT NULL,
    total_rent            NUMERIC(12,2),
    currency              TEXT NOT NULL DEFAULT 'GBP',
    deposit_amount        NUMERIC(10,2),
    deposit_paid          BOOLEAN DEFAULT FALSE,
    commission_amount     NUMERIC(10,2),
    commission_paid       BOOLEAN DEFAULT FALSE,
    cancellation_reason   TEXT,
    cancelled_at          TIMESTAMPTZ,
    confirmed_at          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_bookings_lead_id ON bookings(lead_id);
CREATE INDEX idx_bookings_agent_id ON bookings(agent_id);
CREATE INDEX idx_bookings_property_id ON bookings(property_id);
CREATE INDEX idx_bookings_status ON bookings(status);
CREATE INDEX idx_bookings_move_in_date ON bookings(move_in_date);
```

---

### 5.9 payments
```sql
CREATE TABLE payments (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id              UUID NOT NULL REFERENCES bookings(id),
    lead_id                 UUID NOT NULL REFERENCES leads(id),
    agent_id                UUID REFERENCES users(id),
    payment_type            TEXT NOT NULL,
    -- 'deposit','first_installment','second_installment',
    -- 'full_payment','refund','commission'
    amount                  NUMERIC(12,2) NOT NULL,
    currency                TEXT NOT NULL DEFAULT 'GBP',
    amount_in_gbp           NUMERIC(12,2),
    status                  TEXT NOT NULL DEFAULT 'pending',
    -- 'pending','processing','completed','failed','refunded','disputed'
    payment_method          TEXT,  -- 'card','bank_transfer','paypal','stripe'
    gateway_transaction_id  TEXT UNIQUE,
    gateway_name            TEXT,
    paid_at                 TIMESTAMPTZ,
    due_date                DATE,
    refunded_at             TIMESTAMPTZ,
    refund_reason           TEXT,
    notes                   TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_payments_booking_id ON payments(booking_id);
CREATE INDEX idx_payments_lead_id ON payments(lead_id);
CREATE INDEX idx_payments_agent_id ON payments(agent_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_paid_at ON payments(paid_at);
```

---

### 5.10 conversations
```sql
CREATE TABLE conversations (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id),
    session_token    TEXT UNIQUE NOT NULL,
    title            TEXT,
    message_count    INT DEFAULT 0,
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_message_at  TIMESTAMPTZ,
    ended_at         TIMESTAMPTZ
);
CREATE INDEX idx_conversations_user_id ON conversations(user_id);
CREATE INDEX idx_conversations_started_at ON conversations(started_at);
```

---

### 5.11 messages
```sql
CREATE TABLE messages (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id      UUID NOT NULL REFERENCES conversations(id),
    user_id              UUID NOT NULL REFERENCES users(id),
    role                 TEXT NOT NULL,         -- 'user','assistant'
    content              TEXT NOT NULL,
    generated_sql        TEXT,
    sql_row_count        INT,
    sql_execution_ms     INT,
    token_count_input    INT,
    token_count_output   INT,
    model_used           TEXT,
    was_cached           BOOLEAN DEFAULT FALSE,
    was_sql_corrected    BOOLEAN DEFAULT FALSE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_user_id ON messages(user_id);
CREATE INDEX idx_messages_created_at ON messages(created_at);
```

---

### 5.12 query_feedback
```sql
CREATE TABLE query_feedback (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id       UUID UNIQUE NOT NULL REFERENCES messages(id),
    user_id          UUID NOT NULL REFERENCES users(id),
    is_helpful       BOOLEAN NOT NULL,
    correction_note  TEXT,
    corrected_sql    TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_query_feedback_is_helpful ON query_feedback(is_helpful);
```

---

### 5.13 intent_examples
```sql
CREATE EXTENSION IF NOT EXISTS vector;
CREATE TABLE intent_examples (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_text       TEXT NOT NULL,
    question_embedding  vector(1536),
    sql_template        TEXT NOT NULL,
    tables_used         TEXT[],
    intent_category     TEXT,
    source              TEXT NOT NULL,
    -- 'auto_generated','promoted_feedback','manually_added'
    upvote_count        INT DEFAULT 0,
    is_active           BOOLEAN DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_intent_examples_embedding
    ON intent_examples USING hnsw (question_embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
CREATE INDEX idx_intent_examples_category ON intent_examples(intent_category);
```

---

### 5.14 schema_embeddings
```sql
CREATE TABLE schema_embeddings (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_name        TEXT NOT NULL,
    column_name       TEXT,
    description_text  TEXT NOT NULL,
    embedding         vector(1536),
    ddl_hash          TEXT NOT NULL,
    generated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_schema_embeddings_embedding
    ON schema_embeddings USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
CREATE INDEX idx_schema_embeddings_table ON schema_embeddings(table_name);
```

---

### 5.15 intent_gaps
```sql
CREATE TABLE intent_gaps (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_cluster  TEXT NOT NULL,
    example_questions TEXT[],
    failure_count     INT DEFAULT 1,
    is_resolved       BOOLEAN DEFAULT FALSE,
    resolution_note   TEXT,
    first_seen_at     TIMESTAMPTZ DEFAULT NOW(),
    last_seen_at      TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 6. Seed Data Volumes

| Table | Count | Notes |
|---|---|---|
| teams | 4 | UK Pre-Sales, Canada Pre-Sales, Partnerships, Supply |
| users | 10 | 8 agents (2/team), 2 admins |
| partners | 20 | 15 universities, 5 portals |
| properties | 50 | London(20), Manchester(10), Toronto(10), Vancouver(10) |
| leads | 500 | ~50/agent, realistic state/priority mix |
| lead_activities | 1000 | Calls, emails, notes — 2/lead avg |
| tasks | 300 | Mix pending/overdue/completed |
| bookings | 200 | ~40% leads have booking |
| payments | 150 | ~75% bookings have ≥1 payment |
| intent_examples | 30 | Pre-seeded by schema pipeline |
