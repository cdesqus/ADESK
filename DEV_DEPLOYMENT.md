# AI-DESK Development Deployment Guide

**Deploy to development server for testing and team collaboration.**

---

## Prerequisites

- Development server with: Ubuntu 22.04 LTS, Docker, Docker Compose
- Git installed on both local machine and server
- SSH access to dev server
- Minimum 2GB RAM, 20GB storage

---

## Step 1: Initialize Git Repositories

### Local Machine (First Time Only)

```bash
cd D:\Demo\AI-DESK\backend
git init
git add .
git commit -m "Initial commit: Phase 1-4 complete - API, Email, WhatsApp, Reports"
git branch -M main

cd D:\Demo\AI-DESK\frontend
git init
git add .
git commit -m "Initial commit: Phase 1-4 complete - React Dashboard, Email, WhatsApp, Reports"
git branch -M main
```

### Setup Git Remote (if using GitHub/GitLab)

```bash
# Backend
cd D:\Demo\AI-DESK\backend
git remote add origin https://github.com/your-org/ai-desk-backend.git
git push -u origin main

# Frontend
cd D:\Demo\AI-DESK\frontend
git remote add origin https://github.com/your-org/ai-desk-frontend.git
git push -u origin main
```

**If no remote (local only):**
```bash
# Just skip the push, deploy directly via SCP instead
```

---

## Step 2: Prepare Development Server

### SSH into Server

```bash
ssh ubuntu@dev-server-ip
```

### Create App Directory

```bash
sudo mkdir -p /opt/ai-desk
sudo chown $USER:$USER /opt/ai-desk
cd /opt/ai-desk
```

### Install Docker & Dependencies

```bash
# Update system
sudo apt-get update
sudo apt-get upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
exit

# Login again to apply docker group
ssh ubuntu@dev-server-ip

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Verify
docker --version
docker-compose --version
```

---

## Step 3: Clone/Deploy Code

### Option A: From Git Remote (Recommended)

```bash
cd /opt/ai-desk

# Clone repositories
git clone https://github.com/your-org/ai-desk-backend.git backend
git clone https://github.com/your-org/ai-desk-frontend.git frontend

# Enter backend directory
cd backend
```

### Option B: Direct SCP Upload (No Git)

**From local machine:**
```bash
# Compress code
cd D:\Demo\AI-DESK
tar -czf ai-desk-backend.tar.gz backend/
tar -czf ai-desk-frontend.tar.gz frontend/

# SCP to server
scp ai-desk-backend.tar.gz ubuntu@dev-server-ip:/opt/ai-desk/
scp ai-desk-frontend.tar.gz ubuntu@dev-server-ip:/opt/ai-desk/

# Extract on server
ssh ubuntu@dev-server-ip
cd /opt/ai-desk
tar -xzf ai-desk-backend.tar.gz
tar -xzf ai-desk-frontend.tar.gz
```

---

## Step 4: Configure Environment

### Backend Configuration

```bash
cd /opt/ai-desk/backend

# Create .env file (copy from .env.example and update)
cat > .env << 'EOF'
# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=dev_password_123
DB_NAME=ai_desk
ENVIRONMENT=development

# JWT
JWT_SECRET=dev_secret_key_minimum_32_characters_long!

# Email (IMAP)
EMAIL_IMAP_HOST=imap.gmail.com
EMAIL_IMAP_PORT=993
EMAIL_USER=your-email@gmail.com
EMAIL_PASSWORD=your-app-password
EMAIL_POLLING_INTERVAL=5m

# Email (SMTP)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM_NAME=IDE SOLUSI INTEGRASI Support

# WhatsApp (Waha Plus)
WAHA_API_URL=http://waha:3000
WAHA_WEBHOOK_URL=http://backend:8000/api/whatsapp/webhook

# Port
PORT=8000
EOF

cat .env
```

### Frontend Configuration

