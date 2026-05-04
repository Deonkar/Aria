# Aria CRM - Dark/Light Theme Guide

## Overview

Your Aria CRM frontend now features a modern **dark/light theme switcher** with Gen-Z aesthetic color palettes. The default theme is **dark mode**, but users can toggle between dark and light modes with a single click.

## Theme Colors

### Dark Mode (Default)
- **Background**: `oklch(0.08 0 0)` - Near-black (#140f1a)
- **Foreground**: `oklch(0.98 0 0)` - Off-white text
- **Card**: `oklch(0.12 0 0)` - Dark slate cards
- **Accent**: `oklch(0.65 0.15 203)` - Vibrant cyan-blue
- **Border**: `oklch(0.18 0 0)` - Dark borders

### Light Mode
- **Background**: `oklch(0.98 0 0)` - Clean white background
- **Foreground**: `oklch(0.15 0 0)` - Dark text for contrast
- **Card**: `oklch(0.99 0 0)` - Bright white cards
- **Accent**: `oklch(0.65 0.15 203)` - Same vibrant cyan-blue
- **Border**: `oklch(0.88 0.01 240)` - Light subtle borders

## Features

✨ **Theme Toggle Button**
- Located in the navigation bar (sun/moon icon)
- Smooth color transitions
- Persists user preference to localStorage

🎨 **Color Harmony**
- Consistent accent colors (cyan-blue) across both themes
- High contrast for accessibility
- Gen-Z aesthetic with trendy vibes

📱 **Responsive Design**
- Works perfectly on all screen sizes
- Mobile-friendly toggle button
- Touch-optimized interface

## Components

### 1. ThemeProvider (`lib/theme-provider.tsx`)
Wraps your entire application and manages theme state.

```tsx
<ThemeProvider>
  {children}
</ThemeProvider>
```

**Features:**
- Manages theme state (dark/light)
- Applies theme to `<html>` element
- Persists preference to localStorage
- Handles SSR properly

### 2. useTheme Hook (`lib/use-theme.ts`)
Use this hook to access the current theme and toggle function.

```tsx
const { theme, toggleTheme } = useTheme()
```

### 3. ThemeToggle Component (`components/theme-toggle.tsx`)
Reusable button component for theme switching.

```tsx
<ThemeToggle />
```

**Visual:**
- Shows ☀️ (sun) in dark mode
- Shows 🌙 (moon) in light mode
- Colored icons for visual feedback

## Usage Examples

### Access Current Theme
```tsx
'use client'
import { useTheme } from '@/lib/use-theme'

export function MyComponent() {
  const { theme } = useTheme()
  
  return <div>Current theme: {theme}</div>
}
```

### Toggle Theme Programmatically
```tsx
'use client'
import { useTheme } from '@/lib/use-theme'

export function ThemeButton() {
  const { toggleTheme } = useTheme()
  
  return <button onClick={toggleTheme}>Toggle Theme</button>
}
```

### Conditional Styling Based on Theme
```tsx
'use client'
import { useTheme } from '@/lib/use-theme'

export function ThemedBox() {
  const { theme } = useTheme()
  
  return (
    <div className={theme === 'dark' ? 'bg-dark-card' : 'bg-light-card'}>
      Content
    </div>
  )
}
```

## File Structure

```
lib/
├── theme-provider.tsx     # Theme context provider
└── use-theme.ts          # useTheme hook

components/
├── theme-toggle.tsx      # Toggle button component
├── navbar.tsx            # Navigation with theme toggle
└── aria-logo.tsx         # Modern Aria logo

app/
├── layout.tsx            # Wraps with ThemeProvider
├── globals.css           # Dark/light theme variables
└── page.tsx              # Home page using theme
```

## Customizing Colors

To customize the theme colors, edit `app/globals.css`:

### Light Mode Colors
```css
:root {
  --background: oklch(0.98 0 0);
  --foreground: oklch(0.15 0 0);
  --accent: oklch(0.65 0.15 203);
  /* ... more colors ... */
}
```

### Dark Mode Colors
```css
.dark {
  --background: oklch(0.08 0 0);
  --foreground: oklch(0.98 0 0);
  --accent: oklch(0.65 0.15 203);
  /* ... more colors ... */
}
```

## OKLch Color Format

Colors use the **OKLch color space** for better perceptual uniformity:

- **L** (Lightness): 0 (dark) to 1 (light)
- **C** (Chroma): Saturation intensity
- **h** (Hue): Color angle (0-360°)

Example: `oklch(0.65 0.15 203)` = Cyan-blue

## Browser Persistence

Theme preference is saved to localStorage:

```javascript
localStorage.getItem('theme')  // Returns 'dark' or 'light'
localStorage.setItem('theme', 'light')
```

## Accessibility

✅ **WCAG Compliant**
- High contrast ratios
- Clear visual indicators (sun/moon icons)
- Smooth transitions (no motion sickness)
- Keyboard accessible (tab navigation)

✅ **Reduced Motion Support**
- Consider adding `prefers-reduced-motion` media query

```css
@media (prefers-reduced-motion: reduce) {
  * {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

## Performance

✅ **Optimized for Speed**
- No unnecessary re-renders
- Efficient localStorage access
- Minimal color calculations
- Fast theme switching (instant UI update)

## Logo

The Aria logo (`components/aria-logo.tsx`) is an **SVG-based modern icon** featuring:
- Abstract AI wave design
- Gradient from cyan-blue to purple
- Responsive scaling
- Works in both light and dark modes

## Troubleshooting

### Theme Not Persisting?
- Check if localStorage is enabled
- Browser may be in private mode (localStorage disabled)
- Clear browser cache and try again

### Theme Toggle Not Working?
- Make sure the component is wrapped in `<ThemeProvider>`
- Check browser console for errors
- Verify JavaScript is enabled

### Styling Issues?
- Ensure `app/layout.tsx` has `suppressHydrationWarning`
- Check that CSS custom properties are properly defined
- Verify `@theme` block in `globals.css`

## Next Steps

1. ✅ **Run the app**: `pnpm dev`
2. ✅ **Visit**: http://localhost:3000
3. ✅ **Click the theme toggle** (sun/moon icon)
4. ✅ **Refresh the page** - theme preference persists!

## Additional Resources

- **Tailwind CSS Colors**: https://tailwindcss.com/docs/colors
- **OKLch Color Space**: https://www.w3.org/TR/css-color-4/#okch-colors
- **Next.js Dark Mode**: https://nextjs.org/docs/app/building-your-application/styling/dark-mode

---

**Your Aria CRM now has a beautiful, modern theme system with dark and light modes!**

Default: Dark Mode 🌙
Toggle: Click the sun/moon icon in the navbar ☀️🌙
Persistence: Your theme choice is saved automatically! 💾
