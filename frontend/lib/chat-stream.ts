import { getApiBaseUrl } from '@/lib/api-config'

export type ChatStreamEvent =
  | { type: 'token'; text: string }
  | { type: 'sql'; sql: string }
  | { type: 'error'; message: string }
  | { type: 'done'; cached?: boolean; row_count?: number; out_of_scope?: boolean; message_id?: string; error?: boolean }

export async function postChatStream(
  accessToken: string,
  body: { question: string; session_id?: string; conversation_id?: string },
  onEvent: (e: ChatStreamEvent) => void
): Promise<void> {
  const res = await fetch(`${getApiBaseUrl()}/chat`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify({
      question: body.question,
      ...(body.session_id ? { session_id: body.session_id } : {}),
      ...(body.conversation_id ? { conversation_id: body.conversation_id } : {}),
    }),
  })

  if (res.status === 401) {
    throw new Error('Unauthorized — sign in with a real API token (JWT). Demo password login does not call the Aria API.')
  }
  if (res.status === 429) {
    const retry = res.headers.get('Retry-After') ?? '60'
    throw new Error(`Rate limited. Try again in ${retry} seconds.`)
  }
  if (!res.ok) {
    const t = await res.text()
    throw new Error(t || `Request failed (${res.status})`)
  }

  const reader = res.body?.getReader()
  if (!reader) throw new Error('No response body')

  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })

    let idx: number
    while ((idx = buffer.indexOf('\n\n')) !== -1) {
      const block = buffer.slice(0, idx)
      buffer = buffer.slice(idx + 2)
      const lines = block.split('\n')
      for (const line of lines) {
        if (!line.startsWith('data: ')) continue
        const data = line.slice(6).trim()
        if (data === '[DONE]') continue
        try {
          const parsed = JSON.parse(data) as Record<string, unknown>
          const typ = parsed.type as string
          if (typ === 'token' && typeof parsed.text === 'string') {
            onEvent({ type: 'token', text: parsed.text })
          } else if (typ === 'sql' && typeof parsed.sql === 'string') {
            onEvent({ type: 'sql', sql: parsed.sql })
          } else if (typ === 'error' && typeof parsed.message === 'string') {
            onEvent({ type: 'error', message: parsed.message })
          } else if (typ === 'done') {
            onEvent({
              type: 'done',
              cached: parsed.cached as boolean | undefined,
              row_count: typeof parsed.row_count === 'number' ? parsed.row_count : undefined,
              out_of_scope: parsed.out_of_scope as boolean | undefined,
              message_id: typeof parsed.message_id === 'string' ? parsed.message_id : undefined,
              error: parsed.error as boolean | undefined,
            })
          }
        } catch {
          /* ignore malformed chunk */
        }
      }
    }
  }
}
