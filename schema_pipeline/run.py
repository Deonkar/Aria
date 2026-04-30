from __future__ import annotations

import argparse
import os
from datetime import datetime, timezone

from dotenv import load_dotenv

from document import format_for_embedding, generate_table_doc
from embed import (
    embed_text,
    init_pgvector,
    load_manifest,
    save_manifest,
    upsert_schema_embedding,
    _fake_embedding_1536,
)
from generate_examples import generate_intent_examples, upsert_intent_examples
from heuristic_doc import build_heuristic_doc
from introspect import (
    connect,
    get_columns,
    get_foreign_keys,
    get_sample_rows,
    get_table_ddl,
    get_tables,
    hash_ddl,
)
from openai_client import make_client


def main() -> int:
    parser = argparse.ArgumentParser(description="Aria schema pipeline (Phase 2)")
    parser.add_argument("--dsn", default=os.getenv("DATABASE_URL"), help="Postgres DSN (defaults to DATABASE_URL)")
    parser.add_argument(
        "--manifest",
        default=os.getenv("SCHEMA_MANIFEST_PATH", "/app/manifest.json"),
        help="Manifest file path for incremental runs",
    )
    parser.add_argument("--topk", type=int, default=5, help="Top-K schema docs to retrieve later (stored as embeddings)")
    args = parser.parse_args()

    load_dotenv()

    dsn = args.dsn or os.getenv("DATABASE_URL")
    api_key = os.getenv("OPENAI_API_KEY", "").strip()
    base_url = os.getenv("OPENAI_BASE_URL", "").strip() or None
    api_key_header = os.getenv("OPENAI_API_KEY_HEADER", "").strip() or None
    doc_mode = (os.getenv("SCHEMA_DOC_MODE", "heuristic") or "heuristic").strip().lower()
    generate_examples_flag = (os.getenv("GENERATE_INTENT_EXAMPLES", "false") or "false").strip().lower() in (
        "1",
        "true",
        "yes",
    )
    if not dsn:
        raise SystemExit("Missing DATABASE_URL / --dsn")
    # Only require OPENAI_API_KEY when we are actually calling the provider.
    if doc_mode != "heuristic" or generate_examples_flag:
        if not api_key:
            raise SystemExit("Missing OPENAI_API_KEY")

    client = make_client(api_key, base_url=base_url, api_key_header=api_key_header) if api_key else None
    manifest = load_manifest(args.manifest)

    conn = connect(dsn)
    conn.autocommit = False
    init_pgvector(conn)

    try:
        tables = get_tables(conn)
        processed = 0

        all_docs: list[dict] = []
        new_manifest: dict[str, str] = dict(manifest)

        for t in tables:
            ddl = get_table_ddl(conn, t)
            ddl_hash = hash_ddl(ddl)
            prev = manifest.get(t)

            if prev == ddl_hash:
                print(f"Skipping table: {t} (unchanged)")
                continue

            print(f"Processing table: {t} (changed)")
            fks = get_foreign_keys(conn, t)
            cols = get_columns(conn, t)

            if doc_mode == "heuristic":
                doc = build_heuristic_doc(t, cols, fks)
            else:
                samples = get_sample_rows(conn, t, n=3)
                if client is None:
                    raise RuntimeError("OPENAI client not configured")
                doc = generate_table_doc(client, t, ddl, fks, samples)
            all_docs.append(doc)

            emb_text = format_for_embedding(doc)
            emb = _fake_embedding_1536(emb_text) if client is None else embed_text(client, emb_text)
            upsert_schema_embedding(conn, t, None, emb_text, emb, ddl_hash)
            new_manifest[t] = ddl_hash
            processed += 1

        # Intent examples are re-generated if any table doc changed, or if table docs are missing on first run.
        if generate_examples_flag and (processed > 0 or not manifest):
            print("Generating intent examples...")
            # Use docs for all tables when possible; fallback to changed tables docs.
            docs_for_examples = all_docs
            if not docs_for_examples:
                docs_for_examples = [{"table_name": t, "purpose": "", "columns": []} for t in tables]
            if client is None:
                raise RuntimeError("OPENAI client not configured for intent examples")
            examples = generate_intent_examples(client, docs_for_examples)
            upsert_intent_examples(conn, client, examples)

        conn.commit()
        save_manifest(args.manifest, new_manifest)

        now = datetime.now(timezone.utc).isoformat()
        print(
            f"Done. Processed {processed}/{len(tables)} tables. "
            f"Manifest saved. Time={now}"
        )
    except Exception:
        conn.rollback()
        raise
    finally:
        conn.close()

    return 0


if __name__ == "__main__":
    raise SystemExit(main())