```bash
cd /opt/ai-desk/frontend

# Create .env file
cat > .env << 'EOF'
VITE_API_URL=http://dev-server-ip:8000/api
VITE_APP_NAME=AI-DESK DEV
VITE_LOG_LEVEL=debug
EOF

cat .env
```

---

## Step 5: Setup Database

### Create docker-compose.yml Adjustments

For development, modify `docker-compose.yml` in backend:

```bash
cd /opt/ai-desk/backend

# Add port forwarding for direct DB access (development only)
cat >> docker-compose.yml << 'EOF'
# Add this to services.postgres section:
# ports:
#   - "5432:5432"  # Uncomment for dev - DO NOT use in production

# Add this to services.go-app section:
# environment:
#   - LOG_LEVEL=debug  # Verbose logging for dev
EOF
```

---

## Step 6: Start Services

### Backend

```bash
cd /opt/ai-desk/backend

# Pull Waha Plus image (using provided credentials)
docker login -u devlikeapro -p YOUR_WAHA_PAT_TOKEN_HERE
docker pull devlikeapro/waha-plus:latest
docker logout

# Note: Get the actual Waha Plus PAT token from your team lead or secure vault

# Start all services
docker-compose up -d

# Wait for services to be ready (30-60 seconds)
sleep 60

# Check status
docker-compose ps

# View logs
docker-compose logs -f go-app
```

**Expected output:**
```
postgres    | database system is ready to accept connections
redis       | Ready to accept connections
waha        | Waha API listening on port 3000
go-app      | Starting AI-DESK API Server on :8000
go-app      | Database connected successfully
go-app      | Email polling started
```

### Frontend (Separate Terminal)

```bash
cd /opt/ai-desk/frontend

# Install dependencies (first time only, ~3 minutes)
npm install

# Build for production
npm run build

# Serve built files
npm install -g serve
serve -s dist -l 5173

# Or run in dev mode (with hot reload)
npm run dev
```

**Expected output:**
```
✓ built in 5.23s
Ready on http://0.0.0.0:5173
```

---

## Step 7: Verify Deployment

### Health Checks

```bash
# Backend health
curl http://localhost:8000/health

# Expected: {"status":"ok"}

# Frontend
curl http://localhost:5173

# Should return HTML
```

### Database Check

```bash
# Check if migrations ran
docker-compose exec postgres psql -U postgres -d ai_desk -c "\dt"

# Should show tables: customers, engineers, tickets, updates, etc
```

### Email Integration Check

```bash
# Check if email polling started
docker-compose logs go-app | grep "Email polling"

# Should see: "Email polling started"
```

### WhatsApp Integration Check

```bash
# Check if Waha Plus is accessible
curl http://localhost:3000/

# Should return Waha Plus API info
```

---

## Step 8: Access Dashboard

### From Development Machine

```bash
# Frontend
http://dev-server-ip:5173

# Backend API
http://dev-server-ip:8000/api

# API Health
http://dev-server-ip:8000/health

# Waha Plus (internal only)
http://dev-server-ip:3000
```

### Default Login Credentials

```
Email: admin@idesolusi.co.id
Password: Admin123!
```

---

## Step 9: Setup Reverse Proxy (Optional but Recommended)

### Install Nginx

```bash
sudo apt-get install -y nginx

# Create config
sudo tee /etc/nginx/sites-available/ai-desk > /dev/null << 'EOF'
upstream backend {
    server localhost:8000;
}

upstream frontend {
    server localhost:5173;
}

server {
    listen 80;
    server_name dev-server-ip;

    # Frontend
    location / {
        proxy_pass http://frontend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }

    # API
    location /api {
        proxy_pass http://backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # Health check
    location /health {
        proxy_pass http://backend;
    }
}
EOF

sudo ln -s /etc/nginx/sites-available/ai-desk /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

**Access via:** `http://dev-server-ip` (port 80)

---

## Step 10: Test Complete Flow

