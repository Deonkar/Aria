from __future__ import annotations

from openai import OpenAI


def make_client(
    api_key: str,
    base_url: str | None = None,
    api_key_header: str | None = None,
) -> OpenAI:
    if api_key_header:
        # Some OpenAI-compatible gateways (e.g. YepAPI) expect an x-api-key header instead of Authorization: Bearer.
        return OpenAI(
            api_key="unused",
            base_url=base_url,
            default_headers={api_key_header: api_key},
        )
    if base_url:
        return OpenAI(api_key=api_key, base_url=base_url)
    return OpenAI(api_key=api_key)

