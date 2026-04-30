from __future__ import annotations

import json
import os
from typing import Any

from openai import NotFoundError

from embed import _fake_embedding_1536

EXAMPLES_SYSTEM_PROMPT = """You generate realistic CRM agent questions and exact PostgreSQL SQL templates.
Rules:
- Return STRICT JSON only (no markdown).
- SQL must be SELECT-only.
- Use :agent_id placeholder for any agent scoping.
- Be concise.
"""


def generate_intent_examples(openai_client, all_table_docs: list[dict[str, Any]]) -> list[dict[str, Any]]:
    model = os.getenv("OPENAI_CHAT_MODEL", "gpt-4o-mini")
    payload = {
        "instructions": "Generate exactly 30 example question->SQL pairs. Cover leads, tasks, bookings, payments, activities, cross-table joins.",
        "schema_docs": all_table_docs,
        "output_shape": [
            {
                "question": "string",
                "sql_template": "string",
                "tables_used": ["strings"],
                "intent_category": "string",
            }
        ],
    }

    resp = openai_client.chat.completions.create(
        model=model,
        temperature=0.1,
        max_tokens=1600,
        response_format={"type": "json_object"},
        messages=[
            {"role": "system", "content": EXAMPLES_SYSTEM_PROMPT},
            {"role": "user", "content": json.dumps(payload)},
        ],
    )
    content = resp.choices[0].message.content or ""
    parsed = json.loads(content)
    # Allow either direct list or { "examples": [...] }
    if isinstance(parsed, dict) and "examples" in parsed:
        examples = parsed["examples"]
    else:
        examples = parsed
    if not isinstance(examples, list):
        raise ValueError("examples response is not a list")
    if len(examples) != 30:
        raise ValueError(f"expected 30 examples, got {len(examples)}")
    return examples


def upsert_intent_examples(conn, openai_client, examples: list[dict[str, Any]]) -> None:
    # Simple strategy: wipe and reinsert (POC scale). Keeps idempotency.
    with conn.cursor() as cur:
        cur.execute("DELETE FROM intent_examples")

    for ex in examples:
        q = ex["question"]
        sql = ex["sql_template"]
        tables_used = ex.get("tables_used") or []
        intent_category = ex.get("intent_category")
        embed_model = os.getenv("OPENAI_EMBED_MODEL", "text-embedding-3-small")
        try:
            embedding = openai_client.embeddings.create(model=embed_model, input=q).data[0].embedding
            emb_list = list(embedding)
        except NotFoundError:
            emb_list = _fake_embedding_1536(q)

        with conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO intent_examples(question_text, question_embedding, sql_template, tables_used, intent_category, source, upvote_count, is_active)
                VALUES (%s, %s, %s, %s, %s, 'auto_generated', 0, TRUE)
                """,
                (q, emb_list, sql, tables_used, intent_category),
            )

