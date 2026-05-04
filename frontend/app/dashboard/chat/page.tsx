'use client'

import { useState, useRef, useEffect, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { MessageCircle, Send, Loader2, Copy, ThumbsUp, ThumbsDown, Plus, Trash2, Sparkles } from 'lucide-react'
import { toast } from 'sonner'
import { postChatStream } from '@/lib/chat-stream'
import { getApiBaseUrl } from '@/lib/api-config'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { ChevronDown } from 'lucide-react'

interface Message {
  id: string
  type: 'user' | 'ai'
  content: string
  timestamp: Date
  isLoading?: boolean
  sql?: string
  assistantMessageId?: string
  meta?: { cached?: boolean; rowCount?: number; outOfScope?: boolean }
}

interface Conversation {
  id: string
  sessionId: string
  title: string
  messages: Message[]
}

const WELCOME =
  "Hi — I'm Aria. Ask about your assigned leads, counts, priorities, and other CRM data. Answers come from your live database (not canned text)."

function newConversationTemplate(): Conversation {
  const sessionId = crypto.randomUUID()
  return {
    id: sessionId,
    sessionId,
    title: 'New chat',
    messages: [
      {
        id: 'welcome',
        type: 'ai',
        content: WELCOME,
        timestamp: new Date(),
      },
    ],
  }
}

export default function ChatPage() {
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [currentConversation, setCurrentConversation] = useState<Conversation | null>(null)
  const [inputValue, setInputValue] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const first = newConversationTemplate()
    setConversations([first])
    setCurrentConversation(first)
  }, [])

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [currentConversation?.messages])

  const authToken = () => {
    if (typeof window === 'undefined') return null
    return localStorage.getItem('authToken')
  }

  const mergeConv = useCallback((convId: string, updater: (c: Conversation) => Conversation) => {
    setConversations((prev) => prev.map((c) => (c.id === convId ? updater(c) : c)))
    setCurrentConversation((prev) => (prev && prev.id === convId ? updater(prev) : prev))
  }, [])

  const handleSendMessage = async () => {
    const q = inputValue.trim()
    if (!q || !currentConversation || isLoading) return

    const token = authToken()
    if (!token || token.startsWith('demo_token_')) {
      toast.error('Chat needs a real API JWT. On the login page, paste your access_token from Google OAuth (API), then try again.')
      return
    }

    if (q.length > 500) {
      toast.error('Question is too long (max 500 characters).')
      return
    }

    const convId = currentConversation.id
    const sessionId = currentConversation.sessionId

    const userMessage: Message = {
      id: `u-${Date.now()}`,
      type: 'user',
      content: q,
      timestamp: new Date(),
    }

    const aiId = `ai-${Date.now()}`
    const aiPlaceholder: Message = {
      id: aiId,
      type: 'ai',
      content: '',
      timestamp: new Date(),
      isLoading: true,
    }

    mergeConv(convId, (c) => {
      const nextTitle =
        c.title === 'New chat' || c.title === 'Customer Insights' ? q.slice(0, 48) + (q.length > 48 ? '…' : '') : c.title
      return {
        ...c,
        title: nextTitle,
        messages: [...c.messages, userMessage, aiPlaceholder],
      }
    })

    setInputValue('')
    setIsLoading(true)

    let full = ''
    let doneMeta: Message['meta']
    let assistantMessageId: string | undefined

    try {
      await postChatStream(token, { question: q, session_id: sessionId }, (ev) => {
        if (ev.type === 'token') {
          full += ev.text
          mergeConv(convId, (c) => ({
            ...c,
            messages: c.messages.map((m) => (m.id === aiId ? { ...m, content: full } : m)),
          }))
        } else if (ev.type === 'sql') {
          mergeConv(convId, (c) => ({
            ...c,
            messages: c.messages.map((m) => (m.id === aiId ? { ...m, sql: ev.sql } : m)),
          }))
        } else if (ev.type === 'error') {
          toast.error(ev.message)
        } else if (ev.type === 'done') {
          assistantMessageId = ev.message_id
          doneMeta = {
            cached: ev.cached,
            rowCount: ev.row_count,
            outOfScope: ev.out_of_scope,
          }
          mergeConv(convId, (c) => ({
            ...c,
            messages: c.messages.map((m) =>
              m.id === aiId
                ? {
                    ...m,
                    isLoading: false,
                    assistantMessageId: ev.message_id,
                    meta: doneMeta,
                    content: full || m.content || '—',
                  }
                : m
            ),
          }))
        }
      })
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Request failed'
      toast.error(msg)
      mergeConv(convId, (c) => ({
        ...c,
        messages: c.messages.map((m) =>
          m.id === aiId ? { ...m, isLoading: false, content: `Error: ${msg}` } : m
        ),
      }))
    } finally {
      setIsLoading(false)
      mergeConv(convId, (c) => ({
        ...c,
        messages: c.messages.map((m) =>
          m.id === aiId && m.isLoading ? { ...m, isLoading: false, content: m.content || full || '—' } : m
        ),
      }))
    }
  }

  const newConversation = () => {
    const conv = newConversationTemplate()
    setConversations((prev) => [conv, ...prev])
    setCurrentConversation(conv)
  }

  const deleteConversation = (id: string) => {
    setConversations((prev) => {
      const filtered = prev.filter((c) => c.id !== id)
      if (currentConversation?.id === id) {
        setCurrentConversation(filtered[0] ?? null)
      }
      return filtered
    })
  }

  const sendFeedback = async (message: Message, helpful: boolean) => {
    const token = authToken()
    if (!token || !message.assistantMessageId) {
      toast.error('No message id from server — cannot submit feedback.')
      return
    }
    try {
      const res = await fetch(`${getApiBaseUrl()}/messages/${message.assistantMessageId}/feedback`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ is_helpful: helpful }),
      })
      if (res.status === 409) {
        toast.message('Feedback already recorded.')
        return
      }
      if (!res.ok) {
        throw new Error(await res.text())
      }
      toast.success(helpful ? 'Thanks — marked helpful.' : 'Thanks — we’ll use this to improve.')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Feedback failed')
    }
  }

  return (
    <div className="flex h-screen bg-gradient-to-br from-background via-background to-secondary dark:from-background dark:via-background dark:to-background/50">
      <div className="hidden lg:flex lg:w-64 bg-card/50 dark:bg-card/40 backdrop-blur-sm border-r border-border/50 flex-col">
        <div className="p-4 border-b border-border/30">
          <Button
            onClick={newConversation}
            className="w-full bg-gradient-to-r from-accent to-accent/80 hover:from-accent hover:to-accent/90 text-accent-foreground shadow-lg hover:shadow-accent/20 transition-all"
          >
            <Plus className="w-4 h-4 mr-2" />
            New Chat
          </Button>
        </div>
        <div className="flex-1 overflow-y-auto p-4 space-y-2 scrollbar-thin scrollbar-thumb-accent/20 scrollbar-track-transparent">
          {conversations.map((conv) => (
            <div key={conv.id} className="flex gap-1 items-start group">
              <button
                onClick={() => setCurrentConversation(conv)}
                className={`flex-1 text-left p-3 rounded-xl transition-all duration-200 ${
                  currentConversation?.id === conv.id
                    ? 'bg-accent/20 text-accent font-medium shadow-md'
                    : 'text-foreground hover:bg-secondary/50 hover:backdrop-blur-sm'
                }`}
              >
                <div className="truncate text-sm font-medium">{conv.title}</div>
                <div className="text-xs text-muted-foreground">{conv.messages.length} messages</div>
              </button>
              <Button
                variant="ghost"
                size="icon"
                className="opacity-0 group-hover:opacity-100 h-8 w-8 shrink-0 text-muted-foreground"
                onClick={() => deleteConversation(conv.id)}
                aria-label="Delete conversation"
              >
                <Trash2 className="w-4 h-4" />
              </Button>
            </div>
          ))}
        </div>
      </div>

      <div className="flex-1 flex flex-col overflow-hidden">
        <div className="bg-card/50 dark:bg-card/40 backdrop-blur-md border-b border-border/30 px-4 sm:px-6 py-4 sticky top-0 z-20">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-xl sm:text-2xl font-bold text-foreground flex items-center gap-2">
                <Sparkles className="w-5 h-5 text-accent" />
                Chat
              </h1>
              <p className="text-xs sm:text-sm text-muted-foreground">Live answers from your Aria API (SQL-backed)</p>
            </div>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-3 sm:p-4 space-y-4 scrollbar-thin scrollbar-thumb-accent/20 scrollbar-track-transparent">
          {!currentConversation || currentConversation.messages.length === 0 ? (
            <div className="h-full flex flex-col items-center justify-center space-y-4 text-center px-4">
              <MessageCircle className="w-16 h-16 text-accent/60" />
              <h2 className="text-xl sm:text-2xl font-bold text-foreground mb-2">Start a conversation</h2>
              <p className="text-muted-foreground max-w-md text-sm sm:text-base">
                Ask about leads, tasks, and metrics. Use a real JWT from the login page.
              </p>
            </div>
          ) : (
            currentConversation.messages.map((msg, idx) => (
              <div
                key={msg.id}
                className={`flex ${msg.type === 'user' ? 'justify-end' : 'justify-start'} animate-slide-up`}
                style={{ animationDelay: `${idx * 50}ms` }}
              >
                <div
                  className={`max-w-xs sm:max-w-md lg:max-w-2xl px-4 py-3 rounded-2xl transition-all duration-200 ${
                    msg.type === 'user'
                      ? 'bg-gradient-to-br from-accent to-accent/80 text-accent-foreground rounded-br-none shadow-lg shadow-accent/20'
                      : 'bg-card/60 dark:bg-card/50 border border-border/50 backdrop-blur-sm rounded-bl-none hover:bg-card/80 dark:hover:bg-card/60'
                  }`}
                >
                  {msg.meta && (
                    <div className="flex flex-wrap gap-1.5 mb-2 text-[10px] uppercase tracking-wide text-muted-foreground">
                      {msg.meta.cached && <span className="px-1.5 py-0.5 rounded bg-secondary">Cached</span>}
                      {msg.meta.rowCount !== undefined && (
                        <span className="px-1.5 py-0.5 rounded bg-secondary">Rows: {msg.meta.rowCount}</span>
                      )}
                      {msg.meta.outOfScope && (
                        <span className="px-1.5 py-0.5 rounded bg-secondary">Out of scope</span>
                      )}
                    </div>
                  )}
                  {msg.type === 'ai' && msg.isLoading && !msg.content ? (
                    <div className="flex items-center gap-2 py-1">
                      <div className="flex gap-1">
                        <span className="w-2 h-2 bg-accent rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
                        <span className="w-2 h-2 bg-accent rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
                        <span className="w-2 h-2 bg-accent rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
                      </div>
                      <span className="text-xs text-muted-foreground">Thinking…</span>
                    </div>
                  ) : (
                    <p className="text-sm leading-relaxed break-words whitespace-pre-wrap">{msg.content}</p>
                  )}

                  {msg.type === 'ai' && msg.sql && (
                    <Collapsible className="mt-3 border-t border-border/30 pt-2">
                      <CollapsibleTrigger className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
                        <ChevronDown className="w-3 h-3" />
                        SQL executed
                      </CollapsibleTrigger>
                      <CollapsibleContent>
                        <pre className="mt-2 text-[11px] overflow-x-auto p-2 rounded bg-black/40 text-muted-foreground">
                          {msg.sql}
                        </pre>
                      </CollapsibleContent>
                    </Collapsible>
                  )}

                  {msg.type === 'ai' && !msg.isLoading && msg.id !== 'welcome' && (
                    <div className="flex gap-1 mt-3 pt-2 border-t border-border/30">
                      <Button
                        size="sm"
                        variant="ghost"
                        className="h-7 w-7 p-0 hover:bg-accent/10 rounded-lg transition-colors"
                        onClick={() => navigator.clipboard.writeText(msg.content).then(() => toast.success('Copied'))}
                      >
                        <Copy className="w-3.5 h-3.5" />
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        className="h-7 w-7 p-0 hover:bg-green-500/10 text-green-600 dark:text-green-400 rounded-lg transition-colors"
                        onClick={() => sendFeedback(msg, true)}
                        disabled={!msg.assistantMessageId}
                      >
                        <ThumbsUp className="w-3.5 h-3.5" />
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        className="h-7 w-7 p-0 hover:bg-red-500/10 text-red-600 dark:text-red-400 rounded-lg transition-colors"
                        onClick={() => sendFeedback(msg, false)}
                        disabled={!msg.assistantMessageId}
                      >
                        <ThumbsDown className="w-3.5 h-3.5" />
                      </Button>
                    </div>
                  )}
                </div>
              </div>
            ))
          )}
          <div ref={messagesEndRef} className="h-2" />
        </div>

        <div className="bg-card/50 dark:bg-card/40 backdrop-blur-md border-t border-border/30 p-3 sm:p-4">
          <div className="max-w-4xl mx-auto flex gap-2 sm:gap-3">
            <div className="flex-1 relative">
              <Input
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault()
                    handleSendMessage()
                  }
                }}
                placeholder="Ask about your leads (max 500 chars)…"
                className="flex-1 bg-secondary/50 dark:bg-secondary/30 border-border/50 rounded-xl focus:ring-accent/50 pr-10"
                disabled={isLoading}
              />
            </div>
            <Button
              onClick={handleSendMessage}
              disabled={isLoading || !inputValue.trim()}
              className="bg-gradient-to-r from-accent to-accent/80 hover:from-accent hover:to-accent/90 text-accent-foreground shadow-lg hover:shadow-accent/20 rounded-xl transition-all disabled:opacity-50"
              size="lg"
            >
              <Send className="w-4 h-4" />
            </Button>
          </div>
          <p className="text-xs text-muted-foreground mt-2 ml-2">
            Enter to send · Shift+Enter for new line · API: {getApiBaseUrl()}
          </p>
        </div>
      </div>
    </div>
  )
}
