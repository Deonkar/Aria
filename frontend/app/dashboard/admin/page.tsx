'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { BarChart3, Users, MessageSquare, TrendingUp, Activity, Lock } from 'lucide-react'

interface Metric {
  label: string
  value: string
  change: number
  icon: React.ReactNode
}

export default function AdminPage() {
  const router = useRouter()
  const [metrics, setMetrics] = useState<Metric[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const isAdmin = localStorage.getItem('isAdmin') === 'true'
    if (!isAdmin) {
      router.push('/dashboard/chat')
      return
    }

    // Simulate data loading
    const timer = setTimeout(() => {
      setMetrics([
        {
          label: 'Total Conversations',
          value: '1,247',
          change: 12,
          icon: <MessageSquare className="w-6 h-6 text-accent" />
        },
        {
          label: 'Active Users',
          value: '458',
          change: 8,
          icon: <Users className="w-6 h-6 text-accent" />
        },
        {
          label: 'Avg Response Time',
          value: '340ms',
          change: -15,
          icon: <TrendingUp className="w-6 h-6 text-accent" />
        },
        {
          label: 'System Health',
          value: '99.8%',
          change: 0.2,
          icon: <Activity className="w-6 h-6 text-accent" />
        }
      ])
      setLoading(false)
    }, 800)

    return () => clearTimeout(timer)
  }, [router])

  return (
    <div className="h-full overflow-auto bg-background">
      {/* Header */}
      <div className="sticky top-0 z-40 border-b border-border/40 bg-card/30 backdrop-blur-xl">
        <div className="max-w-7xl mx-auto px-4 md:px-8 py-6">
          <div className="flex items-center gap-3 mb-2">
            <BarChart3 className="w-8 h-8 text-accent" />
            <h1 className="text-3xl font-bold">Analytics Dashboard</h1>
          </div>
          <p className="text-muted-foreground">Monitor system performance and user metrics</p>
        </div>
      </div>

      {/* Content */}
      <div className="max-w-7xl mx-auto px-4 md:px-8 py-8 space-y-8">
        {/* Metrics Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {metrics.map((metric, i) => (
            <div
              key={i}
              className="p-6 rounded-2xl border border-border/40 bg-card/30 backdrop-blur-sm hover:border-accent/30 transition-all group"
            >
              <div className="flex items-start justify-between mb-4">
                <div className="p-3 rounded-lg bg-accent/10 group-hover:bg-accent/20 transition-colors">
                  {metric.icon}
                </div>
                <span className={`text-sm font-semibold ${
                  metric.change > 0 ? 'text-green-500' : metric.change < 0 ? 'text-orange-500' : 'text-gray-500'
                }`}>
                  {metric.change > 0 ? '↑' : metric.change < 0 ? '↓' : '→'} {Math.abs(metric.change)}%
                </span>
              </div>
              <p className="text-muted-foreground text-sm mb-2">{metric.label}</p>
              <p className="text-3xl font-bold">{metric.value}</p>
            </div>
          ))}
        </div>

        {/* Charts Section */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Activity Chart */}
          <div className="p-6 rounded-2xl border border-border/40 bg-card/30 backdrop-blur-sm">
            <h3 className="text-lg font-semibold mb-6">Conversation Activity (Last 7 Days)</h3>
            <div className="space-y-4">
              {[
                { day: 'Mon', messages: 245, fill: 60 },
                { day: 'Tue', messages: 312, fill: 75 },
                { day: 'Wed', messages: 287, fill: 70 },
                { day: 'Thu', messages: 401, fill: 95 },
                { day: 'Fri', messages: 456, fill: 100 },
                { day: 'Sat', messages: 234, fill: 55 },
                { day: 'Sun', messages: 198, fill: 45 }
              ].map((item, i) => (
                <div key={i} className="flex items-center gap-4">
                  <span className="w-8 text-xs font-medium text-muted-foreground">{item.day}</span>
                  <div className="flex-1 h-8 bg-secondary/30 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-gradient-to-r from-accent to-accent/60 rounded-full transition-all duration-500"
                      style={{ width: `${item.fill}%` }}
                    />
                  </div>
                  <span className="text-sm font-medium w-12 text-right">{item.messages}</span>
                </div>
              ))}
            </div>
          </div>

          {/* System Status */}
          <div className="p-6 rounded-2xl border border-border/40 bg-card/30 backdrop-blur-sm">
            <h3 className="text-lg font-semibold mb-6">System Status</h3>
            <div className="space-y-4">
              {[
                { name: 'API Server', status: 'Operational', uptime: 99.98 },
                { name: 'Database', status: 'Operational', uptime: 99.95 },
                { name: 'Cache Layer', status: 'Operational', uptime: 99.99 },
                { name: 'Message Queue', status: 'Operational', uptime: 99.90 }
              ].map((item, i) => (
                <div key={i} className="flex items-center justify-between p-3 rounded-lg bg-background/50 border border-border/40">
                  <div>
                    <p className="font-medium text-sm">{item.name}</p>
                    <p className="text-xs text-muted-foreground mt-1">Uptime: {item.uptime}%</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
                    <span className="text-xs font-medium text-green-500">{item.status}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Detailed Metrics */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Top Users */}
          <div className="p-6 rounded-2xl border border-border/40 bg-card/30 backdrop-blur-sm">
            <h3 className="text-lg font-semibold mb-6">Top Users</h3>
            <div className="space-y-4">
              {[
                { name: 'Alex Johnson', conversations: 45 },
                { name: 'Sarah Williams', conversations: 38 },
                { name: 'Mike Chen', conversations: 32 },
                { name: 'Emma Davis', conversations: 28 }
              ].map((user, i) => (
                <div key={i} className="flex items-center justify-between p-3 rounded-lg bg-background/50 border border-border/40">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-full bg-accent/20 flex items-center justify-center">
                      <span className="text-xs font-bold text-accent">{user.name[0]}</span>
                    </div>
                    <span className="text-sm font-medium">{user.name}</span>
                  </div>
                  <span className="text-xs text-muted-foreground">{user.conversations}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Popular Queries */}
          <div className="p-6 rounded-2xl border border-border/40 bg-card/30 backdrop-blur-sm">
            <h3 className="text-lg font-semibold mb-6">Top Queries</h3>
            <div className="space-y-4">
              {[
                { query: 'Customer trends', count: 234 },
                { query: 'Sales analysis', count: 189 },
                { query: 'Churn prediction', count: 156 },
                { query: 'Revenue forecast', count: 142 }
              ].map((item, i) => (
                <div key={i} className="p-3 rounded-lg bg-background/50 border border-border/40">
                  <p className="text-sm font-medium truncate">{item.query}</p>
                  <p className="text-xs text-muted-foreground mt-1">{item.count} queries</p>
                </div>
              ))}
            </div>
          </div>

          {/* Security Info */}
          <div className="p-6 rounded-2xl border border-border/40 bg-card/30 backdrop-blur-sm">
            <div className="flex items-center gap-3 mb-6">
              <Lock className="w-5 h-5 text-accent" />
              <h3 className="text-lg font-semibold">Security Status</h3>
            </div>
            <div className="space-y-4">
              {[
                { label: 'Data Encryption', status: 'Enabled' },
                { label: 'SSL Certificate', status: 'Valid' },
                { label: 'Auth Protocol', status: 'OAuth 2.0' },
                { label: 'Compliance', status: 'GDPR' }
              ].map((item, i) => (
                <div key={i} className="flex items-center justify-between p-2">
                  <span className="text-sm">{item.label}</span>
                  <span className="text-xs font-medium text-accent bg-accent/10 px-2 py-1 rounded-full">
                    {item.status}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
