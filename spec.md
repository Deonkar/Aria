# spec.md
# Aria — Technical Specification
### Architecture, system design, component contracts, and implementation decisions

---

## 1. System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              BROWSER                                        │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │   Next.js + TypeScript (port 3000)                                  │   │
│   │                                                                     │   │
│   │  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐   │   │
│   │  │  Login Page  │  │  Chat UI     │  │  Conversation History  │   │   │
│   │  │  Google SSO  │  │  SSE Stream  │  │  Feedback Buttons      │   │   │
│   │  └──────────────┘  └──────────────┘  └────────────────────────┘   │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│           │ HTTPS REST + SSE           │ httpOnly Cookie (refresh token)    │
└───────────┼────────────────────────────┼─────────────────────────────────────┘
            │                            │
┌───────────▼────────────────────────────▼─────────────────────────────────────┐
│                         Go API Server — Chi Router (port 8080)               │
│                                                                              │
│  ┌──────────────┐  ┌───────────────┐  ┌──────────────┐  ┌───────────────┐  │
│  │ Auth Handler │  │  Chat Handler │  │ Feedback     │  │ Admin Handler │  │
│  │ Google OAuth │  │  SSE Stream   │  │ Handler      │  │ Metrics       │  │
│  └──────┬───────┘  └───────┬───────┘  └──────┬───────┘  └───────┬───────┘  │
│         │                  │                  │                  │          │
│  ┌──────▼──────────────────▼──────────────────▼──────────────────▼───────┐  │
│  │                        Middleware Stack                               │  │
│  │   JWT Auth → Rate Limiter (Redis) → Request Logger → CORS           │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
│  ┌───────────────────┐  ┌────────────────────┐  ┌─────────────────────┐   │
│  │  AI Service       │  │  Query Executor    │  │  Session Manager    │   │
│  │  (text-to-SQL)    │  │  (read-only pgx)   │  │  (Redis)            │   │
│  └─────────┬─────────┘  └─────────┬──────────┘  └──────────┬──────────┘   │
│            │                      │                         │              │
└────────────┼──────────────────────┼─────────────────────────┼──────────────┘
             │                      │                         │
    ┌────────▼───────┐    ┌─────────▼──────────┐    ┌────────▼────────┐
    │  OpenAI API    │    │  PostgreSQL         │    │  Redis          │
    │  gpt-4o        │    │  (read-only role)   │    │  Cache+Sessions │
    │  embeddings    │    │  + pgvector         │    │  Rate limits    │
    └────────────────┘    └────────────────────┘    └─────────────────┘
             ▲
    ┌────────┴───────────────────────────────┐
    │  Python Schema Pipeline (one-time CLI) │
    │  Introspect DB → GPT-4o docs →         │
    │  Embed → store in schema_embeddings    │
    └────────────────────────────────────────┘
```

---

## 2. Request Flow Diagrams

### 2.1 Google OAuth Login Flow

```
Browser                  Go API               Google OAuth          PostgreSQL
   │                        │                      │                     │
   │── GET /auth/google ───►│                      │                     │
   │                        │── redirect ─────────►│                     │
   │                        │   (consent screen)    │                     │
   │◄─ redirect to Google ──│                      │                     │
   │                        │                      │                     │
   │── login on Google ────────────────────────►  │                     │
   │◄─ redirect to /auth/callback?code=xxx ──────  │                     │
   │── GET /auth/callback ──►│                     │                     │
   │                        │── exchange code ────►│                     │
   │                        │◄─ id_token+profile ──│                     │
   │                        │                      │                     │
   │                        │── SELECT * FROM users WHERE google_id=? ──►│
   │                        │◄─ user row (or empty) ─────────────────── │
   │                        │                      │                     │
   │                        │── (if new) INSERT user ──────────────────►│
   │                        │                      │                     │
   │                        │── sign JWT (user_id, email, role, exp)     │
   │                        │── set refresh token in httpOnly cookie      │
   │◄─ { access_token } ────│                      │                     │
   │                        │                      │                     │
   │  store JWT in memory   │                      │                     │
   │  (NOT localStorage)    │                      │                     │
