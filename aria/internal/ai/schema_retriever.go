package ai

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openai/openai-go"
	"github.com/pgvector/pgvector-go"
	"github.com/rs/zerolog/log"
)

type SchemaDoc struct {
	TableName       string
	ColumnName      string
	DescriptionText string
}

type IntentExample struct {
	QuestionText     string
	SQLTemplate      string
	IntentCategory   string
}

func countSchemaEmbeddings(ctx context.Context, readPool *pgxpool.Pool) (int64, error) {
	var n int64
	err := readPool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_embeddings`).Scan(&n)
	return n, err
}

// RetrieveRelevantSchema runs similarity search over schema_embeddings.
func RetrieveRelevantSchema(ctx context.Context, readPool *pgxpool.Pool, client openai.Client, embedModel, question string, topK int) ([]SchemaDoc, error) {
	if topK <= 0 {
		topK = 5
	}
	n, err := countSchemaEmbeddings(ctx, readPool)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	vec, _, _, err := EmbedText(ctx, client, embedModel, question)
	if err != nil {
		log.Warn().Err(err).Msg("schema retrieval: embedding API failed, skipping pgvector similarity (use information_schema fallback)")
		return nil, nil
	}
	pgvec := pgvector.NewVector(vec)
	rows, err := readPool.Query(ctx, `
SELECT table_name, COALESCE(column_name, ''), description_text
FROM schema_embeddings
ORDER BY embedding <=> $1
LIMIT $2
`, pgvec, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SchemaDoc
	for rows.Next() {
		var d SchemaDoc
		if err := rows.Scan(&d.TableName, &d.ColumnName, &d.DescriptionText); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// RetrieveIntentExamples searches intent_examples by question_embedding similarity.
func RetrieveIntentExamples(ctx context.Context, readPool *pgxpool.Pool, client openai.Client, embedModel, question string, topK int) ([]IntentExample, error) {
	if topK <= 0 {
		topK = 3
	}
	var n int64
	if err := readPool.QueryRow(ctx, `SELECT COUNT(*) FROM intent_examples WHERE question_embedding IS NOT NULL AND is_active = true`).Scan(&n); err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	vec, _, _, err := EmbedText(ctx, client, embedModel, question)
	if err != nil {
		log.Warn().Err(err).Msg("intent examples: embedding API failed, skipping similarity search")
		return nil, nil
	}
	pgvec := pgvector.NewVector(vec)
	rows, err := readPool.Query(ctx, `
SELECT question_text, sql_template, COALESCE(intent_category, '')
FROM intent_examples
WHERE question_embedding IS NOT NULL AND is_active = true
ORDER BY question_embedding <=> $1
LIMIT $2
`, pgvec, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []IntentExample
	for rows.Next() {
		var ex IntentExample
		if err := rows.Scan(&ex.QuestionText, &ex.SQLTemplate, &ex.IntentCategory); err != nil {
			return nil, err
		}
		out = append(out, ex)
	}
	return out, rows.Err()
}

// SchemaContext builds a prompt block from docs and optional fallback.
func SchemaContext(docs []SchemaDoc, fallback string) string {
	if len(docs) == 0 {
		if fallback != "" {
			return "DATABASE CONTEXT (fallback — no embeddings yet):\n" + fallback
		}
		return "DATABASE CONTEXT: (none retrieved)"
	}
	var b string
	for _, d := range docs {
		col := d.ColumnName
		if col == "" {
			col = "(table)"
		}
		b += fmt.Sprintf("- %s.%s: %s\n", d.TableName, col, d.DescriptionText)
	}
	return "DATABASE CONTEXT:\n" + b
}

func IntentExamplesContext(examples []IntentExample) string {
	if len(examples) == 0 {
		return "EXAMPLE QUERIES: (none retrieved)"
	}
	var b string
	for _, e := range examples {
		b += fmt.Sprintf("- Q: %s\n  SQL template: %s\n  Category: %s\n", e.QuestionText, e.SQLTemplate, e.IntentCategory)
	}
	return "EXAMPLE QUERIES:\n" + b
}
