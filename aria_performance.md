# p99 Latency / RPS / Throughput — Aria Performance Guide

## What these terms actually mean before anything else

**Latency** — how long a single request takes from the moment it arrives to the moment the first byte of response leaves the server. Not when the full response finishes — when it *starts*.

**p99 latency** — if you take 100 requests and sort them by how long they took, p99 is the 99th slowest one. In other words: 99% of your requests are faster than this number. The 1% that are slower are your worst-case users. p99 is what SLAs are written against. Not average, not median — p99. Average lies. A system where 95 requests take 10ms and 5 take 10 seconds has an "average" of ~600ms which sounds fine but is catastrophically broken.

**RPS (Requests Per Second)** — how many requests your server can handle per second at steady state without falling over. This is throughput at the server level.

**Throughput** — the volume of work completed per unit of time. For Aria specifically this has two dimensions: HTTP requests per second (RPS) and tokens streamed per second (for the SSE endpoints). A request that streams 300 tokens is fundamentally different work than a request that returns a cached 3-word answer.

---

## Aria's request types and their natural latency profiles

Before benchmarking you need to understand that Aria has three completely different request types with completely different latency characteristics. Benchmarking them as one number is meaningless.

```
Request Type 1 — Cache Hit
  Path:  JWT verify → Redis GET (1ms) → stream cached answer
  Expected p99: 80–120ms
  Bottleneck: Redis network round trip

Request Type 2 — Cache Miss, simple question
  Path:  JWT verify → Redis MISS → pgvector search (2 queries) →
         GPT-4o tool call → SQL execute → GPT-4o stream → Redis SET
  Expected p99: 1500–2500ms
  Bottleneck: OpenAI API latency (1–1.5s)

Request Type 3 — Cache Miss, complex join question
  Path:  Same as above + multi-table SQL → more LLM context
  Expected p99: 2500–4000ms
  Bottleneck: OpenAI API latency + SQL execution
```

A p99 of 2000ms is excellent for Request Type 2. A p99 of 2000ms for Request Type 1 means your Redis is on fire.

---

## Tools you will use

**k6** — load testing. Write JavaScript scenarios. Runs from CLI. Free, open source.

**wrk** — raw HTTP benchmarking. Simpler than k6. Good for baseline RPS numbers.

**hey** — even simpler. Good for quick one-liners.

**Grafana + Prometheus** — visualise latency over time under load. Not required for POC but the production approach.

**Go's built-in `pprof`** — CPU and memory profiling when you need to find where time is actually spent.

Install:
```bash
brew install k6         # macOS
brew install wrk
go install github.com/rakyll/hey@latest

# Or via Docker (no install needed)
docker run --rm grafana/k6 version
```

---

## Step 1 — Establish baselines before any load

Run these before any load test so you have a clean comparison point.

```bash
# 1. Health endpoint — your server's raw overhead with no business logic
hey -n 1000 -c 10 http://localhost:8080/health

# 2. Auth refresh — Redis read only
hey -n 500 -c 10 \
  -H "Cookie: refresh_token=<your_token>" \
  http://localhost:8080/auth/refresh

# 3. Conversations list — PostgreSQL read, no AI
hey -n 500 -c 10 \
  -H "Authorization: Bearer <jwt>" \
  http://localhost:8080/conversations
```

Expected output from `hey`:
```
Summary:
  Total:        2.1234 secs
  Slowest:      0.0512 secs
  Fastest:      0.0021 secs
  Average:      0.0098 secs
  Requests/sec: 470.50

Latency distribution:
  10% in 0.0031 secs
  25% in 0.0044 secs
  50% in 0.0087 secs
  75% in 0.0132 secs
  90% in 0.0201 secs
  95% in 0.0287 secs
  99% in 0.0412 secs    ← this is your p99
```

---

## Step 2 — k6 load test scripts

### Script 1 — Cache hit performance (what should be fast)

