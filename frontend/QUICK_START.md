# Aria CRM - Quick Start Guide

## 🚀 Get Started in 30 Seconds

### Step 1: Start the Server
```bash
cd /vercel/share/v0-project
pnpm dev
```

### Step 2: Open in Browser
Visit: **http://localhost:3000**

### Step 3: Login
Use demo credentials:
- Email: `demo@example.com`
- Password: `demo123`

**That's it!** You're in the dashboard.

---

## 📱 What to Explore

### Landing Page (`/`)
- Modern hero section with gradients
- Feature highlights
- Call-to-action buttons

### Login Page (`/login`)
- Try both demo accounts
- See form validation
- Experience smooth animations

### Chat Interface (`/dashboard/chat`)
- Send messages to AI
- Create new conversations
- Rate AI responses
- Copy messages

### Admin Dashboard (`/dashboard/admin`)
- View system metrics
- See activity charts
- Monitor system health
- Check top users

---

## 👥 Demo Accounts

| Role | Email | Password |
|------|-------|----------|
| Admin | demo@example.com | demo123 |
| User | user@example.com | user123 |

**Note**: Admin can see both chat and analytics. User can only see chat.

---

## 🎨 Design Features

✨ **Modern Dark Theme**
- Elegant near-black background
- Vibrant cyan-blue accents
- Professional typography
- Smooth animations

📱 **Fully Responsive**
- Works on all devices
- Mobile-optimized sidebar
- Touch-friendly buttons
- Adaptive layouts

⚡ **Interactive Elements**
- Hover effects
- Loading states
- Smooth transitions
- Glassmorphism effects

---

## 🔌 Backend Integration

All mock APIs are clearly marked in code:

```typescript
// Look for these comments:
// In production: replace with actual API call
// Mock: this is simulated, replace when backend ready
```

Key files to update:
- `/app/login/page.tsx` - Authentication
- `/app/dashboard/chat/page.tsx` - Chat messages
- `/app/dashboard/admin/page.tsx` - Analytics data

---

## 📁 Project Structure

```
/app
  /page.tsx          ← Landing page
  /login/page.tsx    ← Login page
  /dashboard
    /chat/page.tsx   ← Chat interface
    /admin/page.tsx  ← Analytics dashboard
  /layout.tsx        ← Root layout
  /globals.css       ← Theme & animations
```

---

## 🛠️ Useful Commands

```bash
# Development
pnpm dev              # Start dev server (localhost:3000)

# Production
pnpm build            # Create optimized build
pnpm start            # Start production server

# Utilities
pnpm type-check       # Check TypeScript errors
```

---

## 🎯 Common Tasks

### Change Theme Colors
Edit `/app/globals.css`:
```css
.dark {
  --accent: oklch(0.65 0.15 203);  /* Cyan-blue */
  --background: oklch(0.08 0 0);    /* Near-black */
}
```

### Add New Route
1. Create folder: `/app/dashboard/my-page/`
2. Create file: `/app/dashboard/my-page/page.tsx`
3. Add to sidebar in `/app/dashboard/layout.tsx`

### Change Accent Color
Search and replace in globals.css:
- `--accent: oklch(0.65 0.15 203);` with your color

---

## 📱 Mobile Testing

View on mobile:
1. Open DevTools (F12)
2. Toggle device toolbar (Ctrl+Shift+M)
3. Test different screen sizes

Or on actual phone:
1. Get your IP: `ipconfig getifaddr en0` (Mac) or `hostname -I` (Linux)
2. Visit: `http://YOUR_IP:3000`

---

## 🐛 Troubleshooting

### Port 3000 is in use?
```bash
kill $(lsof -ti:3000)
pnpm dev
```

### Styling not updating?
```bash
# Clear Next.js cache
rm -rf .next
pnpm dev
```

### Dependencies missing?
```bash
rm -rf node_modules pnpm-lock.yaml
pnpm install
pnpm dev
```

---

## 📚 Documentation

- **FINAL_SUMMARY.md** - Complete feature overview
- **API_GUIDE.md** - Backend integration (when ready)
- **SETUP.md** - Detailed installation

---

## 🎓 Learning Path

1. **Explore the UI** (5 min)
   - Click through all pages
   - Try both demo accounts
   - Test responsive design on mobile

2. **Understand the Code** (15 min)
   - Read `/app/page.tsx` - Landing page structure
   - Read `/app/dashboard/chat/page.tsx` - Chat logic
   - Read `/app/globals.css` - Theme system

3. **Customize** (20 min)
   - Change accent color
   - Modify hero section text
   - Add new menu items

4. **Integrate Backend** (2-4 hours)
   - Replace mock API calls
   - Connect to your backend
   - Update authentication

---

## ✅ Checklist

- [ ] Server is running (`pnpm dev`)
- [ ] Visited `http://localhost:3000`
- [ ] Logged in with demo account
- [ ] Tested chat interface
- [ ] Viewed admin dashboard
- [ ] Tested on mobile
- [ ] Explored code files

---

## 🚀 Ready to Deploy?

### To Vercel
```bash
git push origin main
# Deployment happens automatically
```

### To Other Platforms
See FINAL_SUMMARY.md for deployment guides

---

## 💡 Pro Tips

- Use Ctrl+Shift+M in DevTools for mobile view
- Try both demo accounts to see permission differences
- Check browser console for any errors (`F12`)
- All UI is responsive - try resizing the window
- Dark theme is automatic - your OS preference is respected

---

## 📞 Need Help?

All code is documented with comments. Look for:
- `// In production:` - Backend integration points
- `// TODO:` - Areas for customization
- `// Mock:` - Simulated functionality

---

**You're all set! Start with `pnpm dev` and explore.**

Questions? Check the code comments - they're very detailed!
