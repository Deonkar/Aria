from __future__ import annotations

import hashlib
from dataclasses import dataclass
from typing import Any

import psycopg2
from psycopg2.extras import RealDictCursor


def get_tables(conn) -> list[str]:
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT table_name
            FROM information_schema.tables
            WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
            ORDER BY table_name
            """
        )
        return [r[0] for r in cur.fetchall()]


@dataclass(frozen=True)
class ColumnInfo:
    name: str
    data_type: str
    is_nullable: bool
    default: str | None


def _get_columns(conn, table_name: str) -> list[ColumnInfo]:
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT
              column_name,
              data_type,
              is_nullable,
              column_default
            FROM information_schema.columns
            WHERE table_schema='public' AND table_name=%s
            ORDER BY ordinal_position
            """,
            (table_name,),
        )
        out: list[ColumnInfo] = []
        for name, data_type, is_nullable, default in cur.fetchall():
            out.append(
                ColumnInfo(
                    name=name,
                    data_type=data_type,
                    is_nullable=(is_nullable == "YES"),
                    default=default,
                )
            )
        return out


def get_columns(conn, table_name: str) -> list[ColumnInfo]:
    return _get_columns(conn, table_name)


def get_table_ddl(conn, table_name: str) -> str:
    # Reconstruct a CREATE TABLE using information_schema (sufficient for hashing/incremental runs).
    cols = _get_columns(conn, table_name)
    if not cols:
        raise ValueError(f"table not found: {table_name}")

    col_lines: list[str] = []
    for c in cols:
        line = f'    "{c.name}" {c.data_type}'
        if not c.is_nullable:
            line += " NOT NULL"
        if c.default:
            line += f" DEFAULT {c.default}"
        col_lines.append(line)

    ddl = f'CREATE TABLE "{table_name}" (\n' + ",\n".join(col_lines) + "\n);"
    return ddl


def get_foreign_keys(conn, table_name: str) -> list[dict[str, Any]]:
    with conn.cursor(cursor_factory=RealDictCursor) as cur:
        cur.execute(
            """
            SELECT
              tc.constraint_name,
              kcu.column_name,
              ccu.table_name AS foreign_table_name,
              ccu.column_name AS foreign_column_name
            FROM information_schema.table_constraints AS tc
            JOIN information_schema.key_column_usage AS kcu
              ON tc.constraint_name = kcu.constraint_name
             AND tc.table_schema = kcu.table_schema
            JOIN information_schema.constraint_column_usage AS ccu
              ON ccu.constraint_name = tc.constraint_name
             AND ccu.table_schema = tc.table_schema
            WHERE tc.constraint_type = 'FOREIGN KEY'
              AND tc.table_schema = 'public'
              AND tc.table_name = %s
            ORDER BY tc.constraint_name, kcu.ordinal_position
            """,
            (table_name,),
        )
        return list(cur.fetchall())


def get_sample_rows(conn, table_name: str, n: int = 3) -> list[dict[str, Any]]:
    with conn.cursor(cursor_factory=RealDictCursor) as cur:
        cur.execute(f'SELECT * FROM "{table_name}" LIMIT %s', (n,))
        return [dict(r) for r in cur.fetchall()]


def hash_ddl(ddl: str) -> str:
    return hashlib.sha256(ddl.encode("utf-8")).hexdigest()


def connect(dsn: str):
    return psycopg2.connect(dsn)

