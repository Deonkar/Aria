package ai

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Execute(ctx context.Context, readPool *pgxpool.Pool, sql string, args ...any) ([]map[string]any, time.Duration, error) {
	if readPool == nil {
		return nil, 0, fmt.Errorf("nil pool")
	}
	qctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	start := time.Now()
	rows, err := readPool.Query(qctx, sql, args...)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(qctx.Err(), context.DeadlineExceeded) {
			return nil, time.Since(start), ErrQueryTimeout
		}
		return nil, time.Since(start), err
	}
	defer rows.Close()

	fds := rows.FieldDescriptions()
	out := make([]map[string]any, 0)
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, time.Since(start), err
		}
		row := make(map[string]any, len(fds))
		for i, fd := range fds {
			row[string(fd.Name)] = vals[i]
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(qctx.Err(), context.DeadlineExceeded) {
			return nil, time.Since(start), ErrQueryTimeout
		}
		return nil, time.Since(start), err
	}
	return out, time.Since(start), nil
}

// ErrNoRows is returned when pgx returns pgx.ErrNoRows for a single-row helper (not used in Execute).
var ErrNoRows = pgx.ErrNoRows
