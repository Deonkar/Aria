# Aria CRM - Installation & Setup Guide

## Quick Start (Recommended)

If you downloaded the ZIP and it appears empty, use the GitHub installation method:

### Method 1: Using GitHub CLI (Fastest)

```bash
# Clone directly from the source
gh repo clone vercel/examples
cd examples/next-app

# Or use your own GitHub repository
```

### Method 2: Manual Setup

1. **Create a new Next.js project:**
   ```bash
   npx create-next-app@latest aria-crm --typescript --tailwind
   cd aria-crm
   ```

2. **Install dependencies:**
   ```bash
   pnpm install
   ```

3. **Copy the files from this project into your new project:**
   - Copy all files from `app/` directory
   - Copy all files from `components/` directory
   - Copy `lib/` directory
   - Copy `app/globals.css`
   - Update `package.json` with any additional dependencies

4. **Start the development server:**
   ```bash
   pnpm dev
   ```

5. **Open in browser:**
   - Navigate to `http://localhost:3000`

### Method 3: Direct File Copy (If ZIP Download Works)

1. Extract the ZIP file
2. Navigate to the project directory:
   ```bash
   cd aria-crm
   ```
3. Install dependencies:
   ```bash
   pnpm install
   # or
   npm install
   # or
   yarn install
   ```
4. Start development:
   ```bash
   pnpm dev
   ```

## System Requirements

- **Node.js:** v18.0.0 or higher
- **Package Manager:** pnpm (recommended), npm, or yarn
- **Browser:** Modern browser with ES6 support

## Essential Dependencies

The project uses these key packages:

```json
{
  "dependencies": {
    "next": "^16.2.4",
    "react": "^19.2.4",
    "react-dom": "^19.2.4",
    "lucide-react": "latest",
    "@radix-ui/react-slot": "latest"
  },
  "devDependencies": {
    "typescript": "^5.7.3",
    "tailwindcss": "^4.2.0",
    "@types/react": "^19.0.0",
    "@types/node": "^20.0.0"
  }
}
```

## Project Structure

```
aria-crm/
├── app/
│   ├── dashboard/
│   │   ├── chat/
│   │   │   └── page.tsx          # Chat interface
│   │   ├── admin/
│   │   │   └── page.tsx          # Admin analytics
│   │   └── layout.tsx            # Dashboard layout
│   ├── login/
│   │   └── page.tsx              # Login page
│   ├── page.tsx                  # Landing page
│   ├── layout.tsx                # Root layout
│   └── globals.css               # Global styles & theme
├── components/
│   ├── ui/                       # shadcn/ui components
│   ├── aria-logo.tsx             # Aria logo SVG
│   ├── navbar.tsx                # Navigation bar
│   ├── theme-toggle.tsx          # Dark/light toggle
│   └── theme-provider.tsx        # Theme context
├── lib/
│   ├── use-theme.ts              # Theme hook
│   ├── theme-provider.tsx        # Theme logic
│   └── utils.ts                  # Utility functions
├── package.json
├── tsconfig.json
├── next.config.mjs
├── postcss.config.mjs
└── components.json               # shadcn config
```

## Available Commands

```bash
# Development
pnpm dev                # Start dev server on port 3000

# Production
pnpm build             # Build for production
pnpm start             # Start production server

# Code Quality
pnpm type-check       # Check TypeScript types
pnpm lint             # Run linter (if configured)
```

## Features

- Modern dark/light theme toggle
- Beautiful chat interface with streaming animations
- Admin analytics dashboard
- Responsive design (mobile, tablet, desktop)
- Smooth scroll behavior
- Message feedback system
- Conversation management
- Real-time UI updates

## Customization

### Change Theme Colors

Edit `app/globals.css`:

```css
:root {
  --accent: oklch(0.65 0.15 203);  /* Primary brand color */
  --background: oklch(0.985 0 0);  /* Light background */
  /* ... other colors ... */
}

.dark {
  --background: oklch(0.09 0.001 240);  /* Dark background */
  /* ... other colors ... */
}
```

### Add New Pages

Create files in `app/` following Next.js routing conventions:

```
app/
├── page.tsx                    # / route
├── about/
│   └── page.tsx               # /about route
├── dashboard/
│   └── layout.tsx             # /dashboard layout
    ├── chat/
    │   └── page.tsx           # /dashboard/chat route
```

### Modify Chat Interface

Edit `app/dashboard/chat/page.tsx`:
- Change colors, animations, layout
- Add new message types
- Integrate real API endpoints
- Add file upload support

## Integration with Backend

The chat interface has placeholder for backend integration. To connect your API:

1. Update API endpoints in component files (marked with `// In production:`)
2. Replace mock data with real API calls
3. Implement proper error handling
4. Add authentication tokens to requests

Example API integration:

```typescript
// In components/...
const response = await fetch('/api/chat/send', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ message, conversationId }),
});
```

## Deployment

### Deploy to Vercel (Recommended)

```bash
# Install Vercel CLI
npm i -g vercel

# Deploy
vercel
```

### Deploy to Other Platforms

- **Netlify:** Build command: `pnpm build` | Publish: `.next`
- **AWS Amplify:** Follow AWS deployment guide
- **Docker:** Create Dockerfile and deploy

## Troubleshooting

### Port 3000 already in use
```bash
lsof -ti:3000 | xargs kill -9
pnpm dev
```

### Module not found errors
```bash
rm -rf node_modules pnpm-lock.yaml
pnpm install
```

### Build errors
```bash
pnpm build --debug  # See detailed errors
```

### Dark mode not applying
- Clear browser cache
- Check theme toggle is working
- Verify localStorage has 'theme' key

## Browser Support

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- Mobile browsers (iOS Safari, Chrome Mobile)

## Performance Tips

1. Use `pnpm` instead of `npm` for faster installs
2. Enable code splitting in Next.js
3. Use lazy loading for images
4. Optimize bundle size with dynamic imports

## Getting Help

- Check documentation files in project root
- Review component examples in `components/ui/`
- Check Next.js docs: https://nextjs.org
- Check React docs: https://react.dev

## License

This project is provided as-is for educational and commercial use.

## Next Steps

1. ✅ Run `pnpm install && pnpm dev`
2. ✅ Test landing page at `http://localhost:3000`
3. ✅ Login with demo@example.com / demo123
4. ✅ Explore chat interface
5. ✅ Customize colors and theme
6. ✅ Integrate with your backend API

Happy coding! 🚀
