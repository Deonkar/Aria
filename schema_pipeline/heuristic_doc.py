from __future__ import annotations

from typing import Any

from introspect import ColumnInfo


def build_heuristic_doc(table_name: str, columns: list[ColumnInfo], fks: list[dict[str, Any]]) -> dict[str, Any]:
    # Minimal, deterministic docs (no LLM). Keeps Phase 2 unblocked and cheap.
    rels: list[str] = []
    for fk in fks:
        rels.append(
            f"{table_name}.{fk.get('column_name')} -> {fk.get('foreign_table_name')}.{fk.get('foreign_column_name')}"
        )

    cols_out: list[dict[str, Any]] = []
    for c in columns:
        desc = f"{c.data_type}" + (" (nullable)" if c.is_nullable else "")
        if c.default:
            desc += f", default={c.default}"
        cols_out.append({"name": c.name, "description": desc})

    purpose = f"Table {table_name} (auto-generated docs from information_schema)."
    return {
        "table_name": table_name,
        "purpose": purpose,
        "columns": cols_out,
        "common_query_patterns": [],
        "relationships": rels,
    }