```

### 2.2 AI Query Flow (cache miss)

```
Browser          Go API           Redis         OpenAI           PostgreSQL
   │                │                │              │                 │
   │── POST /chat ──►│               │              │                 │
   │   { question,  │               │              │                 │
   │     session_id }│              │              │                 │
   │                │── check cache ►│              │                 │
   │                │◄─ MISS ────────│              │                 │
   │                │── validate SQL safety         │                 │
   │                │── load session history ──────►│ (Redis)         │
   │                │◄─ last 10 messages ───────────│                 │
   │                │                              │                 │
   │                │── retrieve relevant schema embeddings ────────►│
   │                │◄─ top-5 similar docs ────────────────────────  │
   │                │                              │                 │
   │                │── GPT-4o tool call ──────────►│                 │
   │                │   (question + schema context  │                 │
   │                │    + conversation history     │                 │
   │                │    + agent_id for scoping)    │                 │
   │                │◄─ { tool: query_db,           │                 │
   │                │     sql: "SELECT..." } ───────│                 │
   │                │                              │                 │
   │                │── validate SELECT-only        │                 │
   │                │── inject WHERE agent_id=$1    │                 │
   │                │── execute SQL ───────────────────────────────►│
   │                │◄─ rows ──────────────────────────────────────  │
   │                │                              │                 │
   │◄─ SSE: open ───│── GPT-4o format+stream ──────►│                 │
   │◄─ SSE: token ──│◄─ streamed tokens ─────────── │                 │
   │◄─ SSE: token ──│                              │                 │
   │◄─ SSE: done ───│                              │                 │
   │                │── cache result in Redis ─────►│                 │
   │                │── persist message to DB ────────────────────►  │
```

### 2.3 AI Query Flow (cache hit)

```
Browser          Go API           Redis
   │                │                │
   │── POST /chat ──►│               │
   │                │── check cache ►│
   │                │◄─ HIT ─────────│
   │◄─ SSE: open ───│                │
   │◄─ SSE: cached response (chunk) ─│ (Redis value streamed directly)
   │◄─ SSE: done ───│                │
   │                │ (< 100ms total)│
```

### 2.4 SSE Streaming Architecture (Go)

```
HTTP Handler (goroutine per request)
        │
        │ 1. Set headers:
        │    Content-Type: text/event-stream
        │    Cache-Control: no-cache
        │    X-Accel-Buffering: no
        │
        │ 2. Start OpenAI stream (returns <-chan ChatCompletionChunk)
        │
        ├── goroutine: pump channel → flush SSE events
        │      for chunk := range openaiStream {
        │          fmt.Fprintf(w, "data: %s\n\n", chunk.Text)
        │          w.(http.Flusher).Flush()
        │      }
        │
        │ 3. On channel close: send [DONE] event
        │ 4. Persist full message to DB
        │ 5. Cache in Redis
