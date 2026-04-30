from __future__ import annotations

import json
import os
from typing import Any


DOC_SYSTEM_PROMPT = """You are a careful database documentation generator.
Return STRICT JSON only. No markdown. No extra keys.
Be concise: keep descriptions short and factual."""


def generate_table_doc(
    openai_client,
    table_name: str,
    ddl: str,
    fks: list[dict[str, Any]],
    sample_rows: list[dict[str, Any]],
) -> dict[str, Any]:
    model = os.getenv("OPENAI_CHAT_MODEL", "gpt-4o-mini")
    prompt = {
        "table_name": table_name,
        "ddl": ddl,
        "foreign_keys": fks,
        "sample_rows": sample_rows,
        "instructions": {
            "output_shape": {
                "table_name": table_name,
                "purpose": "string",
                "columns": [
                    {"name": "string", "description": "string", "possible_values": ["optional strings"]}
                ],
                "common_query_patterns": ["strings"],
                "relationships": ["strings like table.col -> other.col"],
            }
        },
    }

    resp = openai_client.chat.completions.create(
        model=model,
        temperature=0.1,
        max_tokens=800,
        response_format={"type": "json_object"},
        messages=[
            {"role": "system", "content": DOC_SYSTEM_PROMPT},
            {"role": "user", "content": json.dumps(prompt)},
        ],
    )

    content = resp.choices[0].message.content or ""
    try:
        doc = json.loads(content)
    except json.JSONDecodeError as e:
        raise ValueError(f"invalid JSON response for {table_name}: {content[:200]}") from e

    # Minimal validation
    for k in ("table_name", "purpose", "columns"):
        if k not in doc:
            raise ValueError(f"doc missing key: {k} (table={table_name})")
    if doc["table_name"] != table_name:
        raise ValueError("doc table_name mismatch")
    return doc


def format_for_embedding(doc: dict[str, Any]) -> str:
    # Flatten doc into a stable, plain-text representation.
    parts: list[str] = []
    parts.append(f"Table: {doc.get('table_name','')}")
    parts.append(f"Purpose: {doc.get('purpose','')}")

    cols = doc.get("columns") or []
    for c in cols:
        name = c.get("name", "")
        desc = c.get("description", "")
        parts.append(f"Column {name}: {desc}")
        pv = c.get("possible_values")
        if isinstance(pv, list) and pv:
            parts.append(f"Possible values for {name}: {', '.join(str(x) for x in pv)}")

    rels = doc.get("relationships")
    if isinstance(rels, list) and rels:
        parts.append("Relationships: " + " | ".join(str(x) for x in rels))

    patterns = doc.get("common_query_patterns")
    if isinstance(patterns, list) and patterns:
        parts.append("Common query patterns: " + " | ".join(str(x) for x in patterns))

    text = ". ".join(p.strip() for p in parts if p and p.strip())
    # Keep within a reasonable embedding input size.
    return text[:8000]

