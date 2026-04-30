# tasks.md
# Aria — Build Tasks, Test Cases & Acceptance Criteria
### Every task is atomic. Every phase has a checkpoint. Do not proceed until all tests in a phase pass.

---

## How to use this file

- Tasks are numbered `TASK-XXX` and sequential within each phase
- Each task has explicit **acceptance tests** — manual verifications you run before marking done
- Each phase ends with a **PHASE CHECKPOINT** — all items must pass before moving to the next phase
- `[TEST]` = something you verify manually or with a script
- `[AUTO]` = write a Go test for this

---

## PHASE 0 — Project Scaffolding & Infrastructure
**Goal:** Everything runs with `docker compose up`. No app code yet. Just infra.
**Time estimate:** 1–2 days

---

### TASK-001 — Initialise Go module and project structure

- [ ] `mkdir aria && cd aria && git init`
- [ ] `go mod init github.com/{yourhandle}/aria`
- [ ] Create directory tree exactly as defined in spec.md section 3
- [ ] Create `cmd/api/main.go` with a single `fmt.Println("aria starting")` and `os.Exit(0)`
- [ ] Create `.env.example` with all variables from requirements.md section 6
- [ ] Create `.gitignore`: ignore `.env`, `bin/`, `*.exe`, `__pycache__`, `node_modules`, `*.log`
- [ ] Create `Makefile` with targets: `build`, `run`, `seed`, `schema-pipeline`, `test`, `lint`
- [ ] `go build ./cmd/api` — must compile with zero errors

**Acceptance tests:**
```
[TEST] go build ./cmd/api        → exits 0, binary produced
[TEST] git status                → .env not tracked
[TEST] make build                → compiles
```

---

### TASK-002 — Write docker-compose.yml

- [ ] Add `postgres` service: image `postgres:16`, env vars, port `5432`, volume mount for `./db/init.sql`
- [ ] Add healthcheck to postgres: `pg_isready -U aria`
- [ ] Add `redis` service: image `redis:7-alpine`, port `6379`
- [ ] Add healthcheck to redis: `redis-cli ping`
- [ ] Add `api` service: build from `./`, port `8080`, `depends_on: postgres (healthy), redis (healthy)`
- [ ] Add `frontend` service: build from `./frontend`, port `3000`, `depends_on: api`
- [ ] Add `schema-pipeline` service: build from `./schema_pipeline`, `restart: "no"`, depends on postgres healthy
- [ ] All services read from `.env` file via `env_file: .env`

**Acceptance tests:**
```
[TEST] docker compose config     → no errors, valid YAML
[TEST] docker compose up postgres redis
       → both show (healthy) in docker compose ps within 30s
[TEST] docker compose ps         → ports correctly mapped
```

---

### TASK-003 — Write db/init.sql — all tables

- [ ] `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`
- [ ] `CREATE EXTENSION IF NOT EXISTS vector;`
- [ ] Create read-only PostgreSQL role:
  ```sql
  CREATE ROLE aria_readonly LOGIN PASSWORD 'aria_ro';
  GRANT CONNECT ON DATABASE aria TO aria_readonly;
  GRANT USAGE ON SCHEMA public TO aria_readonly;
  GRANT SELECT ON ALL TABLES IN SCHEMA public TO aria_readonly;
  ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO aria_readonly;
  ```
- [ ] Write all 15 CREATE TABLE statements from requirements.md section 5 in dependency order:
  - teams → users → partners → properties
  - leads → lead_activities → tasks → bookings → payments
  - conversations → messages → query_feedback
  - intent_examples → schema_embeddings → intent_gaps
- [ ] All indexes from requirements.md section 5 included
- [ ] All HNSW indexes for pgvector columns included

**Acceptance tests:**
```
[TEST] docker compose up postgres
[TEST] psql $DATABASE_URL -c "\dt"
       → must show all 15 tables
[TEST] psql $DATABASE_URL -c "\di"
       → must show all declared indexes
[TEST] psql $DATABASE_URL_READONLY -c "SELECT 1 FROM leads LIMIT 1"
       → succeeds (read works)
[TEST] psql $DATABASE_URL_READONLY -c "INSERT INTO leads(first_name,last_name) VALUES('x','y')"
       → FAILS with permission denied (write blocked at DB level)
[TEST] psql $DATABASE_URL -c "SELECT extname FROM pg_extension WHERE extname='vector'"
       → returns 'vector'
```

---

### TASK-004 — Write seed data (db/seed.sql or Go seed script)

- [ ] Seed 4 teams (UK Pre-Sales, Canada Pre-Sales, Partnerships, Supply)
- [ ] Seed 10 users: 8 agents (2 per team), 2 admins — all with fake but realistic google_ids and emails
- [ ] Seed 20 partners: 15 universities, 5 portals, spread across UK/Canada/India
- [ ] Seed 50 properties: London(20), Manchester(10), Toronto(10), Vancouver(10) — realistic names, prices
- [ ] Seed 500 leads: ~50 per agent, realistic distribution:
  - States: new(15%), contacted(25%), interested(30%), qualified(15%), booked(10%), lost(4%), junk(1%)
  - Priorities: low(20%), medium(45%), high(25%), urgent(10%)
  - Mix of source countries, destination countries
- [ ] Seed 1000 lead_activities: ~2 per lead, mix of calls/emails/notes/state_changes
- [ ] Seed 300 tasks: mix of pending(40%), completed(40%), overdue(15%), snoozed(5%)
  - Overdue = due_date in the past with status=pending
- [ ] Seed 200 bookings linked to ~40% of leads
- [ ] Seed 150 payments linked to ~75% of bookings
- [ ] Add `make seed` to Makefile: runs seed script against DATABASE_URL

**Acceptance tests:**
```
[TEST] make seed
[TEST] psql $DATABASE_URL -c "SELECT COUNT(*) FROM leads"      → 500
[TEST] psql $DATABASE_URL -c "SELECT COUNT(*) FROM tasks"      → 300
[TEST] psql $DATABASE_URL -c "SELECT COUNT(*) FROM bookings"   → 200
[TEST] psql $DATABASE_URL -c "SELECT COUNT(*) FROM payments"   → 150
[TEST] psql $DATABASE_URL -c "SELECT COUNT(DISTINCT assigned_agent_id) FROM leads"
       → 8 (all agents have leads)
[TEST] psql $DATABASE_URL -c "SELECT COUNT(*) FROM tasks WHERE due_date < CURRENT_DATE AND status='pending'"
       → > 0 (overdue tasks exist)
[TEST] make seed (run twice) → same counts, no duplicate key errors (idempotent)
```

---

### TASK-005 — Go config loading

