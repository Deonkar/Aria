#!/bin/bash

echo "🚀 Aria CRM - Quick Setup"
echo "========================="
echo ""

# Check Node.js
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js v18+"
    exit 1
fi

echo "✅ Node.js version: $(node -v)"

# Check for package manager
if command -v pnpm &> /dev/null; then
    PM="pnpm"
    echo "✅ Using pnpm"
elif command -v yarn &> /dev/null; then
    PM="yarn"
    echo "✅ Using yarn"
else
    PM="npm"
    echo "✅ Using npm"
fi

echo ""
echo "📦 Installing dependencies..."
$PM install

echo ""
echo "✅ Setup complete!"
echo ""
echo "🎯 Next steps:"
echo "   1. Run: $PM dev"
echo "   2. Open: http://localhost:3000"
echo "   3. Login with: demo@example.com / demo123"
echo ""
echo "📚 Documentation: Check INSTALL.md for more details"
echo ""