```

---

## 3. Go Project Structure

```
aria/
├── cmd/
│   └── api/
│       └── main.go                  # Entry point — wires everything
│
├── internal/
│   ├── auth/
│   │   ├── google.go                # Google OAuth handler
│   │   ├── jwt.go                   # JWT sign, verify, blacklist
│   │   └── middleware.go            # JWT middleware for Chi
│   │
│   ├── ai/
│   │   ├── service.go               # Orchestrates the full query pipeline
│   │   ├── text_to_sql.go           # GPT-4o tool calling → SQL generation
│   │   ├── sql_validator.go         # Ensure SELECT-only, no injections
│   │   ├── sql_executor.go          # pgx read-only query runner
│   │   ├── schema_retriever.go      # pgvector similarity search for schema
│   │   ├── response_formatter.go    # GPT-4o formats SQL results → text
│   │   └── streamer.go              # SSE token streaming via Go channels
│   │
│   ├── cache/
│   │   ├── redis.go                 # Redis client + helpers
│   │   ├── query_cache.go           # Cache get/set/invalidate for queries
│   │   └── session.go               # Conversation history in Redis
│   │
│   ├── handlers/
│   │   ├── chat.go                  # POST /chat — main AI endpoint
│   │   ├── feedback.go              # POST /messages/:id/feedback
│   │   ├── conversations.go         # GET /conversations
│   │   └── admin.go                 # GET /admin/metrics
│   │
│   ├── db/
│   │   ├── postgres.go              # pgx pool setup, read-only role
│   │   └── migrations/              # SQL migration files
│   │       ├── 001_create_tables.sql
│   │       └── 002_seed_data.sql
│   │
│   ├── models/
│   │   ├── user.go
│   │   ├── lead.go
│   │   ├── message.go
│   │   └── ...                      # One file per domain entity
│   │
│   └── config/
│       └── config.go                # Env var loading + validation
│
├── schema_pipeline/                 # Python — runs once
│   ├── run.py
│   ├── introspect.py                # Query information_schema
│   ├── document.py                  # GPT-4o generates descriptions
│   ├── embed.py                     # OpenAI embeddings → pgvector
│   ├── generate_examples.py         # Auto-generates Q→SQL pairs
│   └── requirements.txt
│
├── frontend/                        # Next.js + TypeScript
│   ├── app/
│   │   ├── login/page.tsx
│   │   ├── chat/page.tsx
│   │   └── layout.tsx
│   ├── components/
│   │   ├── ChatWindow.tsx
│   │   ├── MessageBubble.tsx        # Shows answer + "View SQL" toggle
│   │   ├── FeedbackButtons.tsx      # Thumbs up / down
│   │   └── SSEStream.tsx            # EventSource hook
│   └── hooks/
│       └── useChat.ts               # SSE connection + message state
│
├── docker-compose.yml
├── Makefile
├── .env.example
└── README.md
```

---

## 4. Go — Key Packages

| Package | Library | Why |
|---|---|---|
| HTTP router | `go-chi/chi v5` | Idiomatic Go, middleware chains, no magic |
| PostgreSQL | `jackc/pgx v5` | Native driver, no ORM, pgvector support |
| Redis | `go-redis/redis v9` | Mature, context-aware |
| OpenAI | `openai-go` (official) | Tool calling, streaming, structured outputs |
| JWT | `golang-jwt/jwt v5` | Signing and verification |
| Google OAuth | `golang.org/x/oauth2` | Standard Google OAuth2 flow |
| Env config | `joho/godotenv` | .env loading in development |
| Logging | `rs/zerolog` | Structured JSON logs, zero alloc |
| Hashing | `crypto/sha256` (stdlib) | Cache key generation |

---

## 5. API Contracts

### 5.1 Auth Endpoints

```
GET /auth/google
→ 302 redirect to Google consent URL

GET /auth/callback?code=xxx&state=yyy
→ 200 { access_token: "jwt...", user: { id, email, full_name, avatar_url, role } }
→ Sets httpOnly cookie: refresh_token

POST /auth/refresh
Body: (reads refresh_token cookie automatically)
→ 200 { access_token: "new_jwt..." }

POST /auth/logout
Headers: Authorization: Bearer <jwt>
→ 200 { message: "logged out" }
→ Blacklists JWT in Redis
```

### 5.2 Chat Endpoints

```
POST /chat
Headers: Authorization: Bearer <jwt>
Body: {
  "question": "What are my high priority leads today?",
  "session_id": "uuid",        // optional — omit to create new session
  "conversation_id": "uuid"   // optional — for continuing a conversation
}
→ SSE stream:
  data: {"type":"sql","sql":"SELECT ..."}        // generated SQL
  data: {"type":"token","text":"You have "}      // streamed token
  data: {"type":"token","text":"7 high "}
  data: {"type":"done","message_id":"uuid","cached":false,"row_count":7}
  data: [DONE]

→ Error: 400 if non-SELECT SQL attempted
→ Error: 422 if question is out of CRM scope
→ Error: 408 if SQL execution times out
→ Error: 429 if rate limit exceeded
```

### 5.3 Conversation Endpoints

```
GET /conversations
Headers: Authorization: Bearer <jwt>
→ 200 [{ id, title, message_count, started_at, last_message_at }]

GET /conversations/:id/messages
Headers: Authorization: Bearer <jwt>
→ 200 [{ id, role, content, generated_sql, was_cached, created_at }]