- [ ] Create `internal/config/config.go`
- [ ] Define `Config` struct with all env vars from requirements.md section 6
- [ ] `Load() (*Config, error)` reads from environment (godotenv loads .env in dev)
- [ ] Validation: if any required field is empty, return descriptive error
- [ ] Required fields: `DatabaseURL`, `DatabaseURLReadonly`, `RedisURL`, `JWTSecret`, `OpenAIAPIKey`, `GoogleClientID`, `GoogleClientSecret`, `GoogleRedirectURL`
- [ ] Wire into `cmd/api/main.go` — fail fast if config invalid

**Acceptance tests:**
```
[AUTO] TestConfig_LoadValid      → all fields populated, no error
[AUTO] TestConfig_MissingJWTSecret → returns error mentioning JWT_SECRET
[AUTO] TestConfig_MissingOpenAIKey → returns error mentioning OPENAI_API_KEY
[TEST] Run binary with empty .env → exits with clear error message, not panic
```

---

### TASK-006 — Database connection pool (pgx)

- [ ] Create `internal/db/postgres.go`
- [ ] `NewPool(ctx, config) (*pgxpool.Pool, error)` — creates pgxpool with max 25 conns
- [ ] `NewReadOnlyPool(ctx, config) (*pgxpool.Pool, error)` — connects as aria_readonly role
- [ ] Both pools have `statement_timeout = 5000ms` set via connection string
- [ ] Both pools ping on startup — fail fast if DB unreachable
- [ ] Wire both pools into main.go, pass to handlers via dependency injection (not globals)

**Acceptance tests:**
```
[AUTO] TestNewPool_Connect       → pool created, Ping succeeds
[AUTO] TestReadOnlyPool_NoWrite  → INSERT on readonly pool returns permission error
[TEST] docker compose up api     → logs show "database connected" not a panic
[TEST] Stop postgres mid-run     → api logs connection error, does not crash
```

---

### TASK-007 — Redis connection

- [ ] Create `internal/cache/redis.go`
- [ ] `NewClient(config) (*redis.Client, error)` — connects, pings, returns error if unreachable
- [ ] Helper: `SetWithTTL(ctx, key, value, ttl)` — JSON marshal + SET EX
- [ ] Helper: `Get(ctx, key) (string, bool, error)` — GET, returns (value, found, err)
- [ ] Helper: `Delete(ctx, key) error`
- [ ] Helper: `Exists(ctx, key) (bool, error)`
- [ ] Wire into main.go

**Acceptance tests:**
```
[AUTO] TestRedis_SetGet          → set "k"="v", get returns "v", found=true
[AUTO] TestRedis_GetMissing      → get non-existent key returns found=false, no error
[AUTO] TestRedis_TTLExpiry       → set TTL=1s, sleep 2s, get returns found=false
[AUTO] TestRedis_Delete          → set, delete, get returns found=false
[TEST] docker compose up api     → logs show "redis connected"
```

**PHASE 0 CHECKPOINT ✓**
```
[TEST] docker compose up                    → all services start, all healthy
[TEST] psql $DATABASE_URL -c "\dt"          → 15 tables present
[TEST] make seed                            → 500 leads, 300 tasks, 200 bookings
[TEST] psql $DATABASE_URL_READONLY (write)  → permission denied
[TEST] redis-cli -u $REDIS_URL ping         → PONG
[TEST] go build ./...                       → zero errors
[TEST] go test ./...                        → all written tests pass
```

---

## PHASE 1 — Authentication (Google OAuth + JWT)
**Goal:** Agent can log in with Google. JWT issued. All subsequent requests authenticated.
**Time estimate:** 3–4 days

---

### TASK-008 — JWT service

- [ ] Create `internal/auth/jwt.go`
- [ ] `SignToken(userID, email, role string, secret string, expiry time.Duration) (string, error)`
  - Claims: `sub` (userID), `email`, `role`, `jti` (uuid for blacklisting), `iat`, `exp`
- [ ] `VerifyToken(tokenString, secret string) (*Claims, error)`
  - Returns error if expired, malformed, or wrong signature
- [ ] `BlacklistToken(ctx, redis, jti string, remainingTTL time.Duration) error`
  - `SET jwt_blacklist:{jti} "1" EX {remainingTTL}`
- [ ] `IsBlacklisted(ctx, redis, jti string) (bool, error)`

**Acceptance tests:**
```
[AUTO] TestSignToken_Valid        → token parses, claims match inputs
[AUTO] TestVerifyToken_Expired    → expired token returns error
[AUTO] TestVerifyToken_Tampered   → modified token returns error
[AUTO] TestVerifyToken_WrongKey   → different secret returns error
[AUTO] TestBlacklist_BlocksToken  → blacklisted jti returns isBlacklisted=true
[AUTO] TestBlacklist_ExpiredEntry → after TTL, returns isBlacklisted=false
```

---

### TASK-009 — JWT middleware

- [ ] Create `internal/auth/middleware.go`
- [ ] `Authenticate(jwtSecret string, redis *redis.Client) func(http.Handler) http.Handler`
- [ ] Extracts `Authorization: Bearer <token>` header
- [ ] Returns 401 if header missing
- [ ] Returns 401 if token invalid/expired
- [ ] Returns 401 if jti is blacklisted in Redis
- [ ] On success: attach `Claims` to request context via typed context key
- [ ] Helper: `ClaimsFromContext(ctx) (*Claims, bool)` — used in handlers

**Acceptance tests:**
```
[AUTO] TestMiddleware_NoHeader        → 401
[AUTO] TestMiddleware_InvalidToken    → 401
[AUTO] TestMiddleware_ExpiredToken    → 401
[AUTO] TestMiddleware_BlacklistedJTI  → 401
[AUTO] TestMiddleware_ValidToken      → next handler called, claims in context
[AUTO] TestClaimsFromContext_Present  → returns claims, true
[AUTO] TestClaimsFromContext_Missing  → returns nil, false
```

---

### TASK-010 — User repository

- [ ] Create `internal/models/user.go` — User struct matching users table
- [ ] Create `internal/db/user_repo.go`
- [ ] `FindByGoogleID(ctx, pool, googleID string) (*User, error)` — returns nil if not found
- [ ] `Create(ctx, pool, user *User) (*User, error)` — inserts, returns with generated id
- [ ] `FindByID(ctx, pool, id string) (*User, error)`
- [ ] `UpdateLastLogin(ctx, pool, id string) error` — sets last_login_at = NOW()

**Acceptance tests:**
```
[AUTO] TestFindByGoogleID_Exists     → returns correct user
[AUTO] TestFindByGoogleID_NotFound   → returns nil, no error
[AUTO] TestCreate_NewUser            → user created, id populated
[AUTO] TestCreate_DuplicateGoogleID  → returns unique constraint error
[AUTO] TestUpdateLastLogin           → last_login_at changes
```

---

### TASK-011 — Google OAuth handlers

