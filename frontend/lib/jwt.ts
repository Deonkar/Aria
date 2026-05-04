/** Decode JWT payload without verifying (client-side display / role hints only). */
export function decodeJwtPayload(token: string): {
  sub?: string
  email?: string
  role?: string
} | null {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) return null
    const b64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const json = atob(b64)
    const p = JSON.parse(json) as Record<string, unknown>
    return {
      sub: typeof p.sub === 'string' ? p.sub : undefined,
      email: typeof p.email === 'string' ? p.email : undefined,
      role: typeof p.role === 'string' ? p.role : undefined,
    }
  } catch {
    return null
  }
}
