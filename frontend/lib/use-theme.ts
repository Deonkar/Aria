'use client'

import { useContext } from 'react'
import { ThemeContext } from './theme-provider'

export function useTheme() {
  const context = useContext(ThemeContext)
  if (!context) {
    // Return a default value during SSR
    return {
      theme: 'dark' as const,
      toggleTheme: () => {},
    }
  }
  return context
}