- [ ] Install `golang.org/x/oauth2` and `golang.org/x/oauth2/google`
- [ ] Create `internal/auth/google.go`
- [ ] `GoogleConfig(cfg *Config) *oauth2.Config` — scopes: openid, email, profile
- [ ] `HandleGoogleLogin(oauthCfg) http.HandlerFunc`
  - Generate random state token, store in Redis TTL 10min: `oauth_state:{state}`
  - Redirect to Google consent URL with state param
- [ ] `HandleGoogleCallback(oauthCfg, pool, rdb, cfg) http.HandlerFunc`
  - Verify state token exists in Redis (CSRF protection)
  - Exchange code for token
  - Fetch user info from Google userinfo endpoint
  - Upsert user in DB (FindByGoogleID → if nil, Create)
  - UpdateLastLogin
  - Sign JWT (8h)
  - Generate refresh token (UUID), store in Redis `refresh:{token}` = userID, TTL 30 days
  - Set refresh token as httpOnly, Secure, SameSite=Strict cookie
  - Return JSON: `{ access_token, user: { id, email, full_name, avatar_url, role } }`
- [ ] `HandleRefresh(pool, rdb, cfg) http.HandlerFunc`
  - Read refresh_token cookie
  - Lookup in Redis → get userID
  - FindByID in DB
  - Sign new JWT
  - Return `{ access_token }`
- [ ] `HandleLogout(rdb) http.HandlerFunc`
  - Blacklist current JWT jti in Redis
  - Delete refresh token from Redis
  - Clear cookie

**Acceptance tests:**
```
[TEST] GET /auth/google
       → 302 redirect to accounts.google.com

[TEST] GET /auth/callback with invalid state
       → 400 Bad Request

[TEST] Full OAuth flow (use Google OAuth Playground or real browser):
       → JWT returned in response body
       → refresh_token cookie set (httpOnly=true, check DevTools)
       → User row created in DB: SELECT * FROM users WHERE email='your@gmail.com'
       → last_login_at populated

[TEST] Second login with same Google account
       → No duplicate user created
       → last_login_at updated

[TEST] POST /auth/refresh with valid cookie
       → New access_token returned

[TEST] POST /auth/refresh with missing/expired cookie
       → 401

[TEST] POST /auth/logout
       → 200
       → GET /chat with old token → 401 (blacklisted)
       → Cookie cleared in response headers
```

---

### TASK-012 — Wire auth routes into Chi router

- [ ] Create Chi router in `cmd/api/main.go`
- [ ] Public routes (no middleware): `GET /auth/google`, `GET /auth/callback`, `POST /auth/refresh`
- [ ] Protected routes group (with JWT middleware): all other routes
- [ ] `GET /health` — returns `{"status":"ok","time":"..."}` — no auth required
- [ ] `POST /auth/logout` — requires auth (needs JWT to blacklist)
- [ ] Add request logging middleware (zerolog) to all routes
- [ ] Add CORS middleware: allow `http://localhost:3000`

**Acceptance tests:**
```
[TEST] GET /health                            → 200 {"status":"ok"}
[TEST] GET /chat (no token)                  → 401
[TEST] GET /chat (valid token)               → 404 (route not yet built, but not 401)
[TEST] Logs show structured JSON per request → check docker compose logs api
[TEST] curl from localhost:3001 (wrong port)  → CORS error
[TEST] curl from localhost:3000              → no CORS error
```

**PHASE 1 CHECKPOINT ✓**
```
[TEST] Full Google login in browser → JWT returned
[TEST] curl -H "Authorization: Bearer <invalid>" /health → 401
[TEST] curl -H "Authorization: Bearer <valid>" /health → 200
[TEST] Logout → old token rejected
[TEST] Refresh token flow works
[TEST] go test ./internal/auth/... → all pass
[TEST] New user auto-created in DB on first login
```

---

## PHASE 2 — Schema Intelligence Pipeline (Python)
**Goal:** Python script reads DB schema, generates GPT-4o docs, stores embeddings in pgvector.
**Time estimate:** 3–4 days

---

### TASK-013 — Python project setup

- [ ] Create `schema_pipeline/requirements.txt`:
  ```
  openai==1.30.0
  psycopg2-binary==2.9.9
  pgvector==0.2.5
  python-dotenv==1.0.1
  hashlib (stdlib)
  ```
- [ ] Create `schema_pipeline/Dockerfile`: Python 3.12 slim, install requirements
- [ ] Create `schema_pipeline/run.py` — entry point, reads CLI args or env vars
- [ ] Test: `docker compose run schema-pipeline python run.py --help` — no import errors

---

### TASK-014 — DB introspection module

- [ ] Create `schema_pipeline/introspect.py`
- [ ] `get_tables(conn) -> list[str]` — all table names in public schema
- [ ] `get_table_ddl(conn, table_name) -> str` — reconstructs CREATE TABLE statement from information_schema
- [ ] `get_foreign_keys(conn, table_name) -> list[dict]` — FK relationships
- [ ] `get_sample_rows(conn, table_name, n=3) -> list[dict]` — SELECT n rows as dicts
- [ ] `hash_ddl(ddl: str) -> str` — SHA256 hex of DDL string

**Acceptance tests:**
```
[TEST] python -c "from introspect import get_tables; print(get_tables(conn))"
       → prints list of 15 table names

[TEST] python -c "from introspect import get_table_ddl; print(get_table_ddl(conn, 'leads'))"
       → prints valid CREATE TABLE with all columns

[TEST] hash_ddl("same string") == hash_ddl("same string") → True
[TEST] hash_ddl("string a") != hash_ddl("string b")       → True

[TEST] get_sample_rows(conn, 'leads', 3)
       → returns 3 dicts with all lead columns as flat keys (no nesting)
```

---

### TASK-015 — GPT-4o documentation generator

- [ ] Create `schema_pipeline/document.py`
- [ ] `generate_table_doc(openai_client, table_name, ddl, fks, sample_rows) -> dict`
  - Prompt instructs GPT-4o to return structured JSON:
    ```json
    {
      "table_name": "leads",
      "purpose": "...",
      "columns": [
        { "name": "state", "description": "...", "possible_values": ["new","contacted",...] }
      ],
      "common_query_patterns": ["..."],
      "relationships": ["leads.assigned_agent_id → users.id"]
    }
    ```
  - Temperature: 0.1 (factual, consistent)
  - Parse and validate JSON response
- [ ] `format_for_embedding(doc: dict) -> str`
  - Flattens the doc to a plain-text string suitable for embedding
  - Format: "Table: leads. Purpose: ... Column state: ... Column priority: ..."

**Acceptance tests:**
```
[TEST] generate_table_doc(client, "leads", leads_ddl, fks, samples)
       → returns dict with keys: table_name, purpose, columns, relationships
       → columns list has entry for every column in DDL
       → state column has possible_values: ['new','contacted','interested','qualified','booked','lost','junk']

[TEST] format_for_embedding(doc)
       → returns non-empty string
       → contains "leads" and "assigned_agent_id"
       → under 8000 characters (fits in embedding context)
```

