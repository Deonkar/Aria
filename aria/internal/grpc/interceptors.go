package grpcserver

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RecoveryUnaryInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp any, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Str("method", info.FullMethod).Msg("grpc panic recovered")
			err = status.Error(codes.Internal, "internal server error")
		}
	}()
	return handler(ctx, req)
}

func LoggingUnaryInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	log.Info().
		Str("method", info.FullMethod).
		Dur("duration", time.Since(start)).
		Bool("success", err == nil).
		Msg("grpc unary")
	return resp, err
}

func LoggingStreamInterceptor(
	srv any,
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	start := time.Now()
	err := handler(srv, ss)
	log.Info().
		Str("method", info.FullMethod).
		Dur("duration", time.Since(start)).
		Bool("success", err == nil).
		Msg("grpc stream")
	return err
}