DELETE /conversations/:id
Headers: Authorization: Bearer <jwt>
→ 204 (marks ended_at, clears Redis session)
```

### 5.4 Feedback Endpoints

```
POST /messages/:id/feedback
Headers: Authorization: Bearer <jwt>
Body: {
  "is_helpful": false,
  "correction_note": "It showed all leads not just mine",
  "corrected_sql": "SELECT ... WHERE assigned_agent_id = ..."
}
→ 201 { id, message_id, is_helpful }
→ Side effect: invalidates Redis cache entry for this question
```

### 5.5 Admin Endpoints

```
GET /admin/metrics
Headers: Authorization: Bearer <jwt> (role=admin required)
→ 200 {
    "queries_today": 142,
    "avg_response_ms": 1840,
    "cache_hit_rate": 0.34,
    "top_question_types": ["lead_status", "task_overdue", "booking_count"],
    "feedback_score": 0.87,
    "intent_gaps_unresolved": 3
  }
```

---

## 6. AI Service — Internal Design

### 6.1 The Tool Calling Pattern

```go
// text_to_sql.go — simplified

func (s *AIService) GenerateSQL(ctx context.Context, req QueryRequest) (string, error) {
    tools := []openai.Tool{
        {
            Type: "function",
            Function: openai.Function{
                Name:        "query_crm_database",
                Description: "Execute a read-only SQL query against the CRM database",
                Parameters: map[string]any{
                    "type": "object",
                    "properties": map[string]any{
                        "sql": map[string]any{
                            "type":        "string",
                            "description": "A PostgreSQL SELECT query",
                        },
                        "explanation": map[string]any{
                            "type":        "string",
                            "description": "Plain English explanation of what this query does",
                        },
                    },
                    "required": []string{"sql", "explanation"},
                },
            },
        },
    }

    // Build messages: system + schema context + history + question
    messages := s.buildMessages(ctx, req)

    resp, err := s.openaiClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
        Model:    openai.F(openai.ChatModelGPT4o),
        Tools:    openai.F(tools),
        Messages: openai.F(messages),
    })
    // Extract tool call → SQL string
    return extractSQL(resp), nil
}
```

### 6.2 System Prompt Structure

```
You are Aria, an AI assistant for CRM agents at a student accommodation company.

AGENT CONTEXT:
- Agent ID: {agent_id}
- Agent name: {full_name}
- Role: {role}
- Current time: {current_time} ({timezone})

DATA ACCESS RULES:
- You MUST always filter queries by assigned_agent_id = '{agent_id}' 
  UNLESS the agent's role is 'admin'
- You MUST only generate SELECT queries
- You MUST NOT use subqueries that could expose other agents' data

DATABASE CONTEXT:
{retrieved_schema_docs}  ← top-5 from pgvector similarity search

EXAMPLE QUERIES:
{retrieved_intent_examples}  ← top-3 from pgvector similarity search

CONVERSATION HISTORY:
{last_10_messages}

If you cannot answer using the available tables, explain why clearly.
Never fabricate data. If a query returns 0 results, say so.
```

### 6.3 SQL Validator

```go
// sql_validator.go

func ValidateSQL(sql string) error {
    upper := strings.TrimSpace(strings.ToUpper(sql))

    // Must start with SELECT
    if !strings.HasPrefix(upper, "SELECT") {
        return ErrNotSelectStatement
    }

    // Block dangerous keywords
    forbidden := []string{
        "INSERT", "UPDATE", "DELETE", "DROP", "CREATE",
        "ALTER", "TRUNCATE", "EXEC", "EXECUTE",
        "--", "/*",  // comment-based injection
    }
    for _, keyword := range forbidden {
        if strings.Contains(upper, keyword) {
            return fmt.Errorf("forbidden keyword: %s", keyword)
        }
    }

    // Must contain agent_id filter (enforced before this by injection)
    if !strings.Contains(upper, "AGENT_ID") {
        return ErrMissingAgentFilter
    }

    return nil
}
```

### 6.4 Parallel Query Execution (Go goroutines)

When a question requires multiple SQL queries (e.g. "show my leads AND today's tasks"):

```go
// sql_executor.go