---

### TASK-016 — Embedding + pgvector storage

- [ ] Create `schema_pipeline/embed.py`
- [ ] `embed_text(openai_client, text: str) -> list[float]` — text-embedding-3-small, dim=1536
- [ ] `upsert_schema_embedding(conn, table_name, column_name, description, embedding, ddl_hash)`
  - DELETE existing rows for this table_name + column_name combo
  - INSERT new row
- [ ] `load_manifest(path) -> dict` — loads `{table_name: ddl_hash}` from JSON file
- [ ] `save_manifest(path, manifest: dict)` — saves updated manifest

**Acceptance tests:**
```
[TEST] embed_text(client, "leads table stores potential students")
       → returns list of 1536 floats
       → all values between -1.0 and 1.0

[TEST] upsert_schema_embedding(conn, "leads", None, "Leads table...", embedding, hash)
       → SELECT COUNT(*) FROM schema_embeddings WHERE table_name='leads' → 1
       → run again → still 1 (upsert, not insert)

[TEST] pgvector similarity search:
       SELECT description_text FROM schema_embeddings
       ORDER BY embedding <=> '[...query_embedding...]' LIMIT 3
       → returns rows (proves HNSW index working)
```

---

### TASK-017 — Intent examples generator

- [ ] Create `schema_pipeline/generate_examples.py`
- [ ] `generate_intent_examples(openai_client, all_table_docs: list[dict]) -> list[dict]`
  - Prompt: given all table documentation, generate 30 example Q→SQL pairs
  - Each pair: `{ question, sql_template, tables_used, intent_category }`
  - SQL uses `:agent_id` as placeholder for the logged-in agent
  - Examples must cover: lead status queries, task queries, booking queries, payment queries, activity queries, cross-table joins
- [ ] `upsert_intent_examples(conn, openai_client, examples: list[dict])`
  - Embed each question
  - INSERT INTO intent_examples (with all fields)

**Acceptance tests:**
```
[TEST] generate_intent_examples(client, docs)
       → returns list of 30 dicts
       → every dict has: question, sql_template, tables_used, intent_category
       → sql_template for lead queries contains 'assigned_agent_id'
       → sql_template contains ':agent_id' placeholder

[TEST] After upsert:
       SELECT COUNT(*) FROM intent_examples → 30
       SELECT * FROM intent_examples WHERE intent_category='lead_status' → > 0
       SELECT * FROM intent_examples WHERE intent_category='task_overdue' → > 0

[TEST] Similarity search on intent_examples:
       embed "my pending tasks today" → cosine search → top result is a task query
```

---

### TASK-018 — Wire pipeline + incremental run

- [ ] Wire `run.py`: introspect → hash check → document → embed → generate examples
- [ ] On first run: process all tables
- [ ] On subsequent run: only process tables whose DDL hash changed
- [ ] Print progress: `Processing table: leads (changed)` / `Skipping table: teams (unchanged)`
- [ ] Final summary: `Done. Processed 15/15 tables. Stored 90 embeddings. Generated 30 intent examples.`

**Acceptance tests:**
```
[TEST] make schema-pipeline (first run)
       → all 15 tables processed
       → SELECT COUNT(*) FROM schema_embeddings → > 0
       → SELECT COUNT(*) FROM intent_examples → 30
       → manifest.json created with 15 entries

[TEST] make schema-pipeline (second run, no changes)
       → output: "Skipping X tables (unchanged)"
       → schema_embeddings count unchanged
       → runs in < 5 seconds (no API calls)

[TEST] ALTER TABLE leads ADD COLUMN test_col TEXT; make schema-pipeline
       → output: "Processing table: leads (changed)"
       → only leads re-processed
       → manifest updated
       → ALTER TABLE leads DROP COLUMN test_col; (cleanup)
```

**PHASE 2 CHECKPOINT ✓**
```
[TEST] make schema-pipeline → completes without error
[TEST] SELECT COUNT(*) FROM schema_embeddings → > 50 rows
[TEST] SELECT COUNT(*) FROM intent_examples → 30 rows
[TEST] SELECT description_text FROM schema_embeddings WHERE table_name='leads' LIMIT 1
       → readable English description mentioning 'agent' and 'state'
[TEST] Similarity search returns task-related docs for "pending tasks" query
[TEST] Second run is fast (incremental, no API calls)
```

---

## PHASE 3 — AI Query Service (Core)
**Goal:** Natural language → SQL → executed → result. No streaming yet. No auth scope yet.
**Time estimate:** 4–5 days

---

### TASK-019 — Schema retriever

- [ ] Create `internal/ai/schema_retriever.go`
- [ ] `RetrieveRelevantSchema(ctx, readPool, openaiClient, question string, topK int) ([]SchemaDoc, error)`
  - Embed the question using OpenAI text-embedding-3-small
  - Query `schema_embeddings` by cosine similarity: `ORDER BY embedding <=> $1 LIMIT $2`
  - Return top-K SchemaDoc structs (table_name, column_name, description_text)
- [ ] `RetrieveIntentExamples(ctx, readPool, openaiClient, question string, topK int) ([]IntentExample, error)`
  - Embed question, similarity search on `intent_examples.question_embedding`
  - Return top-K intent examples (question_text, sql_template, intent_category)

**Acceptance tests:**
```
[AUTO] TestRetrieveSchema_ReturnsTopK
       → question "show me overdue tasks" → returns 5 docs
       → all returned docs have non-empty description_text
       → tasks table appears in results

[AUTO] TestRetrieveIntentExamples_SimilarQuestion
       → question "my pending follow ups" → top result has intent_category containing "task"

[AUTO] TestRetrieve_HandlesNoEmbeddings
       → empty schema_embeddings table → returns empty slice, no error
```

---

### TASK-020 — SQL validator

- [ ] Create `internal/ai/sql_validator.go`
- [ ] `Validate(sql string) error`
  - Must start with SELECT (case insensitive, after trimming whitespace)
  - Must not contain: INSERT, UPDATE, DELETE, DROP, CREATE, ALTER, TRUNCATE, EXEC, EXECUTE
  - Must not contain SQL comment sequences: `--`, `/*`, `*/`
  - Must not contain semicolons except at end (prevents statement chaining)
- [ ] `InjectAgentFilter(sql, agentID string) (string, error)`
  - Replace `:agent_id` placeholder with parameterised `$1`
  - Return modified SQL + confirm placeholder was present
  - If placeholder missing: append `WHERE assigned_agent_id = $1` or `AND assigned_agent_id = $1`

