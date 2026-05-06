package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Deonkar/Aria/aria/internal/cache"
	"github.com/Deonkar/Aria/aria/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openai/openai-go"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	QueryCacheTTL  = 5 * time.Minute
	SessionTTL     = 2 * time.Hour
	SessionMaxMsgs = 10
)

type QueryService struct {
	cfg      *config.Config
	client   openai.Client
	readPool *pgxpool.Pool
	rdb      *redis.Client
}

func NewQueryService(cfg *config.Config, client openai.Client, readPool *pgxpool.Pool, rdb *redis.Client) *QueryService {
	return &QueryService{cfg: cfg, client: client, readPool: readPool, rdb: rdb}
}

func (s *QueryService) Cfg() *config.Config { return s.cfg }

func (s *QueryService) RDB() *redis.Client { return s.rdb }

func (s *QueryService) historyBlock(ctx context.Context, req QueryRequest) string {
	hist, err := cache.GetHistory(ctx, s.rdb, req.AgentID, req.SessionID)
	if err != nil {
		log.Warn().Err(err).Str("user_id", req.AgentID).Msg("redis session unavailable")
		return ""
	}
	var b strings.Builder
	for _, m := range hist {
		fmt.Fprintf(&b, "%s: %s\n", m.Role, m.Content)
	}
	return b.String()
}

// Query runs the full non-streaming pipeline (TASK-027).
func (s *QueryService) Query(ctx context.Context, req QueryRequest) (*QueryResult, error) {
	if strings.TrimSpace(req.Question) == "" {
		return nil, fmt.Errorf("empty question")
	}

	if cr, ok, err := cache.GetCachedQuery(ctx, s.rdb, req.AgentID, req.Question); err == nil && ok {
		return &QueryResult{
			Answer:    cr.Answer,
			SQL:       cr.SQL,
			RowCount:  cr.RowCount,
			WasCached: true,
		}, nil
	} else if err != nil {
		log.Warn().Err(err).Msg("redis query cache unavailable, skipping cache read")
	}

	historyText := s.historyBlock(ctx, req)

	docs, err := RetrieveRelevantSchema(ctx, s.readPool, s.client, s.cfg.OpenAIEmbedModel, req.Question, 5)
	if err != nil {
		return nil, fmt.Errorf("schema retrieve: %w", err)
	}
	fallback := ""
	if len(docs) == 0 {
		var ferr error
		fallback, ferr = LoadInformationSchemaFallback(ctx, s.readPool)
		if ferr != nil {
			log.Warn().Err(ferr).Msg("schema fallback load failed")
		} else {
			log.Warn().Msg("schema_embeddings empty, using information_schema fallback")
		}
	}
	schemaBlock := SchemaContext(docs, fallback)

	examples, err := RetrieveIntentExamples(ctx, s.readPool, s.client, s.cfg.OpenAIEmbedModel, req.Question, 3)
	if err != nil {
		return nil, fmt.Errorf("intent examples: %w", err)
	}
	examplesBlock := IntentExamplesContext(examples)

	sqlText, expl, pt, ct, err := GenerateSQL(ctx, s.client, s.cfg, schemaBlock, examplesBlock, historyText, req)
	tokIn, tokOut := pt, ct
	if err != nil {
		if err == ErrOutOfScope {
			return &QueryResult{
				Answer:     expl,
				TokensIn:   tokIn,
				TokensOut:  tokOut,
				WasCached:  false,
			}, nil
		}
		return nil, err
	}

	if err := Validate(sqlText); err != nil {
		return nil, fmt.Errorf("sql validation: %w", err)
	}

	injected, err := InjectAgentFilter(sqlText, req.AgentID, req.AgentRole)
	if err != nil {
		return nil, err
	}

	rows, execDur, err := Execute(ctx, s.readPool, injected, execArgs(injected, req.AgentID)...)
	wasCorrected := false
	if err != nil {
		sql2, _, pt2, ct2, rerr := GenerateSQLWithRetry(ctx, s.client, s.cfg, schemaBlock, examplesBlock, historyText, req, injected, err.Error())
		tokIn += pt2
		tokOut += ct2
		if rerr != nil {
			return nil, fmt.Errorf("sql retry: %w (orig: %v)", rerr, err)
		}
		if err := Validate(sql2); err != nil {
			return nil, fmt.Errorf("retry validation: %w", err)
		}
		injected2, err := InjectAgentFilter(sql2, req.AgentID, req.AgentRole)
		if err != nil {
			return nil, err
		}
		rows, execDur, err = Execute(ctx, s.readPool, injected2, execArgs(injected2, req.AgentID)...)
		sqlText = sql2
		injected = injected2
		wasCorrected = true
		if err != nil {
			return nil, err
		}
	}

	answer, fpt, fct, err := FormatAnswer(ctx, s.client, s.cfg, req.Question, injected, rows)
	tokIn += fpt
	tokOut += fct
	if err != nil {
		return nil, fmt.Errorf("format answer: %w", err)
	}

	res := &QueryResult{
		Answer:       answer,
		SQL:          injected,
		RowCount:     len(rows),
		ExecutionMs:  execDur.Milliseconds(),
		WasCached:    false,
		WasCorrected: wasCorrected,
		TokensIn:     tokIn,
		TokensOut:    tokOut,
	}

	cr := &cache.CachedResult{
		Answer:      answer,
		SQL:         injected,
		RowCount:    len(rows),
		GeneratedAt: time.Now().UTC(),
	}
	if err := cache.SetCachedQuery(ctx, s.rdb, req.AgentID, req.Question, cr, QueryCacheTTL); err != nil {
		log.Warn().Err(err).Msg("redis query cache set failed")
	}

	_ = cache.AppendMessage(ctx, s.rdb, req.AgentID, req.SessionID, cache.SessionMessage{
		Role:    "user",
		Content: req.Question,
	}, SessionMaxMsgs, SessionTTL)
	_ = cache.AppendMessage(ctx, s.rdb, req.AgentID, req.SessionID, cache.SessionMessage{
		Role:         "assistant",
		Content:      answer,
		GeneratedSQL: injected,
	}, SessionMaxMsgs, SessionTTL)

	return res, nil
}