func (e *Executor) ExecuteParallel(ctx context.Context, queries []string) ([]QueryResult, error) {
    results := make([]QueryResult, len(queries))
    errs := make([]error, len(queries))

    var wg sync.WaitGroup
    for i, sql := range queries {
        wg.Add(1)
        go func(idx int, query string) {
            defer wg.Done()
            rows, err := e.pool.Query(ctx, query)
            results[idx] = parseRows(rows)
            errs[idx] = err
        }(i, sql)
    }
    wg.Wait()
    return results, combineErrors(errs)
}
```

---

## 7. Security Design

### 7.1 Data Isolation Layers

```
Layer 1 — PostgreSQL role (infrastructure)
  The app connects as 'aria_readonly' role
  GRANT SELECT ON ALL TABLES IN SCHEMA public TO aria_readonly
  No INSERT/UPDATE/DELETE possible at DB level

Layer 2 — SQL validator (application)
  Reject any non-SELECT before execution
  Runs BEFORE the query hits the DB

Layer 3 — Agent filter injection (application)
  After SQL is generated, inject WHERE assigned_agent_id = $1
  Agent can never see another agent's rows even if LLM fails to include filter

Layer 4 — JWT middleware (transport)
  Every request verifies JWT, extracts user_id
  user_id passed in context — handlers cannot bypass it
```

### 7.2 JWT Flow

```
Sign:     claims = { sub: user_id, email, role, exp: now+8h }
          token = jwt.Sign(claims, HS256, JWT_SECRET)

Verify:   jwt.Parse(token, JWT_SECRET)
          check exp, check not in Redis blacklist

Blacklist: SET jwt_blacklist:{jti} "" EX <remaining_ttl>
           check on every request: EXISTS jwt_blacklist:{jti}
```

### 7.3 Rate Limiting

```
Key:    rate_limit:{user_id}
Value:  INCR per request, EXPIRE after 60s window
Logic:  if GET rate_limit:{user_id} > 30 → 429 Too Many Requests
```

---

## 8. Schema Pipeline — Python Architecture

```
schema_pipeline/run.py

┌─────────────────────────────────────────────────┐
│ Step 1: Introspect                              │
│   Query information_schema.columns             │
│   Query information_schema.table_constraints   │
│   Build per-table DDL + FK map                 │
└───────────────────┬─────────────────────────────┘
                    │
┌───────────────────▼─────────────────────────────┐
│ Step 2: Hash check                              │
│   SHA256(table DDL) vs stored ddl_hash          │
│   Skip table if hash unchanged                  │
│   Only process new/changed tables               │
└───────────────────┬─────────────────────────────┘
                    │
┌───────────────────▼─────────────────────────────┐
│ Step 3: GPT-4o documentation                    │
│   For each changed table:                       │
│   Prompt: "Here is a PostgreSQL table DDL.     │
│   Generate: table purpose, column descriptions, │
│   enum value meanings, common query patterns." │
│   Output: structured JSON doc                   │
└───────────────────┬─────────────────────────────┘
                    │
┌───────────────────▼─────────────────────────────┐
│ Step 4: Embed + store                           │
│   OpenAI text-embedding-3-small per doc        │
│   UPSERT into schema_embeddings                │
└───────────────────┬─────────────────────────────┘
                    │
┌───────────────────▼─────────────────────────────┐
│ Step 5: Generate intent examples                │
│   GPT-4o: "Given this CRM schema, generate    │
│   20 example questions an agent would ask,     │
│   with the exact SQL to answer each."          │
│   Embed each question + UPSERT intent_examples │
└─────────────────────────────────────────────────┘
```

---

## 9. Frontend Architecture

### 9.1 SSE Chat Hook

```typescript
// hooks/useChat.ts