**Acceptance tests:**
```
[AUTO] TestValidate_ValidSelect      → "SELECT id FROM leads" → nil
[AUTO] TestValidate_Insert           → "INSERT INTO leads..." → error
[AUTO] TestValidate_Update           → "UPDATE leads SET..." → error
[AUTO] TestValidate_Drop             → "DROP TABLE leads"    → error
[AUTO] TestValidate_CommentInjection → "SELECT 1--; DROP..."  → error
[AUTO] TestValidate_Semicolon        → "SELECT 1; DELETE..."  → error
[AUTO] TestInjectFilter_HasPlaceholder
       → SQL with :agent_id → replaced with $1
[AUTO] TestInjectFilter_MissingPlaceholder
       → SQL without filter → AND injected at end
```

---

### TASK-021 — SQL executor

- [ ] Create `internal/ai/sql_executor.go`
- [ ] `Execute(ctx, readPool, sql string, args ...any) ([]map[string]any, time.Duration, error)`
  - Use pgx read-only pool
  - ctx with 5s timeout: `ctx, cancel = context.WithTimeout(ctx, 5*time.Second)`
  - Execute query with args
  - Scan all rows into `[]map[string]any`
  - Return rows, execution duration, error
  - On timeout: return specific `ErrQueryTimeout` error

**Acceptance tests:**
```
[AUTO] TestExecute_SimpleSelect
       → "SELECT id, first_name FROM leads LIMIT 3" → 3 rows returned

[AUTO] TestExecute_WithParam
       → "SELECT id FROM leads WHERE assigned_agent_id = $1" + agentID → filters correctly

[AUTO] TestExecute_Timeout
       → use pg_sleep(10) → returns ErrQueryTimeout within ~5s

[AUTO] TestExecute_InvalidSQL
       → malformed SQL → returns error, not panic

[AUTO] TestExecute_ZeroRows
       → valid SQL returning no rows → empty slice, no error
```

---

### TASK-022 — Text-to-SQL service (GPT-4o tool calling)

- [ ] Create `internal/ai/text_to_sql.go`
- [ ] Define the `query_crm_database` tool spec (see spec.md section 6.1)
- [ ] `GenerateSQL(ctx, config, schemaDocs, intentExamples, conversation []Message, question, agentID string) (string, string, error)`
  - Returns: (sql, explanation, error)
  - Build system prompt from spec.md section 6.2: agent context + schema docs + intent examples + history
  - Call GPT-4o with tool spec
  - Extract tool call arguments → parse sql and explanation
  - If GPT-4o returns text instead of tool call (out-of-scope question) → return (empty, explanation, ErrOutOfScope)
- [ ] Temperature: 0.1

**Acceptance tests:**
```
[AUTO] TestGenerateSQL_LeadQuestion
       → "show me my leads" → returns non-empty SQL containing SELECT and FROM leads

[AUTO] TestGenerateSQL_TaskQuestion
       → "what tasks are due today" → SQL contains FROM tasks and current date filter

[AUTO] TestGenerateSQL_OutOfScope
       → "what is the weather in London" → returns ErrOutOfScope

[AUTO] TestGenerateSQL_IncludesAgentFilter
       → any lead question → returned SQL contains 'assigned_agent_id'

[AUTO] TestGenerateSQL_ConversationContext
       → history has "7 leads shown", follow-up "sort by last contact"
       → new SQL contains ORDER BY last_activity_at
```

---

### TASK-023 — Self-correction retry

- [ ] In `text_to_sql.go`, add `GenerateSQLWithRetry(ctx, ..., originalSQL, dbError string) (string, error)`
  - Called when first SQL attempt fails execution
  - Appends error to prompt: "Your previous SQL failed with: {error}. Please fix it."
  - Single retry attempt only
  - Returns corrected SQL or error

**Acceptance tests:**
```
[AUTO] TestSelfCorrection_FixesTypo
       → feed SQL with wrong column name, db error message → corrected SQL has right name

[AUTO] TestSelfCorrection_SingleAttemptOnly
       → verify only one retry happens, not infinite loop
```

---

### TASK-024 — Query cache

- [ ] Create `internal/cache/query_cache.go`
- [ ] `NormaliseQuestion(question string) string` — lowercase, trim, collapse whitespace
- [ ] `CacheKey(userID, question string) string` — `query_cache:{userID}:{sha256(normalised)}`
- [ ] `GetCachedQuery(ctx, rdb, userID, question string) (*CachedResult, bool, error)`
- [ ] `SetCachedQuery(ctx, rdb, userID, question string, result *CachedResult, ttl time.Duration) error`
- [ ] `InvalidateCachedQuery(ctx, rdb, userID, question string) error`
- [ ] CachedResult struct: `{ Answer string, SQL string, RowCount int, GeneratedAt time.Time }`

**Acceptance tests:**
```
[AUTO] TestCacheKey_Deterministic
       → same question always produces same key
[AUTO] TestCacheKey_CaseInsensitive
       → "My Leads" and "my leads" produce same key
[AUTO] TestGetSet_RoundTrip
       → set result → get → returns same result, found=true
[AUTO] TestGet_Miss
       → non-existent key → found=false, no error
[AUTO] TestInvalidate
       → set → invalidate → get → found=false
[AUTO] TestCacheTTL
       → set with 1s TTL → sleep 2s → found=false
```

---

### TASK-025 — Conversation/session manager

- [ ] Create `internal/cache/session.go`
- [ ] `SessionKey(userID, sessionID string) string` — `session:{userID}:{sessionID}`
- [ ] `GetHistory(ctx, rdb, userID, sessionID string) ([]Message, error)` — deserialise JSON array
- [ ] `AppendMessage(ctx, rdb, userID, sessionID string, msg Message, maxMessages int, ttl time.Duration) error`
  - Get existing history, append, trim to last `maxMessages`, re-save with TTL reset
- [ ] `ClearSession(ctx, rdb, userID, sessionID string) error`
- [ ] Message struct: `{ Role string, Content string, GeneratedSQL string, CreatedAt time.Time }`

**Acceptance tests:**
```
[AUTO] TestGetHistory_Empty       → new session → empty slice, no error
[AUTO] TestAppend_SingleMessage   → append 1 → history has 1
[AUTO] TestAppend_TrimToMax       → append 12 messages, max=10 → history has 10 (oldest dropped)
[AUTO] TestClearSession           → append 3 → clear → history empty
[AUTO] TestHistoryTTLReset        → append message → TTL resets (verify with Redis TTL command)
```

---

### TASK-026 — Conversation + message persistence (PostgreSQL)

- [ ] Create `internal/db/conversation_repo.go`
- [ ] `CreateConversation(ctx, pool, userID, sessionToken string) (*Conversation, error)`
- [ ] `FindConversationBySession(ctx, pool, sessionToken string) (*Conversation, error)`
- [ ] `UpdateConversationLastMessage(ctx, pool, conversationID string) error`
- [ ] `CreateMessage(ctx, pool, msg *Message) (*Message, error)` — insert with all fields
- [ ] `ListConversations(ctx, pool, userID string) ([]*Conversation, error)`
- [ ] `ListMessages(ctx, pool, conversationID string) ([]*Message, error)`

