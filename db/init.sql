-- Phase 0: Database bootstrap (schema only)
-- This file is executed by the postgres docker image on first startup.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS vector;

-- Read-only role for app query execution (defence in depth)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'aria_readonly') THEN
    CREATE ROLE aria_readonly LOGIN PASSWORD 'aria_ro';
  END IF;
END
$$;

GRANT CONNECT ON DATABASE aria TO aria_readonly;
GRANT USAGE ON SCHEMA public TO aria_readonly;

-- 5.1 teams
CREATE TABLE IF NOT EXISTS teams (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL,
    department   TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5.2 users
CREATE TABLE IF NOT EXISTS users (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    google_id           TEXT UNIQUE NOT NULL,
    email               TEXT UNIQUE NOT NULL,
    full_name           TEXT NOT NULL,
    avatar_url          TEXT,
    role                TEXT NOT NULL DEFAULT 'agent',
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    department          TEXT,
    team_id             UUID REFERENCES teams(id),
    manager_id          UUID REFERENCES users(id),
    timezone            TEXT NOT NULL DEFAULT 'Asia/Kolkata',
    last_login_at       TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_team_id ON users(team_id);

-- 5.3 partners
CREATE TABLE IF NOT EXISTS partners (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             TEXT NOT NULL,
    partner_type     TEXT NOT NULL,
    country          TEXT,
    contact_email    TEXT,
    contact_phone    TEXT,
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    commission_rate  NUMERIC(5,2),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5.4 properties
CREATE TABLE IF NOT EXISTS properties (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                      TEXT NOT NULL,
    city                      TEXT NOT NULL,
    country                   TEXT NOT NULL,
    address_line1             TEXT,
    address_line2             TEXT,
    postcode                  TEXT,
    property_type             TEXT,
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
CREATE INDEX IF NOT EXISTS idx_properties_city ON properties(city);
CREATE INDEX IF NOT EXISTS idx_properties_country ON properties(country);

-- 5.5 leads
CREATE TABLE IF NOT EXISTS leads (
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
    source_channel               TEXT,
    utm_source                   TEXT,
    utm_medium                   TEXT,
    utm_campaign                 TEXT,
    state                        TEXT NOT NULL DEFAULT 'new',
    priority                     TEXT NOT NULL DEFAULT 'medium',
    source_country               TEXT,
    destination_country          TEXT,
    destination_city             TEXT,
    preferred_university         TEXT,
    preferred_room_type          TEXT,
    preferred_move_in_month      INT,
    preferred_move_in_year       INT,
    budget_min                   NUMERIC(10,2),
    budget_max                   NUMERIC(10,2),
    budget_currency              TEXT DEFAULT 'GBP',
    duration_weeks               INT,
    is_phone_valid               BOOLEAN DEFAULT FALSE,
    is_ai_eligible               BOOLEAN DEFAULT FALSE,
    is_ai_qualified              BOOLEAN,
    ai_call_status               TEXT,
    ai_ineligible_reason         TEXT,
    is_bulk_upload               BOOLEAN DEFAULT FALSE,
    retry_count                  INT DEFAULT 0,
    next_retry_at                TIMESTAMPTZ,
    lost_reason                  TEXT,
    notification_sent_at         TIMESTAMPTZ,
    notification_delivery_status TEXT,
    last_activity_at             TIMESTAMPTZ,
    assigned_at                  TIMESTAMPTZ,
    state_changed_at             TIMESTAMPTZ,
    created_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_leads_assigned_agent ON leads(assigned_agent_id);
CREATE INDEX IF NOT EXISTS idx_leads_state ON leads(state);
CREATE INDEX IF NOT EXISTS idx_leads_priority ON leads(priority);
CREATE INDEX IF NOT EXISTS idx_leads_partner ON leads(partner_id);
CREATE INDEX IF NOT EXISTS idx_leads_destination_country ON leads(destination_country);
CREATE INDEX IF NOT EXISTS idx_leads_last_activity ON leads(last_activity_at);
CREATE INDEX IF NOT EXISTS idx_leads_next_retry ON leads(next_retry_at);
CREATE INDEX IF NOT EXISTS idx_leads_created_at ON leads(created_at);

-- 5.6 lead_activities
CREATE TABLE IF NOT EXISTS lead_activities (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id          UUID NOT NULL REFERENCES leads(id),
    agent_id         UUID REFERENCES users(id),
    activity_type    TEXT NOT NULL,
    subject          TEXT,
    body             TEXT,
    direction        TEXT,
    duration_seconds INT,
    outcome          TEXT,
    old_state        TEXT,
    new_state        TEXT,
    occurred_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_lead_activities_lead_id ON lead_activities(lead_id);
CREATE INDEX IF NOT EXISTS idx_lead_activities_agent_id ON lead_activities(agent_id);
CREATE INDEX IF NOT EXISTS idx_lead_activities_occurred_at ON lead_activities(occurred_at);
CREATE INDEX IF NOT EXISTS idx_lead_activities_type ON lead_activities(activity_type);

-- 5.7 tasks
CREATE TABLE IF NOT EXISTS tasks (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id              UUID NOT NULL REFERENCES leads(id),
    assigned_agent_id    UUID NOT NULL REFERENCES users(id),
    created_by_agent_id  UUID REFERENCES users(id),
    task_type            TEXT NOT NULL,
    title                TEXT NOT NULL,
    description          TEXT,
    priority             TEXT NOT NULL DEFAULT 'medium',
    status               TEXT NOT NULL DEFAULT 'pending',
    due_date             DATE NOT NULL,
    due_time             TIME,
    completed_at         TIMESTAMPTZ,
    snoozed_until        TIMESTAMPTZ,
    reminder_sent_at     TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tasks_assigned_agent ON tasks(assigned_agent_id);
CREATE INDEX IF NOT EXISTS idx_tasks_lead_id ON tasks(lead_id);
CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks(due_date);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);

-- 5.8 bookings
CREATE TABLE IF NOT EXISTS bookings (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id               UUID NOT NULL REFERENCES leads(id),
    property_id           UUID NOT NULL REFERENCES properties(id),
    agent_id              UUID NOT NULL REFERENCES users(id),
    external_booking_ref  TEXT UNIQUE,
    status                TEXT NOT NULL DEFAULT 'pending',
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
CREATE INDEX IF NOT EXISTS idx_bookings_lead_id ON bookings(lead_id);
CREATE INDEX IF NOT EXISTS idx_bookings_agent_id ON bookings(agent_id);
CREATE INDEX IF NOT EXISTS idx_bookings_property_id ON bookings(property_id);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);
CREATE INDEX IF NOT EXISTS idx_bookings_move_in_date ON bookings(move_in_date);

-- 5.9 payments
CREATE TABLE IF NOT EXISTS payments (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id              UUID NOT NULL REFERENCES bookings(id),
    lead_id                 UUID NOT NULL REFERENCES leads(id),
    agent_id                UUID REFERENCES users(id),
    payment_type            TEXT NOT NULL,
    amount                  NUMERIC(12,2) NOT NULL,
    currency                TEXT NOT NULL DEFAULT 'GBP',
    amount_in_gbp           NUMERIC(12,2),
    status                  TEXT NOT NULL DEFAULT 'pending',
    payment_method          TEXT,
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
CREATE INDEX IF NOT EXISTS idx_payments_booking_id ON payments(booking_id);
CREATE INDEX IF NOT EXISTS idx_payments_lead_id ON payments(lead_id);
CREATE INDEX IF NOT EXISTS idx_payments_agent_id ON payments(agent_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_paid_at ON payments(paid_at);

-- 5.10 conversations
CREATE TABLE IF NOT EXISTS conversations (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id),
    session_token    TEXT UNIQUE NOT NULL,
    title            TEXT,
    message_count    INT DEFAULT 0,
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_message_at  TIMESTAMPTZ,
    ended_at         TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_started_at ON conversations(started_at);

-- 5.11 messages
CREATE TABLE IF NOT EXISTS messages (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id      UUID NOT NULL REFERENCES conversations(id),
    user_id              UUID NOT NULL REFERENCES users(id),
    role                 TEXT NOT NULL,
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
CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_user_id ON messages(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

-- 5.12 query_feedback
CREATE TABLE IF NOT EXISTS query_feedback (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id       UUID UNIQUE NOT NULL REFERENCES messages(id),
    user_id          UUID NOT NULL REFERENCES users(id),
    is_helpful       BOOLEAN NOT NULL,
    correction_note  TEXT,
    corrected_sql    TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_query_feedback_is_helpful ON query_feedback(is_helpful);

-- 5.13 intent_examples
CREATE TABLE IF NOT EXISTS intent_examples (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_text       TEXT NOT NULL,
    question_embedding  vector(1536),
    sql_template        TEXT NOT NULL,
    tables_used         TEXT[],
    intent_category     TEXT,
    source              TEXT NOT NULL,
    upvote_count        INT DEFAULT 0,
    is_active           BOOLEAN DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_intent_examples_embedding
    ON intent_examples USING hnsw (question_embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
CREATE INDEX IF NOT EXISTS idx_intent_examples_category ON intent_examples(intent_category);

-- 5.14 schema_embeddings
CREATE TABLE IF NOT EXISTS schema_embeddings (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_name        TEXT NOT NULL,
    column_name       TEXT,
    description_text  TEXT NOT NULL,
    embedding         vector(1536),
    ddl_hash          TEXT NOT NULL,
    generated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_schema_embeddings_embedding
    ON schema_embeddings USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
CREATE INDEX IF NOT EXISTS idx_schema_embeddings_table ON schema_embeddings(table_name);

-- 5.15 intent_gaps
CREATE TABLE IF NOT EXISTS intent_gaps (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_cluster  TEXT NOT NULL,
    example_questions TEXT[],
    failure_count     INT DEFAULT 1,
    is_resolved       BOOLEAN DEFAULT FALSE,
    resolution_note   TEXT,
    first_seen_at     TIMESTAMPTZ DEFAULT NOW(),
    last_seen_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Read-only grants (after tables exist)
GRANT SELECT ON ALL TABLES IN SCHEMA public TO aria_readonly;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO aria_readonly;