```javascript
// k6/cache_hit_test.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Rate } from 'k6/metrics';

const p99Latency = new Trend('p99_latency', true);
const cacheHitRate = new Rate('cache_hit_rate');

export const options = {
  stages: [
    { duration: '30s', target: 10 },   // ramp up to 10 users
    { duration: '1m',  target: 10 },   // hold at 10 users
    { duration: '30s', target: 50 },   // ramp up to 50
    { duration: '2m',  target: 50 },   // hold at 50 — this is your stress test
    { duration: '30s', target: 0 },    // ramp down
  ],
  thresholds: {
    http_req_duration: ['p(99)<200'],  // p99 must be under 200ms for cache hits
    http_req_failed: ['rate<0.01'],    // less than 1% errors
  },
};

const BASE_URL = 'http://localhost:8080';
const JWT = __ENV.JWT_TOKEN;

// Ask the same question repeatedly — after first request it will be cached
const CACHED_QUESTION = 'how many leads are assigned to me?';

export default function () {
  const res = http.post(
    `${BASE_URL}/chat`,
    JSON.stringify({
      question: CACHED_QUESTION,
      session_id: `session-${__VU}`,   // VU = virtual user ID
    }),
    {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${JWT}`,
      },
      timeout: '10s',
    }
  );

  // Parse SSE response to check if it was cached
  const wasCached = res.body.includes('"cached":true');
  cacheHitRate.add(wasCached);

  check(res, {
    'status is 200': (r) => r.status === 200,
    'has SSE data': (r) => r.body.includes('data:'),
    'no error event': (r) => !r.body.includes('"type":"error"'),
  });

  p99Latency.add(res.timings.duration);

  sleep(1);  // 1 second between requests per virtual user
}

export function handleSummary(data) {
  console.log('=== CACHE HIT PERFORMANCE ===');
  console.log(`p50: ${data.metrics.http_req_duration.values['p(50)']}ms`);
  console.log(`p90: ${data.metrics.http_req_duration.values['p(90)']}ms`);
  console.log(`p99: ${data.metrics.http_req_duration.values['p(99)']}ms`);
  console.log(`RPS: ${data.metrics.http_reqs.values.rate}`);
  console.log(`Cache hit rate: ${data.metrics.cache_hit_rate.values.rate * 100}%`);
}
```

Run it:
```bash
JWT_TOKEN="your_jwt_here" k6 run k6/cache_hit_test.js
```

---

### Script 2 — LLM query performance (realistic workload)

```javascript
// k6/llm_query_test.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend } from 'k6/metrics';

const timeToFirstToken = new Trend('time_to_first_token', true);
const totalStreamDuration = new Trend('total_stream_duration', true);