**Acceptance tests:**
```
[AUTO] TestCreateConversation     → creates, returns with ID
[AUTO] TestFindBySession          → created → found by session token
[AUTO] TestCreateMessage          → persists all fields including generated_sql
[AUTO] TestListConversations      → returns only the requesting user's conversations
[AUTO] TestListMessages_Order     → returns messages in created_at ASC order
```

---

### TASK-027 — Full AI query pipeline (non-streaming)

- [ ] Create `internal/ai/service.go`
- [ ] `QueryService` struct: holds openaiClient, readPool, rdb, config
- [ ] `Query(ctx, req QueryRequest) (*QueryResult, error)`:
  1. Check Redis cache → if hit, return cached result
  2. Load session history from Redis
  3. Retrieve relevant schema docs (top-5)
  4. Retrieve intent examples (top-3)
  5. Call GenerateSQL
  6. If ErrOutOfScope → return out-of-scope response
  7. Validate SQL
  8. Inject agent filter
  9. Execute SQL (with 5s timeout)
  10. If execution fails → retry GenerateSQLWithRetry once
  11. Call GPT-4o to format rows into natural language answer
  12. Cache result in Redis
  13. Return QueryResult
- [ ] QueryRequest: `{ Question, SessionID, AgentID, AgentRole, AgentName, Timezone string }`
- [ ] QueryResult: `{ Answer, SQL, RowCount, ExecutionMs, WasCached, WasCorrected, TokensIn, TokensOut int }`

**Acceptance tests:**
```
[AUTO] TestQuery_SimpleLead
       → "show me my leads" with seeded agentID
       → result.SQL contains SELECT
       → result.RowCount > 0
       → result.Answer is non-empty string

[AUTO] TestQuery_CacheHit
       → run same question twice → second result.WasCached = true

[AUTO] TestQuery_AgentIsolation
       → agentID_1 asks "show my leads"
       → agentID_2 asks same question
       → results are different (each sees only their own leads)
       → neither result contains the other agent's data

[AUTO] TestQuery_OutOfScope
       → "what is 2+2?" → returns out-of-scope explanation, no SQL executed

[AUTO] TestQuery_ZeroResults
       → question that genuinely returns no rows
       → Answer says "no results found", does not fabricate

[TEST] Run manually against seeded DB, ask 5 different questions, verify all correct
```

**PHASE 3 CHECKPOINT ✓**
```
[TEST] make schema-pipeline → embeddings present
[TEST] POST /query (temp endpoint) "show me my high priority leads"
       → returns SQL + natural language answer
       → SQL only queries the requesting agent's leads
       → 0 results question returns "no results", not fabricated data
[TEST] Same question twice → second is cache hit (< 100ms)
[TEST] "drop table leads" → rejected at validator
[TEST] "what's the weather" → out-of-scope response
[TEST] go test ./internal/ai/... → all pass
[TEST] go test ./internal/cache/... → all pass
```

---

## PHASE 4 — Chat API with SSE Streaming
**Goal:** `POST /chat` endpoint, authenticated, streams response token by token.
**Time estimate:** 3–4 days

---

### TASK-028 — SSE streamer

- [ ] Create `internal/ai/streamer.go`
- [ ] `StreamQuery(ctx, w http.ResponseWriter, queryService, req QueryRequest) error`
  - Set SSE headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no`, `Connection: keep-alive`
  - If cache hit: stream entire cached answer in chunks of 20 chars, send `{"type":"done","cached":true}`
  - If cache miss:
    1. Run GenerateSQL → send `{"type":"sql","sql":"..."}` event immediately
    2. Start GPT-4o streaming completion with the formatted rows as context
    3. For each streamed token: send `{"type":"token","text":"..."}` + flush
    4. On completion: send `{"type":"done","message_id":"...","cached":false,"row_count":N}`
  - On any error: send `{"type":"error","message":"..."}` event

**Acceptance tests:**
```
[AUTO] TestSSEHeaders
       → response has Content-Type: text/event-stream
       → response has Cache-Control: no-cache

[TEST] curl -N -H "Authorization: Bearer <jwt>" \
       -d '{"question":"show my tasks today"}' \
       http://localhost:8080/chat
       → see SSE events streaming in terminal:
         data: {"type":"sql","sql":"SELECT ..."}
         data: {"type":"token","text":"You have "}
         data: {"type":"token","text":"3 tasks "}
         ...
         data: {"type":"done","message_id":"...","cached":false}
         data: [DONE]

[TEST] Same question again → cached response streams quickly (~100ms total)
```

---

### TASK-029 — Chat handler

- [ ] Create `internal/handlers/chat.go`
- [ ] `POST /chat` — requires JWT auth
- [ ] Parse body: `{ question, session_id?, conversation_id? }`
- [ ] Validate: question non-empty, max 500 chars
- [ ] Rate limit check: `rate_limit:{userID}` in Redis → if > 30 → 429 with `Retry-After` header
- [ ] If no session_id provided: generate new UUID
- [ ] If no conversation_id: CreateConversation in DB, use new ID
- [ ] Extract Claims from context (userID, role, email)
- [ ] Build QueryRequest from claims + body
- [ ] Call StreamQuery
- [ ] After stream completes: persist user + assistant messages to DB
- [ ] Rate limit increment: `INCR rate_limit:{userID}`, `EXPIRE rate_limit:{userID} 60` (if new key)

**Acceptance tests:**
```
[TEST] POST /chat (no auth)            → 401
[TEST] POST /chat (empty question)     → 400 {"error":"question required"}
[TEST] POST /chat (question > 500 chars) → 400 {"error":"question too long"}
[TEST] POST /chat (valid, real question) → SSE stream, correct answer
[TEST] Rate limit: fire 31 requests in 1 minute → 31st returns 429
[TEST] After stream: SELECT * FROM messages WHERE conversation_id=?
       → 2 rows (user message + assistant message)
       → assistant message has generated_sql populated
       → assistant message has sql_row_count, sql_execution_ms populated
[TEST] Two agents ask same question → each gets their own data (isolation)
```

---

### TASK-030 — Conversations handler

- [ ] Create `internal/handlers/conversations.go`
- [ ] `GET /conversations` — returns the logged-in user's conversation list
- [ ] `GET /conversations/:id/messages` — returns messages for a conversation
  - Validate: conversation must belong to requesting user → 403 if not
- [ ] `DELETE /conversations/:id` — marks ended_at, clears Redis session

**Acceptance tests:**
```
[TEST] GET /conversations (no conversations yet)  → 200 []
[TEST] After POST /chat → GET /conversations      → 1 conversation returned
[TEST] GET /conversations/:id/messages            → messages in order
[TEST] GET /conversations/:other_user_id/messages → 403 (not your conversation)
[TEST] DELETE /conversations/:id
       → 204
       → GET /conversations/:id/messages → 404
       → Redis session cleared
