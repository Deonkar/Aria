# Aria CRM - Deployment Guide

## Vercel Deployment (Recommended)

Vercel is the optimal platform for Next.js applications. Deployment is seamless and includes:
- Automatic git integration
- Serverless functions for API routes
- Edge network for global performance
- Environment variable management
- Preview deployments for pull requests

### Step 1: Prepare Repository

```bash
# Initialize git if needed
git init

# Commit all changes
git add .
git commit -m "Initial Aria CRM commit"

# Push to GitHub (or your git provider)
git push origin main
```

### Step 2: Connect to Vercel

1. Go to [vercel.com](https://vercel.com)
2. Sign up or log in with GitHub account
3. Click "Add New" → "Project"
4. Import your GitHub repository
5. Select Next.js as the framework (auto-detected)
6. Click "Deploy"

### Step 3: Configure Environment Variables

In Vercel dashboard:

1. Go to Project Settings → Environment Variables
2. Add production variables:

```
NEXT_PUBLIC_GOOGLE_CLIENT_ID=your_google_client_id
NEXT_PUBLIC_API_URL=https://your-api.com
NODE_ENV=production
```

3. Add preview/development variables (optional):

```
NEXT_PUBLIC_API_URL=https://staging-api.com
```

4. Redeploy after adding variables

### Step 4: Configure Custom Domain (Optional)

1. Go to Project Settings → Domains
2. Add your custom domain
3. Update DNS records as instructed by Vercel
4. Wait for SSL certificate (typically a few minutes)

## Docker Deployment

For self-hosted deployments on servers or Kubernetes:

### Build Docker Image

Create `Dockerfile`:
```dockerfile
FROM node:18-alpine

WORKDIR /app

# Install dependencies
COPY package*.json ./
RUN npm install

# Build application
COPY . .
RUN npm run build

# Start production server
EXPOSE 3000
ENV NODE_ENV=production
CMD ["npm", "start"]
```

### Build and Run

```bash
# Build image
docker build -t aria-crm:latest .

# Run container
docker run -p 3000:3000 \
  -e NEXT_PUBLIC_API_URL=https://your-api.com \
  -e NEXT_PUBLIC_GOOGLE_CLIENT_ID=your_client_id \
  aria-crm:latest
```

### Docker Compose

Create `docker-compose.yml`:
```yaml
version: '3.8'

services:
  web:
    build: .
    ports:
      - "3000:3000"
    environment:
      - NEXT_PUBLIC_API_URL=https://your-api.com
      - NEXT_PUBLIC_GOOGLE_CLIENT_ID=your_client_id
      - NODE_ENV=production
    restart: unless-stopped
```

Run with:
```bash
docker-compose up -d
```

## AWS Deployment

### Using AWS Amplify

```bash
# Install Amplify CLI
npm install -g @aws-amplify/cli

# Initialize Amplify
amplify init

# Deploy
amplify publish
```

### Using EC2 + Nginx

1. Launch EC2 instance (Ubuntu 20.04 LTS)
2. SSH into instance
3. Install Node.js and npm
4. Clone repository
5. Install dependencies: `npm install`
6. Build: `npm run build`
7. Install PM2: `npm install -g pm2`
8. Start app: `pm2 start npm --name "aria-crm" -- start`
9. Install Nginx as reverse proxy
10. Configure SSL with Let's Encrypt

### EC2 Setup Script

```bash
#!/bin/bash
sudo apt update
sudo apt install -y nodejs npm nginx certbot python3-certbot-nginx

# Clone repo
git clone https://github.com/yourusername/aria-crm.git
cd aria-crm

# Install dependencies
npm install

# Build
npm run build

# Install PM2
sudo npm install -g pm2

# Start app
pm2 start npm --name "aria-crm" -- start
pm2 startup
pm2 save

# Configure Nginx
sudo nano /etc/nginx/sites-available/aria-crm
# Add reverse proxy config pointing to localhost:3000

# Enable SSL
sudo certbot --nginx -d yourdomain.com

# Restart Nginx
sudo systemctl restart nginx
```

## DigitalOcean Deployment

### Using App Platform

1. Log in to DigitalOcean
2. Create new App
3. Connect GitHub repository
4. Set build command: `npm run build`
5. Set run command: `npm start`
6. Add environment variables
7. Enable automatic deployments
8. Deploy

### Nginx Reverse Proxy Config

```nginx
server {
    listen 80;
    server_name yourdomain.com;

    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;

    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Cache static assets
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|woff|woff2|ttf|svg)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

## Environment Variables for Production

```env
# Frontend URLs
NEXT_PUBLIC_API_URL=https://api.yourdomain.com
NEXT_PUBLIC_APP_URL=https://app.yourdomain.com

# OAuth Configuration
NEXT_PUBLIC_GOOGLE_CLIENT_ID=your_google_client_id
NEXT_PUBLIC_ENABLE_GOOGLE_AUTH=true

# Error Tracking (Optional)
NEXT_PUBLIC_SENTRY_DSN=your_sentry_dsn

# Analytics (Optional)
NEXT_PUBLIC_ANALYTICS_ID=your_analytics_id

# Feature Flags
NEXT_PUBLIC_ENABLE_EMAIL_AUTH=true
NEXT_PUBLIC_ENABLE_ADMIN_PANEL=true

# Security
NODE_ENV=production
NEXT_TELEMETRY_DISABLED=1
```

## Pre-Deployment Checklist

- [ ] All environment variables configured
- [ ] Backend API endpoints updated in code
- [ ] Google OAuth credentials added to .env
- [ ] Production database configured
- [ ] HTTPS/SSL certificate installed
- [ ] Domain DNS records updated
- [ ] Error logging setup (Sentry, etc.)
- [ ] Analytics configured
- [ ] Backups strategy in place
- [ ] Monitoring and alerts configured
- [ ] Rate limiting implemented on backend
- [ ] CORS headers configured correctly

## Post-Deployment Verification

After deployment, verify:

1. **Homepage loads**: Visit https://yourdomain.com
2. **Login works**: Test with demo credentials
3. **Chat streams**: Send message and verify streaming
4. **Admin access**: Confirm admin can access metrics
5. **Error handling**: Check error pages display correctly
6. **Mobile responsive**: Test on mobile device
7. **Performance**: Check Lighthouse score
8. **SSL certificate**: Verify in browser
9. **API connectivity**: Monitor backend calls
10. **Logs accessible**: Check server logs for errors

## Monitoring & Maintenance

### Set Up Monitoring

```bash
# PM2 Plus for monitoring
pm2 install pm2-logrotate
pm2 logrotate
pm2 web  # Start PM2 web dashboard
```

### Regular Maintenance Tasks

- **Daily**: Check logs for errors
- **Weekly**: Review performance metrics
- **Monthly**: Update dependencies: `npm update`
- **Quarterly**: Major version updates: `npm audit`
- **Ongoing**: Monitor uptime and response times

### Backup Strategy

```bash
# Backup database daily
0 2 * * * /path/to/backup-script.sh

# Keep 30 days of backups
find /backups -name "*.backup" -mtime +30 -delete
```

### Log Rotation

Already configured with PM2, but can also use:

```bash
# Install logrotate
sudo apt install logrotate

# Configure rotation in /etc/logrotate.d/aria-crm
/var/log/aria-crm/*.log {
    daily
    rotate 30
    compress
    delaycompress
    notifempty
    create 0644 nobody nobody
}
```

## Rollback Procedure

If deployment has issues:

### Vercel Rollback
1. Go to Deployments
2. Find previous working deployment
3. Click "Promote to Production"

### Docker Rollback
```bash
# List images
docker images

# Run previous version
docker run -p 3000:3000 aria-crm:previous-tag

# Or with docker-compose
docker-compose down
git checkout previous-tag
docker-compose up -d
```

### Manual Rollback
```bash
# Kill current process
pm2 delete aria-crm

# Checkout previous code
git checkout previous-version
npm install
npm run build

# Start again
pm2 start npm --name "aria-crm" -- start
```

## Performance Optimization for Production

```bash
# Enable gzip compression in Next.js config
export const config = {
  compress: true,
}

# Build analysis
npm install -D @next/bundle-analyzer
npm run analyze
```

### Update next.config.mjs

```javascript
export default {
  // Enable compression
  compress: true,
  
  // Optimize images
  images: {
    formats: ['image/avif', 'image/webp'],
  },
  
  // Production optimizations
  productionBrowserSourceMaps: false,
  poweredByHeader: false,
}
```

## Scaling Considerations

For high-traffic deployments:

1. **Load Balancing**: Use Vercel's automatic scaling
2. **Database Scaling**: Implement read replicas
3. **Cache Layer**: Add Redis for session storage
4. **CDN**: Use Cloudflare or similar
5. **Monitoring**: Set up alerts for bottlenecks
6. **Auto-scaling**: Configure based on load metrics

## Troubleshooting Deployment Issues

### Build Fails
- Check Node version matches (18+)
- Clear cache: `rm -rf .next node_modules && npm install`
- Check for TypeScript errors: `npm run type-check`

### Deploy Timeout
- Increase build timeout in Vercel settings
- Optimize build process
- Split large components

### Environment Variables Not Available
- Verify variables are set in deployment platform
- Restart deployment after adding variables
- Check variable names match usage in code
- Use NEXT_PUBLIC_ prefix for client-side variables

### API Calls Failing
- Verify NEXT_PUBLIC_API_URL is correct
- Check CORS headers on backend
- Verify authentication tokens are valid
- Check network connectivity

## Support & Resources

- [Vercel Deployment Docs](https://vercel.com/docs/platform/deployments)
- [Next.js Production Checklist](https://nextjs.org/docs/going-to-production)
- [Node.js Best Practices](https://nodejs.org/en/docs/guides/nodejs-docker-webapp/)
- [Docker Documentation](https://docs.docker.com/)
- [Nginx Documentation](https://nginx.org/en/docs/)

## Getting Help

1. Check deployment platform documentation
2. Review server logs for error messages
3. Test API connectivity separately
4. Verify all environment variables
5. Check DNS configuration
6. Review firewall rules
7. Contact hosting platform support

Good luck with your deployment!
