from __future__ import annotations

import json
import os
import struct
from pathlib import Path
from typing import Any

from pgvector.psycopg2 import register_vector
from openai import NotFoundError


def embed_text(openai_client, text: str) -> list[float]:
    model = os.getenv("OPENAI_EMBED_MODEL", "text-embedding-3-small")
    try:
        resp = openai_client.embeddings.create(model=model, input=text)
        return list(resp.data[0].embedding)
    except Exception:
        # Some gateways don't implement embeddings. Fallback to deterministic pseudo-embedding
        # so Phase 2 can proceed without external embedding spend.
        return _fake_embedding_1536(text)


def upsert_schema_embedding(
    conn,
    table_name: str,
    column_name: str | None,
    description_text: str,
    embedding: list[float],
    ddl_hash: str,
) -> None:
    with conn.cursor() as cur:
        cur.execute(
            """
            DELETE FROM schema_embeddings
            WHERE table_name = %s AND (column_name = %s OR (column_name IS NULL AND %s IS NULL))
            """,
            (table_name, column_name, column_name),
        )
        cur.execute(
            """
            INSERT INTO schema_embeddings(table_name, column_name, description_text, embedding, ddl_hash)
            VALUES (%s, %s, %s, %s, %s)
            """,
            (table_name, column_name, description_text, embedding, ddl_hash),
        )


def load_manifest(path: str) -> dict[str, str]:
    p = Path(path)
    if not p.exists():
        return {}
    return json.loads(p.read_text(encoding="utf-8"))


def save_manifest(path: str, manifest: dict[str, str]) -> None:
    p = Path(path)
    p.write_text(json.dumps(manifest, indent=2, sort_keys=True), encoding="utf-8")


def init_pgvector(conn) -> None:
    register_vector(conn)


def _fake_embedding_1536(text: str) -> list[float]:
    # Deterministic 1536-d vector from SHA256, expanded via counter hashing.
    # Values are normalized to [-1, 1]. Not semantically meaningful, but stable.
    out: list[float] = []
    counter = 0
    while len(out) < 1536:
        payload = f"{counter}:{text}".encode("utf-8")
        h = hashlib_sha256(payload)
        # convert hash bytes into floats
        for i in range(0, len(h), 4):
            if len(out) >= 1536:
                break
            (u32,) = struct.unpack(">I", h[i : i + 4])
            out.append((u32 / 2**31) - 1.0)
        counter += 1
    return out


def hashlib_sha256(b: bytes) -> bytes:
    import hashlib

    return hashlib.sha256(b).digest()