```

---

### TASK-031 — Feedback handler

- [ ] Create `internal/handlers/feedback.go`
- [ ] `POST /messages/:id/feedback`
  - Validate message exists and belongs to requesting user
  - Validate body: `{ is_helpful: bool, correction_note?: string, corrected_sql?: string }`
  - Insert into query_feedback (UNIQUE on message_id → 409 if already rated)
  - If `is_helpful = false`: invalidate the cache entry for this question
  - If `is_helpful = true` and upvote_count would reach 3: auto-promote to intent_examples
  - Return 201 with feedback record

**Acceptance tests:**
```
[TEST] POST /messages/:id/feedback { is_helpful: true }
       → 201
       → SELECT * FROM query_feedback WHERE message_id=? → row with is_helpful=true

[TEST] POST /messages/:id/feedback twice on same message
       → 409 Conflict

[TEST] POST /messages/:id/feedback { is_helpful: false, correction_note: "Wrong agent filter" }
       → 201
       → Redis cache for that question is cleared (next request goes to LLM)

[TEST] POST /messages/:id/feedback (message belongs to other user)
       → 403

[TEST] After 3 upvotes on similar questions:
       → SELECT * FROM intent_examples WHERE source='promoted_feedback' → row exists
```

---

### TASK-032 — Admin metrics handler

- [ ] Create `internal/handlers/admin.go`
- [ ] `GET /admin/metrics` — requires role=admin in JWT claims → 403 if role=agent
- [ ] Queries:
  - Queries today: `SELECT COUNT(*) FROM messages WHERE role='assistant' AND created_at >= CURRENT_DATE`
  - Avg response time: avg of `sql_execution_ms` from today's messages
  - Cache hit rate: `COUNT(*) WHERE was_cached=true / COUNT(*) total` today
  - Feedback score: `COUNT(*) WHERE is_helpful=true / COUNT(*) total` from query_feedback
  - Intent gaps: `SELECT COUNT(*) FROM intent_gaps WHERE is_resolved=false`

**Acceptance tests:**
```
[TEST] GET /admin/metrics (agent JWT)   → 403
[TEST] GET /admin/metrics (admin JWT)   → 200 with all fields
[TEST] After asking 10 questions, 3 cached:
       → cache_hit_rate ~= 0.3
[TEST] After 5 helpful, 1 unhelpful feedback:
       → feedback_score ~= 0.83
```

**PHASE 4 CHECKPOINT ✓**
```
[TEST] Full flow in curl:
       1. GET /auth/google → login with Google → get JWT
       2. POST /chat "what are my pending tasks for today?" → streams answer
       3. GET /conversations → 1 conversation
       4. POST /messages/:id/feedback { is_helpful: true } → 201
       5. POST /chat same question → cached response (< 100ms)
       6. POST /auth/logout → old JWT rejected
[TEST] Agent A cannot see Agent B's leads
[TEST] Rate limit blocks after 30 requests/minute
[TEST] go test ./internal/handlers/... → all pass
[TEST] go test ./... → all pass
```

---

## PHASE 5 — Rate Limiting, Error Handling & Observability
**Goal:** Production-grade error handling, structured logging, graceful degradation.
**Time estimate:** 2–3 days

---

### TASK-033 — Structured logging with zerolog

- [ ] Add `rs/zerolog` to go.mod
- [ ] Create logging middleware: logs every request as JSON: `method`, `path`, `status`, `duration_ms`, `user_id`
- [ ] In AI service: log every LLM call: `question_length`, `sql_generated`, `tokens_in`, `tokens_out`, `execution_ms`, `cache_hit`, `was_corrected`
- [ ] Log levels: DEBUG for dev (sql queries), INFO for request logs, ERROR for failures
- [ ] All errors logged with stack context: `log.Error().Err(err).Str("user_id", uid).Msg("sql execution failed")`

**Acceptance tests:**
```
[TEST] docker compose logs api | head -50
       → every line is valid JSON
       → every request log has: method, path, status, duration_ms, user_id
       → LLM calls logged with token counts
[TEST] Invalid request → error logged with context, not just "error occurred"
```

---

### TASK-034 — Graceful degradation

- [ ] If OpenAI API returns 503/429: return `{"error":"AI service temporarily unavailable, please try again"}` — not 500
- [ ] If Redis unreachable: skip cache entirely, log warning, proceed with LLM call
- [ ] If SQL times out: return `{"error":"Your query took too long. Try asking for a smaller date range."}` — not 500
- [ ] If schema embeddings table is empty: proceed with full schema in prompt (fallback mode), log warning
- [ ] All external calls wrapped in `context.WithTimeout`

**Acceptance tests:**
```
[TEST] docker compose stop redis → POST /chat still works (slower, no cache)
       → logs show "redis unavailable, skipping cache"
[TEST] Set invalid OPENAI_API_KEY → POST /chat returns graceful error, not 500
[TEST] Ask question requiring pg_sleep(10) → returns timeout message within 6s
[TEST] All degradation paths return 200 with error in body (not 5xx) — SSE protocol
```

---

### TASK-035 — Input sanitisation and edge cases

- [ ] Question length validation: max 500 chars → 400
- [ ] Question must not be only whitespace → 400
- [ ] session_id if provided must be valid UUID format → 400
- [ ] SQL injection in question field: "'; DROP TABLE leads; --" → treated as plain text question, validator catches any generated SQL issues
- [ ] Very long questions that would overflow LLM context: truncate at 500 chars before sending

**Acceptance tests:**
```
[TEST] POST /chat { question: "" }                           → 400
[TEST] POST /chat { question: "   " }                        → 400
[TEST] POST /chat { question: "a".repeat(501) }              → 400
[TEST] POST /chat { session_id: "not-a-uuid" }               → 400
[TEST] POST /chat { question: "'; DROP TABLE leads; --" }
       → 200, response explains it couldn't understand the question
       → leads table still exists: SELECT COUNT(*) FROM leads → 500
