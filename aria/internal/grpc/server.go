package grpcserver

import (
	"context"
	"strings"
	"time"

	ariav1 "github.com/Deonkar/Aria/aria/gen/aria/v1"
	"github.com/Deonkar/Aria/aria/internal/ai"
	"github.com/Deonkar/Aria/aria/internal/cache"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	ariav1.UnimplementedAIQueryServiceServer
	QuerySvc *ai.QueryService
}

func NewServer(querySvc *ai.QueryService) *Server {
	return &Server{QuerySvc: querySvc}
}

func (s *Server) Query(ctx context.Context, req *ariav1.QueryRequest) (*ariav1.QueryResponse, error) {
	if strings.TrimSpace(req.GetAgentId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	if strings.TrimSpace(req.GetQuestion()) == "" {
		return nil, status.Error(codes.InvalidArgument, "question is required")
	}

	res, err := s.QuerySvc.Query(ctx, ai.QueryRequest{
		Question:  req.GetQuestion(),
		AgentID:   req.GetAgentId(),
		AgentRole: req.GetAgentRole(),
		AgentName: req.GetAgentName(),
		SessionID: req.GetSessionId(),
		Timezone:  req.GetTimezone(),
	})
	if err != nil {
		log.Warn().Err(err).Msg("grpc query failed")
		return nil, status.Error(codes.Internal, "query failed")
	}

	return &ariav1.QueryResponse{
		Answer:       res.Answer,
		GeneratedSql: res.SQL,
		RowCount:     int32(res.RowCount),
		ExecutionMs:  int32(res.ExecutionMs),
		WasCached:    res.WasCached,
		WasCorrected: res.WasCorrected,
		TokensInput:  int32(res.TokensIn),
		TokensOutput: int32(res.TokensOut),
	}, nil
}

func (s *Server) StreamQuery(req *ariav1.QueryRequest, stream ariav1.AIQueryService_StreamQueryServer) error {
	ctx := stream.Context()

	if strings.TrimSpace(req.GetAgentId()) == "" {
		return status.Error(codes.InvalidArgument, "agent_id is required")
	}
	if strings.TrimSpace(req.GetQuestion()) == "" {
		return status.Error(codes.InvalidArgument, "question is required")
	}

	aiReq := ai.QueryRequest{
		Question:  req.GetQuestion(),
		AgentID:   req.GetAgentId(),
		AgentRole: req.GetAgentRole(),
		AgentName: req.GetAgentName(),
		SessionID: req.GetSessionId(),
		Timezone:  req.GetTimezone(),
	}

	// Cache path: send SQL (if available), then tokens, then done.
	if cr, ok, err := cache.GetCachedQuery(ctx, s.QuerySvc.RDB(), aiReq.AgentID, aiReq.Question); err == nil && ok {
		if cr.SQL != "" {
			if err := stream.Send(&ariav1.QueryChunk{Type: ariav1.QueryChunk_CHUNK_TYPE_SQL, Text: cr.SQL}); err != nil {
				return err
			}
		}
		if err := streamChunkedText(stream, cr.Answer); err != nil {
			return err
		}
		return stream.Send(&ariav1.QueryChunk{
			Type:     ariav1.QueryChunk_CHUNK_TYPE_DONE,
			RowCount: int32(cr.RowCount),
			Cached:   true,
		})
	}

	sqlExec, rows, execMs, corrected, tin, tout, oosAns, err := s.QuerySvc.GenerateExecuteAndFormat(ctx, aiReq)
	_ = execMs
	_ = corrected
	_ = tin
	_ = tout

	if err == ai.ErrOutOfScope {
		if err := streamChunkedText(stream, oosAns); err != nil {
			return err
		}
		return stream.Send(&ariav1.QueryChunk{Type: ariav1.QueryChunk_CHUNK_TYPE_DONE, RowCount: 0, Cached: false})
	}
	if err != nil {
		msg := ai.UserFacingError(err)
		_ = stream.Send(&ariav1.QueryChunk{Type: ariav1.QueryChunk_CHUNK_TYPE_ERROR, Error: msg})
		return stream.Send(&ariav1.QueryChunk{Type: ariav1.QueryChunk_CHUNK_TYPE_DONE, RowCount: 0, Cached: false})
	}

	if err := stream.Send(&ariav1.QueryChunk{Type: ariav1.QueryChunk_CHUNK_TYPE_SQL, Text: sqlExec}); err != nil {
		return err
	}

	answer, _, _, err := s.QuerySvc.StreamAnswerTokens(ctx, aiReq.Question, sqlExec, rows, func(token string) error {
		return stream.Send(&ariav1.QueryChunk{Type: ariav1.QueryChunk_CHUNK_TYPE_TOKEN, Text: token})
	})
	if err != nil {
		msg := ai.UserFacingError(err)
		_ = stream.Send(&ariav1.QueryChunk{Type: ariav1.QueryChunk_CHUNK_TYPE_ERROR, Error: msg})
		return stream.Send(&ariav1.QueryChunk{Type: ariav1.QueryChunk_CHUNK_TYPE_DONE, RowCount: int32(len(rows)), Cached: false})
	}

	// Best-effort cache write, mirroring SSE behavior.
	_ = cache.SetCachedQuery(ctx, s.QuerySvc.RDB(), aiReq.AgentID, aiReq.Question, &cache.CachedResult{
		Answer:      answer,
		SQL:         sqlExec,
		RowCount:    len(rows),
		GeneratedAt: time.Now().UTC(),
	}, ai.QueryCacheTTL)

	return stream.Send(&ariav1.QueryChunk{
		Type:     ariav1.QueryChunk_CHUNK_TYPE_DONE,
		RowCount: int32(len(rows)),
		Cached:   false,
	})
}

func streamChunkedText(stream ariav1.AIQueryService_StreamQueryServer, text string) error {
	// Keep chunks small to mimic token-y delivery even for cached responses.
	const chunk = 20
	for i := 0; i < len(text); i += chunk {
		j := i + chunk
		if j > len(text) {
			j = len(text)
		}
		if err := stream.Send(&ariav1.QueryChunk{Type: ariav1.QueryChunk_CHUNK_TYPE_TOKEN, Text: text[i:j]}); err != nil {
			return err
		}
	}
	return nil
}

