'use client'

import { useState, useEffect, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ArrowRight, MessageCircle, Eye, EyeOff, Loader2, ExternalLink } from 'lucide-react'
import { getApiBaseUrl } from '@/lib/api-config'
import { decodeJwtPayload } from '@/lib/jwt'
import { toast } from 'sonner'

type OAuthPayload = {
  access_token: string
  user?: { email?: string; full_name?: string; role?: string }
}

export default function LoginPage() {
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [email, setEmail] = useState('demo@example.com')
  const [password, setPassword] = useState('demo123')
  const [apiJwt, setApiJwt] = useState('')

  const persistToken = useCallback(
    (accessToken: string, user?: OAuthPayload['user']) => {
      localStorage.setItem('authToken', accessToken)
      const claims = decodeJwtPayload(accessToken)
      const uEmail = user?.email ?? claims?.email ?? email
      const uName =
        user?.full_name?.split(' ')[0] ??
        claims?.email?.split('@')[0] ??
        email.split('@')[0]
      localStorage.setItem('userEmail', uEmail)
      localStorage.setItem('userName', uName)
      const admin = (user?.role ?? claims?.role) === 'admin'
      localStorage.setItem('isAdmin', admin ? 'true' : 'false')
      toast.success('Signed in with Aria API')
      router.push('/dashboard/chat')
    },
    [email, router]
  )

  useEffect(() => {
    const apiOrigin = new URL(getApiBaseUrl()).origin
    function onMessage(ev: MessageEvent) {
      if (ev.origin !== apiOrigin) return
      const data = ev.data as { source?: string; payload?: OAuthPayload }
      if (data?.source !== 'aria-oauth' || !data.payload?.access_token) return
      persistToken(data.payload.access_token, data.payload.user)
    }
    window.addEventListener('message', onMessage)
    return () => window.removeEventListener('message', onMessage)
  }, [persistToken])

  const openGooglePopup = () => {
    const url = `${getApiBaseUrl()}/auth/google?popup=1`
    const w = window.open(url, 'aria-oauth', 'width=520,height=720,scrollbars=yes')
    if (!w) {
      toast.error('Popup blocked — allow popups for this site or use the link below.')
    } else {
      toast.message('Complete Google sign-in in the popup window.')
    }
  }

  /** Real JWT from API when ALLOW_DEMO_AUTH=true — no Google OAuth. */
  const quickDemoLogin = async () => {
    setIsLoading(true)
    try {
      const res = await fetch(`${getApiBaseUrl()}/auth/demo-token`, { method: 'POST' })
      const text = await res.text()
      let body: { error?: string; access_token?: string; user?: OAuthPayload['user'] } = {}
      try {
        body = JSON.parse(text) as typeof body
      } catch {
        /* non-JSON */
      }
      if (!res.ok) {
        throw new Error(body.error || text || res.statusText)
      }
      if (!body.access_token) {
        throw new Error('No access_token in response')
      }
      persistToken(body.access_token, body.user)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Demo login failed')
    } finally {
      setIsLoading(false)
    }
  }

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    const trimmedJwt = apiJwt.trim()
    if (trimmedJwt) {
      setIsLoading(true)
      try {
        persistToken(trimmedJwt)
      } finally {
        setIsLoading(false)
      }
      return
    }
    await quickDemoLogin()
  }

  const demoAccounts = [
    { email: 'demo@example.com', password: 'demo123', role: 'Admin' },
    { email: 'user@example.com', password: 'user123', role: 'User' },
  ]

  return (
    <main className="min-h-screen bg-background flex items-center justify-center px-4 overflow-hidden">
      <div className="absolute inset-0 bg-gradient-to-br from-background via-background to-secondary/5" />
      <div className="absolute top-0 left-1/4 w-96 h-96 bg-accent/10 rounded-full blur-3xl animate-pulse" />
      <div className="absolute bottom-0 right-1/4 w-96 h-96 bg-accent/5 rounded-full blur-3xl animate-pulse animation-delay-2000" />

      <div className="relative z-10 w-full max-w-md">
        <div className="text-center mb-8">
          <div className="inline-flex items-center gap-3 mb-6">
            <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-accent to-accent/70 flex items-center justify-center">
              <MessageCircle className="w-7 h-7 text-accent-foreground" />
            </div>
            <span className="text-2xl font-bold bg-gradient-to-r from-accent to-accent/70 bg-clip-text text-transparent">
              Aria CRM
            </span>
          </div>
          <h1 className="text-3xl font-bold mb-2">Welcome Back</h1>
          <p className="text-muted-foreground">Sign in to your account to continue</p>
        </div>

        <div className="p-8 rounded-2xl border border-border/40 bg-card/50 backdrop-blur-sm mb-6 space-y-4">
          <Button
            type="button"
            variant="outline"
            className="w-full h-11 rounded-lg border-border/60"
            onClick={openGooglePopup}
            disabled={isLoading}
          >
            <ExternalLink className="w-4 h-4 mr-2" />
            Continue with Google (Aria API)
          </Button>
          <p className="text-[11px] text-muted-foreground text-center">
            Opens a popup to <span className="font-mono">{getApiBaseUrl()}</span>. Your app must be allowed in{' '}
            <code className="text-[10px]">FRONTEND_ORIGIN</code> on the API (e.g. http://localhost:3000).
          </p>

          <div className="relative py-2">
            <div className="absolute inset-0 flex items-center">
              <span className="w-full border-t border-border/40" />
            </div>
            <div className="relative flex justify-center text-xs uppercase tracking-wide text-muted-foreground">
              <span className="bg-card/80 px-2">Or</span>
            </div>
          </div>

          <form onSubmit={handleLogin} className="space-y-4">
            <div>
              <Label htmlFor="email" className="text-sm font-medium mb-2 block">
                Email
              </Label>
              <Input
                id="email"
                type="email"
                placeholder="demo@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={isLoading}
                className="rounded-lg bg-secondary/30 border-border/40 focus:border-accent/50"
              />
            </div>

            <div>
              <Label htmlFor="password" className="text-sm font-medium mb-2 block">
                Password
              </Label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  placeholder="••••••••"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  disabled={isLoading}
                  className="rounded-lg bg-secondary/30 border-border/40 focus:border-accent/50 pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  disabled={isLoading}
                >
                  {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
            </div>

          <Button
            type="submit"
            disabled={isLoading}
            className="w-full bg-accent hover:bg-accent/90 text-accent-foreground h-11 rounded-lg font-semibold"
          >
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Signing in...
              </>
            ) : apiJwt.trim() ? (
              <>
                Sign in with pasted token
                <ArrowRight className="w-4 h-4 ml-2" />
              </>
            ) : (
              <>
                Quick demo (API token — no Google)
                <ArrowRight className="w-4 h-4 ml-2" />
              </>
            )}
          </Button>

            <div className="pt-4 border-t border-border/40 space-y-2">
              <Label htmlFor="apiJwt" className="text-sm font-medium">
                Paste API token (optional)
              </Label>
              <Input
                id="apiJwt"
                type="password"
                autoComplete="off"
                placeholder="access_token if you use full-window OAuth"
                value={apiJwt}
                onChange={(e) => setApiJwt(e.target.value)}
                disabled={isLoading}
                className="rounded-lg bg-secondary/30 border-border/40 font-mono text-xs"
              />
              <a
                href={`${getApiBaseUrl()}/auth/google`}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 text-xs text-accent hover:underline"
              >
                Open Google sign-in in a new tab <ExternalLink className="w-3 h-3" />
              </a>
            </div>
          </form>
        </div>

        <div className="p-6 rounded-2xl border border-border/40 bg-secondary/20 backdrop-blur-sm">
          <h3 className="font-semibold mb-4 text-sm">Legacy demo picker</h3>
          <p className="text-xs text-muted-foreground mb-3">
            <strong>Quick demo</strong> uses <code className="text-[10px]">POST /auth/demo-token</code> when{' '}
            <code className="text-[10px]">ALLOW_DEMO_AUTH=true</code> on the API. Below only fills the optional token field.
          </p>
          <div className="space-y-3">
            {demoAccounts.map((account, i) => (
              <div
                key={i}
                className="p-3 rounded-lg bg-background/50 border border-border/40 hover:border-accent/30 cursor-pointer transition-all"
                onClick={() => {
                  void quickDemoLogin()
                }}
              >
                <div className="flex items-start justify-between">
                  <div>
                    <p className="text-sm font-medium">{account.role}</p>
                    <p className="text-xs text-muted-foreground mt-1">{account.email}</p>
                  </div>
                  <span className="text-xs px-2 py-1 rounded-full bg-accent/10 text-accent">Quick demo</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </main>
  )
}