```

**PHASE 5 CHECKPOINT ✓**
```
[TEST] All logs are structured JSON
[TEST] Redis down → service degrades gracefully
[TEST] Invalid inputs return proper 400 errors
[TEST] No unhandled panics (run 50 random questions)
[TEST] go vet ./... → no issues
[TEST] go test -race ./... → no race conditions
```

---

## PHASE 6 — Frontend (Next.js + TypeScript)
**Goal:** Working browser UI. Google login, chat interface, SSE streaming, conversation history, feedback.
**Time estimate:** 4–5 days

---

### TASK-036 — Next.js project setup

- [ ] `npx create-next-app@latest frontend --typescript --tailwind --app`
- [ ] Install: `@tanstack/react-query`, `zustand`, `eventsource-parser`
- [ ] Create `frontend/Dockerfile`: build stage + nginx serve
- [ ] Configure `NEXT_PUBLIC_API_URL` from env
- [ ] `GET /` → redirect to `/login` if no access token in Zustand store

---

### TASK-037 — Auth store + token management

- [ ] Create `store/auth.ts` (Zustand)
  - State: `{ accessToken: string | null, user: User | null }`
  - `setAuth(token, user)`, `clearAuth()`
- [ ] On app load (`app/layout.tsx`): call `POST /auth/refresh` → if succeeds, populate store
- [ ] Axios/fetch interceptor: attach `Authorization: Bearer <token>` to all API calls
- [ ] On 401 response: attempt refresh → if refresh fails → redirect to /login

**Acceptance tests:**
```
[TEST] Hard refresh page → access token refreshed from cookie automatically
[TEST] Open new tab → still logged in (refresh from cookie)
[TEST] Expire JWT (set 1s expiry in test) → 401 → auto-refresh → retries original request
[TEST] Logout → hard refresh → redirected to /login (cookie cleared)
```

---

### TASK-038 — Login page

- [ ] `/login` page: Aria logo, "Continue with Google" button, clean centred layout
- [ ] Button click → `window.location.href = '/api/auth/google'` (proxied to Go backend)
- [ ] After OAuth callback → Go backend redirects to `/?token=<jwt>` or sets cookie + redirects
- [ ] Login page reads token from URL param, stores in Zustand, redirects to `/chat`
- [ ] If already authenticated → redirect to `/chat` immediately

**Acceptance tests:**
```
[TEST] Visit /login → Google button visible
[TEST] Click "Continue with Google" → redirects to Google consent
[TEST] After consent → lands on /chat, user avatar visible in header
[TEST] Visit /login while logged in → redirected to /chat immediately
```

---

### TASK-039 — Chat interface

- [ ] `/chat` page layout: sidebar (conversation history) + main chat area
- [ ] Main chat area:
  - Message list: user messages right-aligned, Aria messages left-aligned
  - "View SQL" collapsible panel on every Aria response (shows generated SQL)
  - Thumbs up / thumbs down buttons on every Aria response
  - Typing indicator (animated dots) while streaming
  - Auto-scroll to latest message
- [ ] Input bar: textarea (Enter sends, Shift+Enter newlines), send button, disabled while streaming
- [ ] "New Chat" button: clears messages, calls `DELETE /conversations/:id`
- [ ] Header: user avatar, full_name, logout button

**Acceptance tests:**
```
[TEST] Type question → Enter → user message appears → typing indicator → tokens stream in
[TEST] "View SQL" toggle → shows/hides generated SQL
[TEST] Thumbs up → button highlighted → POST /messages/:id/feedback called
[TEST] Thumbs down → modal asks "what was wrong?" → submit → feedback sent
[TEST] New Chat → messages cleared → next question starts new conversation
[TEST] Page refresh → conversation history reloaded from GET /conversations
```

---

### TASK-040 — Conversation history sidebar

- [ ] Sidebar shows list of past conversations (title, last_message_at)
- [ ] Click conversation → loads messages via `GET /conversations/:id/messages`
- [ ] Active conversation highlighted
- [ ] "New Chat" creates a new entry in sidebar when first message sent
- [ ] Sidebar collapses on mobile

**Acceptance tests:**
```
[TEST] After 3 conversations → sidebar shows 3 items
[TEST] Click past conversation → messages load
[TEST] Active conversation highlighted in sidebar
[TEST] New Chat → new conversation appears at top of sidebar after first message
```

---

### TASK-041 — SSE hook

- [ ] Create `hooks/useChat.ts` with full SSE implementation from spec.md section 9.1
- [ ] Handle all event types: `sql`, `token`, `done`, `error`
- [ ] On `error` event: display error message in chat, re-enable input
- [ ] On network disconnect mid-stream: show "Connection lost, please try again"
- [ ] Cached responses stream smoothly (chunks of chars, not all at once)

**Acceptance tests:**
```
[TEST] Slow network simulation: tokens arrive one-by-one, UI updates each
[TEST] Cached response: appears quickly but still streams (not instant dump)
[TEST] Kill docker compose api mid-response: "Connection lost" shown in UI
[TEST] SQL event arrives before tokens: "View SQL" button appears before answer
```

---

### TASK-042 — Final integration test

Run through the complete user journey from zero:

- [ ] `docker compose down -v && docker compose up`
- [ ] `make seed`
- [ ] `make schema-pipeline`
- [ ] Open `http://localhost:3000`
- [ ] Login with Google → lands on chat
- [ ] Ask: "What are my high priority leads?"
  - [ ] Streams real answer with real data
  - [ ] "View SQL" shows the generated query
  - [ ] Thumbs up
- [ ] Ask: "Which of those haven't been contacted in 3 days?" (follow-up, uses memory)
  - [ ] Answer correctly references the same leads
- [ ] Ask: "How many bookings have I closed this month?"
  - [ ] Returns a count
- [ ] Ask: "What's the weather today?" (out of scope)
  - [ ] Politely declines
- [ ] Ask same question again
  - [ ] Cached response badge visible, instant reply
- [ ] New Chat → start fresh
- [ ] Logout → old JWT rejected on next request
- [ ] All 150 tests passing: `go test ./...`

**PHASE 6 CHECKPOINT ✓**
```
[TEST] Full user journey above passes end to end
[TEST] Two browser sessions with different agents → each sees only their data
[TEST] Mobile viewport → sidebar collapses, chat still usable
[TEST] Network tab in DevTools → SSE visible as stream, not one response
[TEST] go test ./... → all tests pass
[TEST] No console errors in browser
```

---

## Test Summary

| Phase | Unit Tests | Integration Tests | Manual Tests |
|---|---|---|---|
| 0 — Infrastructure | Config, DB pool, Redis | Docker health | Seed data counts |
| 1 — Auth | JWT sign/verify, middleware | OAuth callback | Full Google login |
| 2 — Schema Pipeline | Hash, embed, similarity | Full pipeline run | Incremental re-run |
| 3 — AI Core | SQL validator, executor, cache | Full query pipeline | 5 manual questions |
| 4 — Chat API | Handlers, rate limit, feedback | SSE stream, isolation | curl streaming test |
| 5 — Reliability | Edge cases, input validation | Degradation paths | Redis down test |
| 6 — Frontend | Hook unit tests | Full E2E journey | Two-agent isolation |

---

## Running All Tests

```bash
# Go tests
go test ./... -v -race

# Specific package
go test ./internal/ai/... -v
go test ./internal/auth/... -v
go test ./internal/cache/... -v

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Lint
golangci-lint run ./...

# Python pipeline tests
cd schema_pipeline && python -m pytest tests/ -v
```

---

## Definition of Done (per task)

A task is DONE when:
1. All `[AUTO]` tests written and passing
2. All `[TEST]` manual verifications confirmed
3. No `TODO` comments left in the code for that task
4. Code compiles with `go build ./...`
5. `go vet ./...` shows no issues for touched packages
6. Structured log output is correct for new code paths
