'use client'

import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { ThemeToggle } from '@/components/theme-toggle'
import { AriaLogo } from '@/components/aria-logo'
import { ArrowRight } from 'lucide-react'

export function Navbar() {
  const router = useRouter()

  return (
    <nav className="sticky top-0 z-50 border-b border-border/40 backdrop-blur-xl bg-background/80">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10">
            <AriaLogo className="w-full h-full" />
          </div>
          <span className="text-xl font-bold bg-gradient-to-r from-accent via-pink-500 to-accent bg-clip-text text-transparent">
            Aria
          </span>
        </div>
        <div className="flex items-center gap-3">
          <ThemeToggle />
          <Button 
            variant="ghost" 
            onClick={() => router.push('/login')}
            className="hidden sm:flex"
          >
            Sign In
          </Button>
          <Button 
            onClick={() => router.push('/login')}
            className="bg-accent hover:bg-accent/90 text-accent-foreground"
          >
            Get Started
            <ArrowRight className="w-4 h-4 ml-2" />
          </Button>
        </div>
      </div>
    </nav>
  )
}
