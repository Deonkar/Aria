'use client'

import { useRouter, usePathname } from 'next/navigation'
import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { MessageCircle, BarChart3, LogOut, Menu, X, Home } from 'lucide-react'

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const router = useRouter()
  const pathname = usePathname()
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [userName, setUserName] = useState('User')
  const [isAdmin, setIsAdmin] = useState(false)

  useEffect(() => {
    const token = localStorage.getItem('authToken')
    if (!token) {
      router.push('/login')
      return
    }

    const name = localStorage.getItem('userName') || 'User'
    const admin = localStorage.getItem('isAdmin') === 'true'
    setUserName(name)
    setIsAdmin(admin)
  }, [router])

  const handleLogout = () => {
    localStorage.removeItem('authToken')
    localStorage.removeItem('userEmail')
    localStorage.removeItem('userName')
    localStorage.removeItem('isAdmin')
    router.push('/login')
  }

  const navItems = [
    {
      href: '/dashboard/chat',
      label: 'Chat',
      icon: MessageCircle,
      show: true
    },
    {
      href: '/dashboard/admin',
      label: 'Analytics',
      icon: BarChart3,
      show: isAdmin
    }
  ]

  return (
    <div className="h-screen bg-background flex overflow-hidden">
      {/* Sidebar */}
      <aside
        className={`${
          sidebarOpen ? 'w-64' : 'w-0'
        } bg-card/30 border-r border-border/40 backdrop-blur-xl transition-all duration-300 flex flex-col overflow-hidden`}
      >
        <div className="p-6 border-b border-border/40">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-accent to-accent/70 flex items-center justify-center">
              <MessageCircle className="w-6 h-6 text-accent-foreground" />
            </div>
            <div className="flex-1">
              <p className="font-bold text-lg bg-gradient-to-r from-accent to-accent/70 bg-clip-text text-transparent">
                Aria
              </p>
              <p className="text-xs text-muted-foreground">CRM</p>
            </div>
          </div>
        </div>

        <nav className="flex-1 px-4 py-6 space-y-2 overflow-y-auto">
          {navItems.filter(item => item.show).map((item) => (
            <a
              key={item.href}
              href={item.href}
              className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-all ${
                pathname === item.href
                  ? 'bg-accent/20 text-accent border border-accent/30'
                  : 'text-muted-foreground hover:bg-secondary/40 border border-transparent'
              }`}
            >
              <item.icon className="w-5 h-5" />
              <span className="font-medium">{item.label}</span>
            </a>
          ))}
        </nav>

        <div className="p-4 border-t border-border/40 space-y-3">
          <div className="px-4 py-3 rounded-lg bg-secondary/40 border border-border/40">
            <p className="text-xs text-muted-foreground mb-1">Logged in as</p>
            <p className="font-semibold text-sm capitalize">{userName}</p>
            {isAdmin && <p className="text-xs text-accent">Admin</p>}
          </div>
          <Button
            onClick={handleLogout}
            variant="ghost"
            className="w-full justify-start text-red-500 hover:text-red-600 hover:bg-red-500/10"
          >
            <LogOut className="w-4 h-4 mr-2" />
            Sign Out
          </Button>
        </div>
      </aside>

      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Mobile Header */}
        <div className="md:hidden h-16 border-b border-border/40 bg-card/30 backdrop-blur-xl flex items-center justify-between px-4">
          <button
            onClick={() => setSidebarOpen(!sidebarOpen)}
            className="p-2 hover:bg-secondary/40 rounded-lg"
          >
            {sidebarOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
          </button>
          <div className="text-lg font-bold">Aria CRM</div>
          <div className="w-8" />
        </div>

        {/* Content */}
        <main className="flex-1 overflow-auto bg-background">
          {children}
        </main>
      </div>
    </div>
  )
}
