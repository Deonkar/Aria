// Smoketest exercises OpenAI + DB + Redis against a running stack (uses API spend).
// Run from repo root (loads .env) or via Docker on compose network, e.g.:
//
//	docker run --rm --network ai_agent_default -v "$PWD/aria:/app" -w /app \
//	  --env-file .env \
//	  -e DATABASE_URL=postgres://aria:aria@postgres:5432/aria?sslmode=disable \
//	  -e DATABASE_URL_READONLY=postgres://aria_readonly:aria_ro@postgres:5432/aria?sslmode=disable \
//	  -e REDIS_URL=redis://redis:6379/0 \
//	  golang:1.22-alpine sh -c "go run ./cmd/smoketest"
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Deonkar/Aria/aria/internal/ai"
	"github.com/Deonkar/Aria/aria/internal/auth"
	"github.com/Deonkar/Aria/aria/internal/cache"
	"github.com/Deonkar/Aria/aria/internal/config"
	"github.com/Deonkar/Aria/aria/internal/db"
	"github.com/google/uuid"
)

const seededAgentID = "10000000-0000-0000-0000-000000000001"

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	if strings.Contains(cfg.OpenAIAPIKey, "placeholder") || cfg.OpenAIAPIKey == "" {
		fmt.Fprintf(os.Stderr, "set a real OPENAI_API_KEY in .env (not placeholder)\n")
		os.Exit(1)
	}

	client := ai.NewOpenAIClient(cfg)

	fmt.Println("== 1) Embeddings (optional — some providers omit this endpoint) ==")
	vec, pt, ct, err := ai.EmbedText(ctx, client, cfg.OpenAIEmbedModel, "CRM leads and tasks for student housing")
	if err != nil {
		fmt.Printf("WARN: embed skipped: %v\n", err)
	} else {
		fmt.Printf("OK: vector dim=%d prompt_tokens=%d total_tokens=%d\n", len(vec), pt, ct)
	}

	roPool, err := db.NewPool(ctx, cfg.DatabaseURLReadonly)
	if err != nil {
		fmt.Fprintf(os.Stderr, "readonly db: %v\n", err)
		os.Exit(1)
	}
	defer roPool.Close()

	rdb, err := cache.NewClient(cfg.RedisURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "redis: %v\n", err)
		os.Exit(1)
	}
	defer rdb.Close()
	if err := rdb.FlushDB(ctx).Err(); err != nil {
		fmt.Fprintf(os.Stderr, "redis flush: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n== 2) Schema retrieval (pgvector, may be empty) ==")
	docs, err := ai.RetrieveRelevantSchema(ctx, roPool, client, cfg.OpenAIEmbedModel, "show my high priority leads", 3)
	if err != nil {
		fmt.Fprintf(os.Stderr, "schema retrieve: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("OK: %d schema chunk(s)\n", len(docs))

	svc := ai.NewQueryService(cfg, client, roPool, rdb)
	sess := uuid.NewString()

	fmt.Println("\n== 3) Query pipeline: lead count (cache miss) ==")
	res1, err := svc.Query(ctx, ai.QueryRequest{
		Question:  "How many leads are assigned to me? Reply with just the number in your summary.",
		SessionID: sess,
		AgentID:   seededAgentID,
		AgentRole: "agent",
		AgentName: "Agent One",
		Timezone:  "Asia/Kolkata",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "query 1: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("OK: was_cached=%v row_count=%d exec_ms=%d tokens_in=%d tokens_out=%d sql_snip=%q\n",
		res1.WasCached, res1.RowCount, res1.ExecutionMs, res1.TokensIn, res1.TokensOut, truncateSQL(res1.SQL, 120))
	if len(res1.Answer) == 0 {
		fmt.Fprintf(os.Stderr, "empty answer\n")
		os.Exit(1)
	}
	fmt.Printf("Answer preview: %.200s...\n", res1.Answer)

	fmt.Println("\n== 4) Same question (expect cache hit) ==")
	res2, err := svc.Query(ctx, ai.QueryRequest{
		Question:  "How many leads are assigned to me? Reply with just the number in your summary.",
		SessionID: sess,
		AgentID:   seededAgentID,
		AgentRole: "agent",
		AgentName: "Agent One",
		Timezone:  "Asia/Kolkata",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "query 2: %v\n", err)
		os.Exit(1)
	}
	if !res2.WasCached {
		fmt.Fprintf(os.Stderr, "expected cache hit on second identical question\n")
		os.Exit(1)
	}
	fmt.Printf("OK: was_cached=%v\n", res2.WasCached)

	fmt.Println("\n== 5) Out of scope ==")
	res3, err := svc.Query(ctx, ai.QueryRequest{
		Question:  "What is the weather in Paris today?",
		SessionID: uuid.NewString(),
		AgentID:   seededAgentID,
		AgentRole: "agent",
		AgentName: "Agent One",
		Timezone:  "Asia/Kolkata",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "query 3: %v\n", err)
		os.Exit(1)
	}
	if res3.RowCount != 0 {
		fmt.Fprintf(os.Stderr, "expected 0 rows for out-of-scope, got row_count=%d\n", res3.RowCount)
		os.Exit(1)
	}
	if len(res3.Answer) == 0 {
		fmt.Fprintf(os.Stderr, "empty answer for out-of-scope\n")
		os.Exit(1)
	}
	fmt.Printf("OK: row_count=0 sql_snip=%q answer preview: %.200s...\n", truncateSQL(res3.SQL, 80), res3.Answer)

	apiBase := strings.TrimSpace(os.Getenv("SMOKETEST_API_URL"))
	if apiBase == "" {
		fmt.Println("\n== 6) HTTP /query (skip: set SMOKETEST_API_URL e.g. http://api:8080) ==")
		fmt.Println("All in-process checks passed.")
		return
	}

	fmt.Println("\n== 6) HTTP POST /query via running API ==")
	jti := uuid.NewString()
	token, err := auth.SignToken(seededAgentID, "agent01@example.com", "agent", jti, cfg.JWTSecret, 2*time.Hour)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sign jwt: %v\n", err)
		os.Exit(1)
	}
	body := map[string]string{
		"question":   "List 3 of my leads with highest priority (limit 3).",
		"session_id": uuid.NewString(),
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(apiBase, "/")+"/query", bytes.NewReader(raw))
	if err != nil {
		fmt.Fprintf(os.Stderr, "http req: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http do: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode, string(b))
		os.Exit(1)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		fmt.Fprintf(os.Stderr, "json: %v body=%s\n", err, string(b))
		os.Exit(1)
	}
	fmt.Printf("OK: HTTP /query answer_preview=%.120s row_count=%v was_cached=%v\n",
		out["answer"], out["row_count"], out["was_cached"])
	fmt.Println("\nAll checks passed (in-process + HTTP).")
}

func truncateSQL(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