// GenerateExecuteAndFormat returns SQL, executed SQL, rows and timing for streaming (after SQL event).
func (s *QueryService) GenerateExecuteAndFormat(ctx context.Context, req QueryRequest) (
	sqlExecuted string,
	rows []map[string]any,
	execMs int64,
	wasCorrected bool,
	tokIn, tokOut int,
	outOfScopeAnswer string,
	err error,
) {
	historyText := s.historyBlock(ctx, req)

	docs, err := RetrieveRelevantSchema(ctx, s.readPool, s.client, s.cfg.OpenAIEmbedModel, req.Question, 5)
	if err != nil {
		return "", nil, 0, false, 0, 0, "", err
	}
	fallback := ""
	if len(docs) == 0 {
		fallback, _ = LoadInformationSchemaFallback(ctx, s.readPool)
		log.Warn().Msg("schema_embeddings empty, using information_schema fallback")
	}
	schemaBlock := SchemaContext(docs, fallback)

	examples, err := RetrieveIntentExamples(ctx, s.readPool, s.client, s.cfg.OpenAIEmbedModel, req.Question, 3)
	if err != nil {
		return "", nil, 0, false, 0, 0, "", err
	}
	examplesBlock := IntentExamplesContext(examples)

	sqlText, expl, pt, ct, err := GenerateSQL(ctx, s.client, s.cfg, schemaBlock, examplesBlock, historyText, req)
	tokIn, tokOut = pt, ct
	if err != nil {
		if err == ErrOutOfScope {
			return "", nil, 0, false, tokIn, tokOut, expl, ErrOutOfScope
		}
		return "", nil, 0, false, tokIn, tokOut, "", err
	}

	if err := Validate(sqlText); err != nil {
		return "", nil, 0, false, tokIn, tokOut, "", err
	}

	injected, err := InjectAgentFilter(sqlText, req.AgentID, req.AgentRole)
	if err != nil {
		return "", nil, 0, false, tokIn, tokOut, "", err
	}

	rows, execDur, err := Execute(ctx, s.readPool, injected, execArgs(injected, req.AgentID)...)
	wasCorrected = false
	if err != nil {
		sql2, _, pt2, ct2, rerr := GenerateSQLWithRetry(ctx, s.client, s.cfg, schemaBlock, examplesBlock, historyText, req, injected, err.Error())
		tokIn += pt2
		tokOut += ct2
		if rerr != nil {
			return "", nil, 0, false, tokIn, tokOut, "", fmt.Errorf("sql retry: %w (orig: %v)", rerr, err)
		}
		if err := Validate(sql2); err != nil {
			return "", nil, 0, false, tokIn, tokOut, "", err
		}
		injected2, err := InjectAgentFilter(sql2, req.AgentID, req.AgentRole)
		if err != nil {
			return "", nil, 0, false, tokIn, tokOut, "", err
		}
		rows, execDur, err = Execute(ctx, s.readPool, injected2, execArgs(injected2, req.AgentID)...)
		injected = injected2
		wasCorrected = true
		if err != nil {
			return "", nil, 0, wasCorrected, tokIn, tokOut, "", err
		}
	}

	return injected, rows, execDur.Milliseconds(), wasCorrected, tokIn, tokOut, "", nil
}

func execArgs(sql string, agentID string) []any {
	if strings.Contains(sql, "$1") {
		return []any{agentID}
	}
	return nil
}