export const options = {
  // Don't hammer the LLM — OpenAI has rate limits
  // Real world: agents don't ask 100 questions per second
  stages: [
    { duration: '30s', target: 5 },   // 5 concurrent users
    { duration: '2m',  target: 5 },   // hold — observe LLM latency under concurrency
    { duration: '30s', target: 15 },  // push to 15 concurrent
    { duration: '1m',  target: 15 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    // LLM requests: p99 under 5 seconds (first token, not full response)
    'time_to_first_token': ['p(99)<5000'],
    http_req_failed: ['rate<0.05'],   // allow up to 5% errors (OpenAI timeouts)
  },
};

// Varied questions to avoid cache hits — test real LLM path
const QUESTIONS = [
  'what are my high priority leads?',
  'which leads have I not contacted this week?',
  'how many bookings did I close this month?',
  'show me recent activity on my leads',
  'what tasks are overdue?',
  'leads going to Manchester vs Leeds',
  'breakdown of my leads by state',
  'which of my leads are in the qualified stage?',
];

export default function () {
  const question = QUESTIONS[Math.floor(Math.random() * QUESTIONS.length)];
  const sessionId = `load-test-vu-${__VU}`;

  const startTime = new Date().getTime();

  const res = http.post(
    'http://localhost:8080/chat',
    JSON.stringify({ question, session_id: sessionId }),
    {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${__ENV.JWT_TOKEN}`,
      },
      timeout: '30s',
    }
  );

  // Approximate time to first token — when we got the first SSE data event
  // (k6 doesn't do true streaming measurement, but response time is a proxy)
  const endTime = new Date().getTime();

  check(res, {
    'status 200':    (r) => r.status === 200,
    'has answer':    (r) => r.body.includes('"type":"token"'),
    'has done event':(r) => r.body.includes('"type":"done"'),
    'no error':      (r) => !r.body.includes('"type":"error"'),
    'has sql':       (r) => r.body.includes('"type":"sql"'),
  });

  timeToFirstToken.add(endTime - startTime);
  totalStreamDuration.add(res.timings.duration);

  // Realistic think time between questions
  sleep(Math.random() * 3 + 2);  // 2–5 seconds between questions
}
```

---

### Script 3 — Spike test (what happens when 100 agents log in at 9am)

```javascript
// k6/spike_test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 1 },    // baseline
    { duration: '10s', target: 100 },  // sudden spike — 100 agents at once
    { duration: '3m',  target: 100 },  // hold the spike
    { duration: '10s', target: 1 },    // recovery
    { duration: '1m',  target: 1 },    // observe recovery
  ],
  thresholds: {
    http_req_duration: ['p(99)<10000'], // even during spike, p99 under 10s
    http_req_failed:   ['rate<0.1'],    // under 10% errors during spike
  },
};

export default function () {
  // During spike test use health + conversations only
  // Don't spike the OpenAI API — that would just test their rate limiter
  const res = http.get('http://localhost:8080/conversations', {
    headers: { 'Authorization': `Bearer ${__ENV.JWT_TOKEN}` },
  });

  check(res, { 'status 200': (r) => r.status === 200 });
  sleep(1);
}
```

---

## Step 3 — What to measure and record

Run each test and record these numbers in a table. Run each test 3 times and take the median.

```
=== BASELINE (no load) ===
Endpoint          p50     p90     p99     RPS
/health            2ms     3ms     8ms    800+
/conversations     8ms    15ms    35ms    200+
/chat (cached)    45ms    80ms   120ms    100+
/chat (llm miss) 1.2s    1.8s    2.8s      5+

=== UNDER LOAD (50 concurrent users, cache hit test) ===
Endpoint          p50     p90     p99     RPS    Error rate
/chat (cached)   65ms   120ms   180ms    180+    <0.1%

=== UNDER LOAD (15 concurrent users, LLM test) ===
Endpoint          p50     p90     p99     RPS    Error rate
/chat (llm miss) 1.8s    2.9s    4.2s     12+    <2%
```

---

## Step 4 — Identify and fix the bottlenecks

### Finding where time goes with pprof

Add this to `cmd/api/main.go` — it starts a pprof HTTP server in development:

```go
import _ "net/http/pprof"

// In main(), before starting the main server:
go func() {
    log.Println("pprof listening on :6060")
    http.ListenAndServe(":6060", nil)
}()
```

While a load test is running:
```bash
# CPU profile — what is the Go process doing?
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profile — what is allocating?
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine dump — how many goroutines are active?
curl http://localhost:6060/debug/pprof/goroutine?debug=1
```

In the pprof interactive shell:
```
(pprof) top10        → top 10 functions by CPU usage
(pprof) web          → open flame graph in browser (requires graphviz)
(pprof) list Query   → show line-by-line time in the Query function
```

### Common bottlenecks and fixes for Aria

**Bottleneck: pgvector similarity search is slow (>50ms)**
```sql
-- Check if HNSW index is being used
EXPLAIN ANALYZE
SELECT description_text FROM schema_embeddings
ORDER BY embedding <=> '[...1536 floats...]'::vector LIMIT 5;

-- Look for "Index Scan using idx_schema_embeddings_embedding"
-- If you see "Seq Scan" instead, the index isn't being used
-- Fix: ensure the query uses the right operator class

-- Tune HNSW ef_search parameter for speed vs accuracy tradeoff
SET hnsw.ef_search = 40;  -- default 40, lower = faster, less accurate
```

**Bottleneck: pgx connection pool exhausted under load**
```go
// Symptom: requests queue waiting for a connection
// Log line: "pgx pool: connection acquire timeout"

// Fix: increase pool size in postgres.go
config, _ := pgxpool.ParseConfig(databaseURL)
config.MaxConns = 50           // was 25 — increase for higher concurrency
config.MinConns = 5            // keep warm connections ready
config.MaxConnIdleTime = 30 * time.Second
config.HealthCheckPeriod = 60 * time.Second
```

**Bottleneck: Redis cache miss rate too high**
```bash
# Check cache hit rate in Redis
redis-cli info stats | grep keyspace_hits
redis-cli info stats | grep keyspace_misses

# Hit rate = hits / (hits + misses)
# Below 60% means agents are asking many unique questions
# Fix: normalise questions more aggressively before cache key generation
```

**Bottleneck: OpenAI rate limits under concurrent load**
```
Symptom: HTTP 429 from OpenAI after ~10 concurrent LLM calls
Log: "openai: rate limit exceeded, retry after 5s"

Fix options:
1. Implement a token bucket rate limiter in front of OpenAI calls
2. Queue LLM requests with a worker pool (e.g. max 10 concurrent OpenAI calls)
3. Upgrade OpenAI tier (Tier 2+: 3500 RPM on gpt-4o)
```

---

## Step 5 — Add latency tracking to the application itself

Don't just measure from the outside. Track it internally and expose via `/admin/metrics`.

```go
// internal/ai/service.go — add timing to every step

func (s *QueryService) Query(ctx context.Context, req QueryRequest) (*QueryResult, error) {
    total := time.Now()

    // Step 1: cache check
    t := time.Now()
    cached, hit, _ := s.cache.GetCachedQuery(ctx, req.AgentID, req.Question)
    cacheCheckMs := time.Since(t).Milliseconds()

    if hit {
        log.Info().
            Int64("cache_check_ms", cacheCheckMs).
            Bool("cache_hit", true).
            Msg("query served from cache")
        return cached, nil
    }

    // Step 2: schema retrieval
    t = time.Now()
    schemaDocs, _ := s.retriever.RetrieveRelevantSchema(ctx, req.Question, 5)
    schemaRetrievalMs := time.Since(t).Milliseconds()

    // Step 3: SQL generation
    t = time.Now()
    sql, explanation, _ := s.textToSQL.GenerateSQL(ctx, schemaDocs, req)
    sqlGenMs := time.Since(t).Milliseconds()

    // Step 4: SQL execution
    t = time.Now()
    rows, execDuration, _ := s.executor.Execute(ctx, sql, req.AgentID)
    sqlExecMs := execDuration.Milliseconds()

    // Step 5: response formatting
    t = time.Now()
    answer, _ := s.formatter.Format(ctx, req.Question, rows)
    formatMs := time.Since(t).Milliseconds()

    totalMs := time.Since(total).Milliseconds()

    log.Info().
        Int64("total_ms", totalMs).
        Int64("cache_check_ms", cacheCheckMs).
        Int64("schema_retrieval_ms", schemaRetrievalMs).
        Int64("sql_generation_ms", sqlGenMs).
        Int64("sql_execution_ms", sqlExecMs).
        Int64("response_format_ms", formatMs).
        Int("row_count", len(rows)).
        Msg("query pipeline complete")

    // ... rest of function
}
```

Now every request logs a breakdown. Run the load test, then look at the logs:
```bash
docker compose logs api | grep "query pipeline complete" | jq '.sql_generation_ms' | sort -n | tail -5
```

---

## Step 6 — SLO targets to define and defend

An SLO (Service Level Objective) is a target you commit to. Define these for Aria before any production deployment:

| Metric | Target | Meaning |
|--------|--------|---------|
| Cache hit p99 latency | < 200ms | 99% of cached queries respond in under 200ms |
| LLM query p99 latency | < 5s | 99% of LLM queries stream first token in under 5s |
| Error rate | < 1% | Less than 1 in 100 requests fails |
| Availability | 99.5% | No more than ~3.6 hours downtime per month |
| SQL execution p99 | < 2s | 99% of DB queries complete in under 2s |
| Cache hit rate | > 40% | At least 40% of queries served from cache |

---

## Step 7 — Record final numbers for resume

After running all tests, fill this in and keep it. These are the numbers you cite in interviews and on your resume.

```
=== Aria Performance Benchmarks (recorded: <date>) ===

Environment: Local Docker Compose, Apple M2 / 16GB RAM
DB: PostgreSQL 16 with pgvector, 500 leads seeded
Cache: Redis 7, empty at test start

CACHE HIT PATH (50 concurrent users, 3 min sustained):
  p50:    48ms
  p90:    89ms
  p99:   142ms     ← "sub-150ms p99 on cache hits"
  RPS:   185 req/s
  Errors: 0.0%

LLM PATH (15 concurrent users, 3 min sustained):
  p50:   1.4s
  p90:   2.3s
  p99:   3.8s      ← "sub-4s p99 on LLM queries"
  RPS:   11 req/s
  Errors: 1.2%     ← OpenAI rate limits at this concurrency

SPIKE TEST (100 concurrent, 3 min):
  /conversations p99: 380ms
  /health p99:         12ms
  Error rate:          3.1%
  Recovery time:       ~15s after spike drops
```

These numbers become resume bullets:
- "Aria achieves sub-150ms p99 latency on cache hits at 50 concurrent users"
- "LLM query p99 under 4 seconds at 15 concurrent agents — bounded by OpenAI API latency"
- "Identified pgx connection pool as bottleneck at 50+ concurrent LLM queries — increased MaxConns from 25 to 50, reduced p99 by 340ms"

---

## Quick reference — the commands you'll use most

```bash
# Quick RPS test on health endpoint
hey -n 5000 -c 50 http://localhost:8080/health

# k6 run with env var
JWT_TOKEN="your_jwt" k6 run --vus 10 --duration 60s k6/cache_hit_test.js

# Watch Redis stats live
watch -n 1 'redis-cli info stats | grep -E "hits|misses|connected"'

# Watch PostgreSQL active connections
watch -n 2 'psql $DATABASE_URL -c "SELECT count(*) FROM pg_stat_activity WHERE state='"'"'active'"'"'"'

# Live log stream during test
docker compose logs -f api | grep -E "total_ms|error|timeout"

# pprof flame graph (requires graphviz)
go tool pprof -http=:8090 http://localhost:6060/debug/pprof/profile?seconds=30
```
