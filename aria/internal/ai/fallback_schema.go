package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LoadInformationSchemaFallback returns a compact text summary of public tables/columns when embeddings are empty.
func LoadInformationSchemaFallback(ctx context.Context, readPool *pgxpool.Pool) (string, error) {
	if readPool == nil {
		return "", fmt.Errorf("nil pool")
	}
	q := `
SELECT table_name, column_name, data_type
FROM information_schema.columns
WHERE table_schema = 'public'
ORDER BY table_name, ordinal_position
`
	rows, err := readPool.Query(ctx, q)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var b strings.Builder
	var curTable string
	for rows.Next() {
		var table, col, dtype string
		if err := rows.Scan(&table, &col, &dtype); err != nil {
			return "", err
		}
		if table != curTable {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			fmt.Fprintf(&b, "Table %s:", table)
			curTable = table
		}
		fmt.Fprintf(&b, " %s(%s)", col, dtype)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if b.Len() == 0 {
		return "(no public columns found)", nil
	}
	return b.String(), nil
}
