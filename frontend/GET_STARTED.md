# Aria CRM - Complete Setup Guide

## 🚀 Fastest Way to Get Started

The project is already live and running. You have multiple options to use it:

### **OPTION 1: Use the Live Preview (Fastest) ✅**

The project is already deployed and fully functional:
- Access the preview URL shown in v0 (top-right of screen)
- No installation needed
- Perfect for testing and demo

**URL Format:** `https://vm-xxxxx.vusercontent.net`

---

### **OPTION 2: Copy Files Manually (Reliable)**

If ZIP download has issues, copy files directly:

```bash
# 1. Create new Next.js project
npx create-next-app@latest aria-crm --typescript --tailwind

# 2. Navigate to project
cd aria-crm

# 3. Copy all files from v0 (from app/, components/, lib/ folders)
# You can use "View Source" in v0's three-dot menu to access files

# 4. Install dependencies
pnpm install

# 5. Run development server
pnpm dev
```

Then open `http://localhost:3000`

---

### **OPTION 3: Use GitHub (Most Professional)**

This is the recommended approach for production use:

```bash
# 1. Create GitHub repository at https://github.com/new
# 2. Clone it locally
git clone https://github.com/YOUR_USERNAME/aria-crm.git
cd aria-crm

# 3. Copy all project files from v0 into this directory

# 4. Commit and push
git add .
git commit -m "Initial Aria CRM commit"
git push origin main

# 5. Install and run
pnpm install
pnpm dev
```

**Benefits:**
- Version control
- Easy deployment to Vercel
- Can share code with team
- Professional workflow

---

### **OPTION 4: Use the Provided Setup Script**

If you did download the ZIP (or copied files manually):

```bash
# Make it executable
chmod +x setup.sh

# Run setup script
./setup.sh
```

This automatically detects your package manager and installs dependencies.

---

## 📋 What to Do If ZIP Download is Empty

This is a known v0 platform limitation. Use these alternatives:

### Solution A: Use View Source
1. In v0, click the three-dot menu
2. Select "View Source"
3. Copy the files you need
4. Paste into a new Next.js project

### Solution B: Use GitHub
1. Create a GitHub repository
2. Add the source files
3. Clone locally
4. Work from there

### Solution C: Use the Preview
1. The preview URL in v0 is already functional
2. Test everything in the live preview
3. Share the preview URL with team

---

## 🎯 Login Credentials

Once you start the project:

**Admin Account:**
- Email: `demo@example.com`
- Password: `demo123`

**User Account:**
- Email: `user@example.com`
- Password: `user123`

---

## 📚 Key Files to Know

After setting up, here are the main files:

```
app/
├── page.tsx                 # Landing page
├── layout.tsx              # Root layout with theme
├── globals.css             # Styles & colors
├── login/page.tsx          # Login interface
└── dashboard/
    ├── layout.tsx          # Dashboard wrapper
    ├── chat/page.tsx       # Modern chat interface
    └── admin/page.tsx      # Analytics dashboard

components/
├── aria-logo.tsx           # Aria logo SVG
├── navbar.tsx              # Navigation with theme toggle
├── theme-toggle.tsx        # Dark/light mode button
└── ui/                     # shadcn UI components
```

---

## ⚙️ Available Commands

```bash
# Start development server (port 3000)
pnpm dev

# Build for production
pnpm build

# Start production server
pnpm start

# Check TypeScript types
pnpm type-check
```

---

## 🎨 Customization Quick Tips

### Change Theme Colors
Edit `app/globals.css`:
```css
:root {
  --accent: oklch(0.65 0.15 203);  /* Change primary color */
}
```

### Modify Chat Interface
Edit `app/dashboard/chat/page.tsx`:
- Change colors, fonts, spacing
- Add/remove features
- Integrate your API

### Add New Pages
Create files in `app/`:
```bash
app/pricing/page.tsx    # Creates /pricing route
app/about/page.tsx      # Creates /about route
```

---

## 🌐 Deploy to Production

### Deploy to Vercel (1-Click)

```bash
npm install -g vercel
vercel
```

Or:
1. Push code to GitHub
2. Go to https://vercel.com/new
3. Import your repository
4. Click Deploy

### Deploy to Other Platforms

- **Netlify:** Connect GitHub repo
- **AWS:** Use AWS Amplify
- **Docker:** Create Dockerfile and deploy
- **Self-hosted:** Run `pnpm build && pnpm start`

---

## 🔧 Troubleshooting

### Port 3000 in use?
```bash
lsof -ti:3000 | xargs kill -9
pnpm dev
```

### Dependencies not installing?
```bash
rm -rf node_modules pnpm-lock.yaml
pnpm install
```

### Dark mode not working?
- Clear browser cache
- Check if theme toggle button appears
- Verify localStorage is enabled

### Build fails?
```bash
pnpm build --debug
```

---

## 📖 Documentation Files

The project includes these documentation files:

- **INSTALL.md** - Detailed installation guide
- **GITHUB_SETUP.md** - GitHub integration guide
- **THEME_GUIDE.md** - Theme customization
- **This file** - Quick start overview

---

## ✨ Features Included

✅ Modern dark/light theme with toggle  
✅ Beautiful responsive chat interface  
✅ Smooth animations & scroll behavior  
✅ Admin analytics dashboard  
✅ Message feedback system  
✅ Conversation management  
✅ Mobile responsive design  
✅ Professional color schemes  
✅ Accessibility features  
✅ Production-ready code  

---

## 🎓 Learning Resources

- **Next.js:** https://nextjs.org/docs
- **React:** https://react.dev
- **Tailwind CSS:** https://tailwindcss.com/docs
- **shadcn/ui:** https://ui.shadcn.com
- **Lucide Icons:** https://lucide.dev

---

## 🚀 Next Steps

### Immediately:
1. Access the live preview (fastest)
2. Test all features
3. Try dark/light theme toggle
4. Login with demo credentials

### Soon:
1. Copy files to GitHub
2. Customize colors & branding
3. Add your own content
4. Integrate with your API

### Eventually:
1. Connect to backend
2. Add real authentication
3. Deploy to production
4. Share with team

---

## 💡 Pro Tips

1. **Use pnpm** instead of npm for faster installs
2. **Keep theme colors consistent** by editing globals.css once
3. **Use GitHub** from the start for easy collaboration
4. **Deploy to Vercel** - it's free and fast
5. **Test on mobile** - responsive design is built-in

---

## 📞 Support

- Check documentation files in project
- Review component examples in `components/ui/`
- Refer to Next.js documentation
- Check React documentation

---

## Summary

| Option | Speed | Setup | Reliability |
|--------|-------|-------|-------------|
| Live Preview | ⚡⚡⚡ | None | ✅ |
| Copy Files | ⚡⚡ | 5 min | ✅ |
| GitHub | ⚡ | 10 min | ✅✅ |
| ZIP Download | ⚡⚡ | 5 min | ⚠️ (issues) |

**Recommendation:** Start with the live preview, then use GitHub for long-term development.

---

**Ready to build?** Pick an option above and get started! 🚀
