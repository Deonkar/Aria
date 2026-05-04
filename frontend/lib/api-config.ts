/** Base URL for the Aria Go API (no trailing slash). */
export function getApiBaseUrl(): string {
  const raw = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'
  return raw.replace(/\/$/, '')
}
