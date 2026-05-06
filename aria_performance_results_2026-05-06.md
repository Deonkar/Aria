# Aria Performance Benchmarks (recorded: 2026-05-06)

These results were produced by following the workflow in `aria_performance.md` on a local Windows machine using Docker Compose, `hey`, and `k6` (via Docker).

## Environment

- **OS**: Windows 10 (PowerShell)
- **Runtime**: Docker Desktop
- **Stack**: `docker compose up -d --build` (Postgres + Redis + API)
- **Auth for tests**: `POST /auth/demo-token` (requires `ALLOW_DEMO_AUTH=true`)
- **Dataset**: `db/demo_agent_seed.sql` applied

## Notes / caveats

- **k6 in Docker cannot reach `localhost` on the host**. For k6 runs, `BASE_URL` was set to `http://host.docker.internal:8080`.
- **YesAPI embeddings**: API logs show the embeddings endpoint returning `404 Not Found` for `OPENAI_BASE_URL=https://api.yepapi.com/v1/ai`, so schema retrieval fell back to `information_schema`. This likely affects “LLM path” numbers.
- `/chat` is SSE streaming; for repeatable benchmarking I used `/query` for load tests because it’s unary JSON and exercises the same cache + SQL execution path as chat (minus SSE streaming overhead).

## Baseline (no sustained load)

### `/health` (hey)

Command:

```bash
hey -n 1000 -c 10 http://localhost:8080/health
```

Results:

- **RPS**: 3909.42 req/s
- **Latency**:
  - p50: 1.9ms
  - p90: 3.1ms
  - p99: 25.2ms

### `/conversations` (hey, JWT)

Command:

```bash
hey -n 500 -c 10 -H "Authorization: Bearer <JWT>" http://localhost:8080/conversations
```

Results:

- **RPS**: 1991.83 req/s
- **Latency**:
  - p50: 2.9ms
  - p90: 6.9ms
  - p99: 43.4ms

## Cache-hit path

### `/query` (hey, warmed cache)

Warm-up (one request to populate cache):

```bash
curl -X POST http://localhost:8080/query \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"question":"how many leads are assigned to me?","session_id":"<uuid>"}'
```

Load test:

```bash
hey -n 500 -c 10 -m POST \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -D ./aria_query_body.json \
  http://localhost:8080/query
```

Results:

- **RPS**: 1222.73 req/s
- **Latency**:
  - p50: 5.9ms
  - p90: 10.3ms
  - p99: 64.3ms

### `/query` (k6, 50 VUs)

Script: `k6/cache_hit_query_test.js`

Command:

```bash
docker run --rm \
  -e BASE_URL="http://host.docker.internal:8080" \
  -e JWT_TOKEN="<JWT>" \
  -v "%CD%\\k6:/scripts" \
  grafana/k6 run /scripts/cache_hit_query_test.js
```

Results:

- **RPS**: 28.52 req/s (script includes `sleep(1)` per iteration)
- **Errors**: 0.00%
- **Latency** (`http_req_duration`):
  - p50 (med): 7.32ms
  - p90: 10.38ms
  - p95: 12.57ms
  - p99: 22.49ms

## “LLM-ish” path (mixed cache misses/hits)

### `/query` (k6, up to 5 VUs)

Script: `k6/llm_query_test.js`

Command:

```bash
docker run --rm \
  -e BASE_URL="http://host.docker.internal:8080" \
  -e JWT_TOKEN="<JWT>" \
  -v "%CD%\\k6:/scripts" \
  grafana/k6 run /scripts/llm_query_test.js
```

Results:

- **RPS**: 1.27 req/s (script includes `sleep(2)` “think time”)
- **Errors**: 1.11% (1/90 requests failed during this run)
- **Latency** (`http_req_duration`):
  - median: 7.53ms (many requests were cache hits)
  - p95: 3.14s
  - p99: 7.47s

## Quick summary table

| Endpoint / test | Tool | Concurrency | RPS | p50 | p90 | p99 | Error rate |
|---|---:|---:|---:|---:|---:|---:|---:|
| `/health` baseline | hey | c=10 | 3909 | 1.9ms | 3.1ms | 25.2ms | 0% |
| `/conversations` baseline | hey | c=10 | 1992 | 2.9ms | 6.9ms | 43.4ms | 0% |
| `/query` cache-hit | hey | c=10 | 1223 | 5.9ms | 10.3ms | 64.3ms | 0% |
| `/query` cache-hit | k6 | 50 VUs | 28.5 | 7.3ms | 10.4ms | 22.5ms | 0% |
| `/query` mixed (LLM-ish) | k6 | 5 VUs | 1.27 | 7.5ms | 12.4ms | 7.47s | 1.11% |