### Test Email Integration

```bash
# Send test email to helpdesk@idesolusi.co.id
# (Configure in settings first with actual email account)

# Wait 5 minutes for polling
# Check dashboard - ticket should auto-create
```

### Test WhatsApp Integration

```bash
# Login to dashboard
# Go to Settings → WhatsApp Settings
# Click "Add WhatsApp Session"
# Scan QR code with WhatsApp
# Send test message to trigger ticket creation
```

### Test Report Generation

```bash
# Dashboard → Reports
# Select customer, month, year
# Click "Generate Report"
# Download CSV/PDF
# Verify metrics are accurate
```

---

## Troubleshooting

### Services Won't Start

```bash
# Check Docker status
docker-compose ps

# Check logs
docker-compose logs -f

# Restart everything
docker-compose down
docker-compose up -d
```

### Database Connection Error

```bash
# Wait longer for DB to initialize
sleep 30
docker-compose ps

# Check postgres logs
docker-compose logs postgres

# Reset database (WARNING: deletes data)
docker-compose down -v
docker-compose up -d postgres
sleep 30
docker-compose up -d
```

### Port Already in Use

```bash
# Find process using port
lsof -i :8000
lsof -i :5173
lsof -i :5432

# Kill process (if safe)
kill -9 PID

# Or modify docker-compose.yml to use different ports
```

### Email Not Working

```bash
# Check email config
cat .env | grep EMAIL

# Verify IMAP credentials (use app password, not Gmail password)
# Check email logs
docker-compose logs go-app | grep -i email

# Test SMTP directly
telnet smtp.gmail.com 587
```

### WhatsApp Not Receiving Messages

```bash
# Check Waha logs
docker-compose logs waha

# Verify webhook URL is correct
# Check if QR was scanned successfully
# Rescan if session disconnected
```

---

## Monitoring & Logs

### Real-Time Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f go-app
docker-compose logs -f waha
docker-compose logs -f postgres

# Last 100 lines
docker-compose logs --tail=100
```

### Database Queries

```bash
# Access PostgreSQL directly
docker-compose exec postgres psql -U postgres -d ai_desk

# Common queries
SELECT COUNT(*) FROM tickets;
SELECT COUNT(*) FROM customers;
SELECT * FROM email_logs ORDER BY created_at DESC LIMIT 10;
```

### Performance Monitoring

```bash
# Check resource usage
docker stats

# Check disk space
df -h

# Check memory
free -h
```

---

## Maintenance

### Backup Database

```bash
docker-compose exec postgres pg_dump -U postgres ai_desk > backup_$(date +%Y%m%d).sql
```

### Restore Database

```bash
docker-compose exec -T postgres psql -U postgres ai_desk < backup_20250601.sql
```

### Update Code

```bash
# Backend
cd /opt/ai-desk/backend
git pull origin main
docker-compose up -d --build

# Frontend
cd /opt/ai-desk/frontend
git pull origin main
npm install
npm run build
sudo systemctl restart nginx  # if using Nginx
```

### Clean Docker

```bash
# Remove unused images/volumes (CAREFUL!)
docker system prune -a --volumes

# Just remove dangling images
docker image prune
```

---

## Next Steps

1. ✅ Verify all services running
2. ✅ Test email integration (add email account)
3. ✅ Test WhatsApp integration (add session)
4. ✅ Generate sample report
5. ✅ Share dev URL with team
6. ✅ Gather feedback
7. ✅ Create production deployment plan

---

## Team Access

**Share with team:**
```
Development URL: http://dev-server-ip
Login: admin@idesolusi.co.id / Admin123!

Features to test:
- Dashboard & ticket management
- Email to helpdesk@ auto-create
- WhatsApp integration (add session + test)
- Report generation & download
- Settings pages (Email, WhatsApp, etc)
```

---

**Deployment Date:** June 1, 2025  
**Environment:** Development  
**Status:** Ready for testing