export function useChat(sessionId: string) {
  const [messages, setMessages] = useState<Message[]>([])
  const [isStreaming, setIsStreaming] = useState(false)

  const sendMessage = useCallback(async (question: string) => {
    // Add user message immediately
    setMessages(prev => [...prev, { role: 'user', content: question }])
    setIsStreaming(true)

    const response = await fetch('/api/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json',
                 'Authorization': `Bearer ${getToken()}` },
      body: JSON.stringify({ question, session_id: sessionId })
    })

    const reader = response.body!.getReader()
    const decoder = new TextDecoder()
    let assistantContent = ''
    let generatedSQL = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      const chunk = decoder.decode(value)

      // Parse SSE events
      for (const line of chunk.split('\n')) {
        if (!line.startsWith('data: ')) continue
        const data = JSON.parse(line.slice(6))

        if (data.type === 'sql') generatedSQL = data.sql
        if (data.type === 'token') {
          assistantContent += data.text
          // Update last message in real time
          setMessages(prev => updateLastAssistant(prev, assistantContent))
        }
        if (data.type === 'done') {
          setMessages(prev => finaliseMessage(prev, data.message_id, generatedSQL))
          setIsStreaming(false)
        }
      }
    }
  }, [sessionId])

  return { messages, sendMessage, isStreaming }
}
```

### 9.2 JWT Storage Strategy

```
NEVER store JWT in localStorage — XSS vulnerability.
Store access_token in memory only (React state / Zustand store).
Store refresh_token in httpOnly cookie (set by Go backend).

On page refresh → access_token is lost from memory.
On page load → call POST /auth/refresh → server reads httpOnly cookie →
  returns new access_token → store in memory.

This is the most secure client-side JWT pattern.
```

---

## 10. Caching Strategy

| What | Key Pattern | TTL | Invalidated by |
|---|---|---|---|
| Query result | `query_cache:{user_id}:{sha256(q)}` | 5 min | Thumbs down |
| Session history | `session:{user_id}:{session_id}` | 2 hours | New chat |
| JWT blacklist | `jwt_blacklist:{jti}` | Until JWT natural expiry | N/A |
| Rate limit counter | `rate_limit:{user_id}` | 60 seconds (rolling) | N/A |

---

## 11. Docker Compose Services

| Service | Image | Port | Notes |
|---|---|---|---|
| `postgres` | postgres:16 | 5432 | Runs init.sql + seed.sql on first start |
| `redis` | redis:7-alpine | 6379 | Persistence disabled for POC |
| `api` | ./aria (Go) | 8080 | Depends on postgres + redis healthy |
| `frontend` | ./frontend (Next.js) | 3000 | Depends on api |
| `schema-pipeline` | ./schema_pipeline (Python) | — | `restart: "no"`, runs once |

### Startup sequence
```
postgres (healthy) ─┐
                    ├──► api (healthy) ──► frontend
redis (healthy) ────┘
        │
        └──► schema-pipeline (runs once after postgres healthy)
```

---

## 12. Key Technical Decisions — Interview Ready

**Why Chi over Gin?**
Chi is closer to net/http — middleware is just `func(http.Handler) http.Handler`.
No struct tags, no magic, no framework-specific types leaking everywhere.
Gin is faster to prototype but Chi teaches idiomatic Go patterns.

**Why pgx over GORM?**
For a system where every query is AI-generated, you need to see and reason
about exact SQL. GORM's abstraction hides queries and makes debugging harder.
pgx forces explicit SQL which is exactly what this project is about.

**Why SSE over WebSockets?**
Streaming LLM responses is one-directional: server → client.
WebSockets are bidirectional — you'd use them if the client needed to push
mid-stream. SSE is simpler, auto-reconnects, and works over standard HTTP/2.
Go's `http.Flusher` interface makes SSE implementation clean.

**Why agent_id injection at Go layer not LLM layer?**
The LLM is non-deterministic. Trusting it to always include the agent filter
is a security risk. The Go service injects `AND assigned_agent_id = $agent_id`
into the generated SQL before execution — regardless of what the LLM produced.
This is defence in depth.

**Why pgvector for embeddings instead of Pinecone?**
We already have PostgreSQL. Adding a second database for vectors at POC scale
(< 10k vectors) has no performance benefit and adds operational overhead.
pgvector with HNSW index gives sub-10ms similarity search at this scale.
