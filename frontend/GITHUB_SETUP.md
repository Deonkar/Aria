# Using GitHub to Get the Project (Recommended)

If the ZIP download is having issues, use GitHub for a more reliable installation.

## Option A: Clone from Your Own GitHub Repository

### 1. Create a GitHub Repository

1. Go to https://github.com/new
2. Name it: `aria-crm`
3. Initialize with README (optional)
4. Click "Create repository"

### 2. Clone the Repository Locally

```bash
git clone https://github.com/YOUR_USERNAME/aria-crm.git
cd aria-crm
```

### 3. Get Files from v0

1. In v0, use the three-dot menu → "View Source" 
2. Copy all files from the project
3. Or download individual files from the Components/Pages sections

### 4. Add Files to Your Local Repository

```bash
# Copy the files into your cloned directory
# Then commit and push

git add .
git commit -m "Initial Aria CRM commit"
git push origin main
```

### 5. Install and Run

```bash
pnpm install
pnpm dev
```

Visit `http://localhost:3000`

---

## Option B: Use the Vercel Project Directly

Since the project is already built and running on Vercel preview:

1. The project is already deployed and accessible
2. Use the preview URL provided by v0
3. This is the fastest way to see the finished product

### Get the Preview URL:
- Look in the top-right of v0 where it shows the preview
- The URL will be something like: `https://vm-xxxxx.vusercontent.net`

---

## Option C: Copy Files Manually

### Step 1: Create New Next.js Project

```bash
npx create-next-app@latest aria-crm \
  --typescript \
  --tailwind \
  --eslint \
  --app \
  --no-git

cd aria-crm
```

### Step 2: Add Components

Copy these files from v0 into your project:

**From `app/` directory:**
- `layout.tsx`
- `globals.css`
- `page.tsx`
- `login/page.tsx`
- `dashboard/layout.tsx`
- `dashboard/chat/page.tsx`
- `dashboard/admin/page.tsx`

**From `components/` directory:**
- `aria-logo.tsx`
- `navbar.tsx`
- `theme-toggle.tsx`
- `theme-provider.tsx`
- All files from `components/ui/`

**From `lib/` directory:**
- `use-theme.ts`
- `theme-provider.tsx`

### Step 3: Install shadcn UI Components

```bash
npx shadcn-ui@latest init -d

# Add button, input, and other components as needed
npx shadcn-ui@latest add button
npx shadcn-ui@latest add input
```

### Step 4: Update Dependencies

Make sure your `package.json` has:

```json
{
  "dependencies": {
    "next": "^16.2.4",
    "react": "^19.2.4",
    "react-dom": "^19.2.4",
    "lucide-react": "latest"
  },
  "devDependencies": {
    "typescript": "^5.7.3",
    "tailwindcss": "^4.2.0"
  }
}
```

Then:
```bash
pnpm install
```

### Step 5: Run the Project

```bash
pnpm dev
```

Visit `http://localhost:3000`

---

## Recommended: Download as ZIP from GitHub

Once you have the files in a GitHub repository:

1. Go to your GitHub repository
2. Click the green "Code" button
3. Select "Download ZIP"
4. Extract the ZIP
5. Run `pnpm install && pnpm dev`

This avoids the v0 ZIP issue entirely!

---

## Vercel Deployment

Once your project is on GitHub, deploy with one click:

```bash
npm install -g vercel
vercel
```

Or use the Vercel dashboard:
1. Go to https://vercel.com/new
2. Import your GitHub repository
3. Deploy

---

## Troubleshooting

### Git not initialized?
```bash
git init
git add .
git commit -m "Initial commit"
git branch -M main
git remote add origin https://github.com/YOUR_USERNAME/aria-crm.git
git push -u origin main
```

### Files missing?
- Check that all folders exist: `app/`, `components/`, `lib/`
- Verify `package.json` has all required dependencies
- Run `pnpm install` again

### Still having issues?
- Check Node.js version: `node -v` (should be v18+)
- Clear cache: `rm -rf node_modules && pnpm install`
- Check build: `pnpm build`

---

## Quick Reference

```bash
# Clone (if you have it on GitHub)
git clone https://github.com/YOUR_USERNAME/aria-crm.git
cd aria-crm

# Install
pnpm install

# Develop
pnpm dev

# Build
pnpm build

# Deploy to Vercel
vercel
```

Happy coding! 🚀
