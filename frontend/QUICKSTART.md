# Aria CRM - Quick Start Checklist

## 5-Minute Setup

- [ ] Copy `.env.example` to `.env.local`
- [ ] Run `pnpm install`
- [ ] Run `pnpm dev`
- [ ] Open http://localhost:3000
- [ ] Click "Get Started" or "Go to Dashboard"

## First Test (Demo Credentials)

Login with either:
- **Admin Account**: demo@example.com / demo123
- **User Account**: user@example.com / user123

- [ ] Navigate to login page
- [ ] Enter credentials
- [ ] Click "Sign in with Email"
- [ ] Should redirect to chat dashboard
- [ ] Type a test message
- [ ] Watch response stream in real-time
- [ ] Click the helpful/unhelpful button
- [ ] See SQL query in the message

## Features to Explore

### Chat Features
- [ ] Open sidebar and see "New Chat" button
- [ ] Create a new conversation
- [ ] Send multiple messages
- [ ] See streaming response in real-time
- [ ] See metadata (SQL, row count, duration)
- [ ] Toggle sidebar visibility
- [ ] Browse previous conversations

### Admin Features (Login as admin)
- [ ] Navigate to Admin in sidebar
- [ ] See metrics dashboard loading
- [ ] View total conversations metric
- [ ] View total messages metric
- [ ] View active users metric
- [ ] See response time metric
- [ ] Check system status indicators
- [ ] Wait for auto-refresh (30 seconds)

### Responsive Design
- [ ] Open DevTools (F12)
- [ ] Toggle device toolbar
- [ ] Test mobile view (375px width)
- [ ] Test tablet view (768px width)
- [ ] Test desktop view (1920px width)
- [ ] Verify all elements responsive

## Next Steps - Backend Integration

### 1. Set Up Your Backend API

Update `.env.local`:
```env
NEXT_PUBLIC_API_URL=https://your-backend-api.com
```

### 2. Implement Required Endpoints

Follow **[API_INTEGRATION.md](./API_INTEGRATION.md)** to implement:

- [ ] `POST /api/auth/login` - Email/password login
- [ ] `POST /api/auth/google` - Google OAuth
- [ ] `GET /api/auth/verify` - Token verification
- [ ] `POST /api/chat/stream` - Chat streaming (SSE)
- [ ] `POST /api/chat/feedback` - Feedback storage
- [ ] `GET /api/admin/metrics` - Admin metrics

### 3. Test Each Endpoint

Use cURL or Postman:

```bash
# Test login
curl -X POST http://localhost:3000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'

# Test verify
curl -H "Authorization: Bearer {token}" \
  http://localhost:3000/api/auth/verify

# Test chat stream (should return SSE)
curl -X POST http://localhost:3000/api/chat/stream \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {token}" \
  -d '{"conversationId":"conv_123","message":"test"}'
```

### 4. Update Authentication

In `components/auth/login-form.tsx`:
- [ ] Remove demo alert from handleGoogleLogin
- [ ] Implement real Google OAuth integration
- [ ] Update handleEmailLogin to call real API
- [ ] Add error handling for production

In `lib/auth-context.tsx`:
- [ ] Update API endpoints to real backend URLs
- [ ] Implement real token verification
- [ ] Add proper error handling

## Production Deployment

### Choose Your Platform

- [ ] **Vercel** (Recommended) - See DEPLOYMENT.md
- [ ] **Docker** - See DEPLOYMENT.md
- [ ] **AWS** - See DEPLOYMENT.md
- [ ] **DigitalOcean** - See DEPLOYMENT.md

### Pre-Deployment

- [ ] Run `pnpm build` - verify no errors
- [ ] Run `pnpm lint` - check for issues
- [ ] Run `npm audit` - check dependencies
- [ ] Test all features locally
- [ ] Update environment variables for production
- [ ] Configure Google OAuth credentials
- [ ] Set up custom domain
- [ ] Configure SSL/HTTPS
- [ ] Set up monitoring

### Deploy

- [ ] Push code to GitHub (if using Vercel)
- [ ] Connect repository to deployment platform
- [ ] Add environment variables
- [ ] Deploy application
- [ ] Test deployed app
- [ ] Monitor for errors

## Customization (Optional)

### Change Theme Colors

Edit `app/globals.css`:

```css
.dark {
  --accent: oklch(0.65 0.15 203);  /* Change cyan-blue */
  --background: oklch(0.08 0 0);   /* Change background */
  --foreground: oklch(0.98 0 0);   /* Change text */
}
```

### Change Fonts

Edit `app/layout.tsx`:
- [ ] Import different Google Font
- [ ] Update font configuration
- [ ] Update tailwind.config.ts

### Add/Remove Features

- [ ] Remove admin dashboard if not needed
- [ ] Add conversation export feature
- [ ] Add user settings page
- [ ] Add analytics
- [ ] Add more integrations

## Documentation

Read in this order:

1. [ ] **SETUP.md** - Getting started details
2. [ ] **API_INTEGRATION.md** - API specification
3. [ ] **DEPLOYMENT.md** - Production deployment
4. [ ] **README.md** - Project overview
5. [ ] **DOCUMENTATION.md** - Complete guide index

## File Structure Overview

```
Key Files to Modify:

1. Authentication:
   - components/auth/login-form.tsx (UI)
   - lib/auth-context.tsx (State)
   - app/api/auth/* (Endpoints)

2. Chat:
   - components/chat/chat-interface.tsx (Main logic)
   - app/api/chat/stream/route.ts (Streaming)

3. Admin:
   - components/admin/admin-dashboard.tsx (Display)
   - app/api/admin/metrics/route.ts (Data)

4. Styling:
   - app/globals.css (Theme colors)
   - app/layout.tsx (Fonts)
```

## Troubleshooting Quick Links

### "Port 3000 already in use"
```bash
# Kill the process
lsof -ti:3000 | xargs kill -9
# Or use different port
pnpm dev -- -p 3001
```

### "Module not found" error
```bash
# Reinstall dependencies
rm -rf node_modules
pnpm install
```

### "Build fails with TypeScript error"
```bash
# Check for type errors
pnpm type-check
```

### "API calls return 401"
- [ ] Check token is being stored in localStorage
- [ ] Verify Authorization header is sent
- [ ] Check backend is running
- [ ] Verify CORS headers

### "Chat not streaming"
- [ ] Check Network tab in DevTools
- [ ] Verify /api/chat/stream returns 200
- [ ] Check Content-Type is text/event-stream
- [ ] Verify backend SSE implementation

## Success Indicators

When everything is working:

✅ App runs locally at http://localhost:3000  
✅ Login works with demo credentials  
✅ Chat sends messages and receives responses  
✅ Responses stream in real-time  
✅ Metadata (SQL, row count) appears  
✅ Feedback buttons work  
✅ Admin dashboard shows metrics  
✅ Responsive design works on mobile  
✅ Build completes without errors  
✅ No console errors in DevTools  

## Quick Commands Reference

```bash
# Development
pnpm dev              # Start dev server
pnpm build            # Build for production
pnpm start            # Run production build
pnpm lint             # Run linter
pnpm type-check       # Check TypeScript
npm audit             # Check dependencies

# Clean up
rm -rf .next
rm -rf node_modules
pnpm install
```

## Environment Variables Checklist

```env
# Required for local development:
NEXT_PUBLIC_API_URL=http://localhost:3000/api

# Optional - for Google OAuth:
NEXT_PUBLIC_GOOGLE_CLIENT_ID=your_client_id

# Required for production:
NEXT_PUBLIC_API_URL=https://your-api.com
NEXT_PUBLIC_GOOGLE_CLIENT_ID=prod_client_id
```

## Support Resources

**Still stuck?**

1. [ ] Check SETUP.md - Most answers are there
2. [ ] Review API_INTEGRATION.md - For backend questions
3. [ ] Look at code comments - Components have explanations
4. [ ] Check browser console - Error messages are helpful
5. [ ] Review Network tab - See what API calls are made
6. [ ] Check server logs - Run `pnpm dev` and watch terminal

## Project Timeline

**Phase 1: Local Setup** (5 minutes)
- [ ] Install and run locally
- [ ] Test with demo credentials

**Phase 2: Understand Code** (15 minutes)
- [ ] Read SETUP.md
- [ ] Explore project structure
- [ ] Review key components

**Phase 3: Backend Integration** (1-2 hours)
- [ ] Read API_INTEGRATION.md
- [ ] Implement required endpoints
- [ ] Test each endpoint
- [ ] Update .env.local
- [ ] Verify app still works

**Phase 4: Customization** (30 minutes - 2 hours)
- [ ] Update theme colors
- [ ] Change fonts
- [ ] Add custom features
- [ ] Update branding

**Phase 5: Deployment** (30 minutes - 2 hours)
- [ ] Follow DEPLOYMENT.md
- [ ] Configure environment
- [ ] Set up custom domain
- [ ] Test production app

## Total Time Estimate

- **Quick Start**: 5 minutes
- **Basic Setup**: 30 minutes
- **Full Integration**: 3-4 hours
- **Production Ready**: 5-6 hours

## Next: Where to Go

### Ready to integrate with backend?
→ Read: **[API_INTEGRATION.md](./API_INTEGRATION.md)**

### Ready to deploy?
→ Read: **[DEPLOYMENT.md](./DEPLOYMENT.md)**

### Need more details?
→ Read: **[SETUP.md](./SETUP.md)**

### Want complete overview?
→ Read: **[README.md](./README.md)**

---

**You're all set!** 🚀

Start with `pnpm install && pnpm dev` and explore the app with demo credentials.

Questions? Check [DOCUMENTATION.md](./DOCUMENTATION.md) for complete guide index.
