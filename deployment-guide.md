# AI-DESK Deployment Guide

**Complete step-by-step guide to run AI-DESK locally and deploy to production.**

---

## Prerequisites

Before starting, ensure you have:
- Docker & Docker Compose installed (`docker --version`, `docker-compose --version`)
- Git installed
- Node.js 18+ installed (`node --version`)
- Go 1.22+ installed (`go version`)

**If not installed:** Download from [docker.com](https://docker.com), [nodejs.org](https://nodejs.org), [golang.org](https://golang.org)

---

## Local Development Setup (5 Minutes)

### Step 1: Navigate to Project

```bash
cd D:\Demo\AI-DESK
```

### Step 2: Start Backend Services

```bash
cd backend
docker-compose up -d
```

**What this does:**
- Starts PostgreSQL (port 5432)
- Starts Redis (port 6379)
- Creates database & runs migrations automatically

**Check if running:**
```bash
docker-compose ps
# You should see: postgres, redis, and go-app all "Up"
```

### Step 3: Run Backend Server

```bash
# Still in backend folder
go run cmd/main.go
```

**Expected output:**
```
2025/06/01 10:30:00 Starting AI-DESK API Server on :8000
2025/06/01 10:30:00 Database connected successfully
```

**Backend ready at:** `http://localhost:8000`
**Health check:** `curl http://localhost:8000/health`

### Step 4: Setup & Run Frontend (New Terminal)

```bash
cd D:\Demo\AI-DESK\frontend
npm install
npm run dev
```

**Expected output:**
```
VITE v5.0.0 ready in 100 ms

➜  Local:   http://localhost:5173/
```

**Frontend ready at:** `http://localhost:5173`

### Step 5: Test Login

1. Open browser: `http://localhost:5173`
2. You should see login page
3. Default test credentials:
   - Email: `admin@idesolusi.co.id`
   - Password: `Admin123!`
4. Click Login → Should redirect to Dashboard

**✅ If you see the dashboard with empty ticket list, setup is successful!**

---

## Database Access (Optional)

To manually access PostgreSQL:

```bash
docker exec -it ai-desk-postgres psql -U postgres -d ai_desk
```

**Common commands:**
```sql
-- View all tables
\dt

-- View customers
SELECT * FROM customers;

-- View tickets
SELECT * FROM tickets;

-- Exit
\q
```

---

## Stop & Clean Up

```bash
# Stop backend
# Press Ctrl+C in the backend terminal

# Stop frontend
# Press Ctrl+C in the frontend terminal

# Stop Docker services
cd backend
docker-compose down

# To remove database (WARNING: deletes all data)
docker-compose down -v
```

---

## Production Deployment

### Option A: AWS EC2 (Recommended)

#### 1. Provision EC2 Instance
- Instance type: `t3.medium` (2 CPU, 4GB RAM)
- OS: Ubuntu 22.04 LTS
- Security group: Open ports 80, 443, 22

#### 2. SSH into Server

```bash
ssh -i your-key.pem ubuntu@your-ec2-ip
```

#### 3. Install Docker & Docker Compose

```bash
# Update system
sudo apt-get update
sudo apt-get upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker ubuntu
exit

# Login again
ssh -i your-key.pem ubuntu@your-ec2-ip

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

#### 4. Clone & Configure

```bash
cd ~
git clone <your-git-repo> ai-desk
cd ai-desk/backend

# Create .env file with production values
cat > .env << EOF
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=YourSecurePassword123!
DB_NAME=ai_desk
JWT_SECRET=YourSecureJWTSecret_MinimumLength32Chars!
ENVIRONMENT=production
EOF

# Ensure frontend can reach backend
# Edit frontend/.env for API URL:
# VITE_API_URL=https://your-domain.com/api
```

#### 5. Update Docker Compose for Production

Edit `docker-compose.yml` - change Go service:

```yaml
  go-app:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8000:8000"
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=postgres
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=ai_desk
      - JWT_SECRET=${JWT_SECRET}
      - ENVIRONMENT=production
    depends_on:
      - postgres
      - redis
```

#### 6. Setup Nginx Reverse Proxy

```bash
sudo apt-get install -y nginx

# Create Nginx config
sudo tee /etc/nginx/sites-available/ai-desk > /dev/null << 'EOF'
server {
    listen 80;
    server_name your-domain.com;

    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;

    # Frontend
    location / {
        root /var/www/ai-desk/frontend/dist;
        try_files $uri $uri/ /index.html;
    }

    # Backend API
    location /api {
        proxy_pass http://localhost:8000/api;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
EOF

sudo ln -s /etc/nginx/sites-available/ai-desk /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

#### 7. Setup SSL with Let's Encrypt

```bash
sudo apt-get install -y certbot python3-certbot-nginx
sudo certbot certonly --nginx -d your-domain.com
# Follow prompts
```

#### 8. Start Services

```bash
cd ~/ai-desk

# Build and start
docker-compose up -d

# Check status
docker-compose logs -f
```

**Production URL:** `https://your-domain.com`

---

### Option B: Google Cloud Run (Simpler)

#### 1. Prepare Dockerfile (Already included)

Ensure `backend/Dockerfile` exists. It should:
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ai-desk cmd/main.go

FROM alpine:latest
COPY --from=builder /app/ai-desk .
EXPOSE 8000
CMD ["./ai-desk"]
```

#### 2. Build & Push to Google Artifact Registry

```bash
# Setup GCP
gcloud auth login
gcloud config set project YOUR_PROJECT_ID

# Create repository
gcloud artifacts repositories create ai-desk --location=us-central1 --repository-format=docker

# Build & push
gcloud builds submit --region=us-central1 backend/ --tag us-central1-docker.pkg.dev/YOUR_PROJECT/ai-desk/backend:latest
```

#### 3. Deploy to Cloud Run

```bash
gcloud run deploy ai-desk-backend \
  --image us-central1-docker.pkg.dev/YOUR_PROJECT/ai-desk/backend:latest \
  --platform managed \
  --region us-central1 \
  --memory 1Gi \
  --cpu 1 \
  --set-env-vars DB_HOST=postgres.c.YOUR_PROJECT.internal,DB_USER=postgres,JWT_SECRET=your-secret
```

#### 4. Deploy Frontend to Cloud Storage + CDN

```bash
# Build frontend
cd frontend
npm run build

# Upload to GCS
gsutil -m cp -r dist/* gs://ai-desk-frontend/

# Serve via Cloud CDN
gsutil cors set cors.json gs://ai-desk-frontend

# Update frontend API endpoint in config
```

---

## Monitoring & Logs

### Local Docker Logs

```bash
# Backend logs
docker-compose logs -f go-app

# Database logs
docker-compose logs -f postgres

# All logs
docker-compose logs -f
```

### Production Logs

**AWS EC2:**
```bash
docker-compose logs --tail=100 -f
```

**Google Cloud Run:**
```bash
gcloud run logs read ai-desk-backend --limit=50
```

---

## Troubleshooting

### Backend won't start

```bash
# Check if port 8000 is in use
lsof -i :8000

# Check database connection
docker-compose logs postgres

# Rebuild containers
docker-compose down
docker-compose up -d
```

### Frontend shows "API connection failed"

1. Check backend is running: `curl http://localhost:8000/health`
2. Check frontend .env has correct API URL
3. Check CORS headers in backend

### PostgreSQL connection error

```bash
# Reset database
docker-compose down -v
docker-compose up -d postgres

# Wait 10 seconds for DB to start
sleep 10

# Restart other services
docker-compose up -d
```

### Permission denied errors

```bash
# Fix Docker permissions
sudo chown -R $USER:$USER ~/ai-desk
sudo chmod -R 755 ~/ai-desk
```

---

## Environment Variables Reference

### Backend (.env)

```
DB_HOST=postgres          # Database host
DB_PORT=5432              # Database port
DB_USER=postgres          # Database user
DB_PASSWORD=secure        # Database password
DB_NAME=ai_desk           # Database name
JWT_SECRET=min32chars     # JWT signing secret (minimum 32 chars)
ENVIRONMENT=development   # development or production
PORT=8000                 # Server port
```

### Frontend (.env)

```
VITE_API_URL=http://localhost:8000/api  # Backend API URL
VITE_APP_NAME=AI-DESK                   # App name
```

---

## Backup & Recovery

### Backup Database

```bash
# Create backup
docker-compose exec postgres pg_dump -U postgres ai_desk > backup.sql

# Restore from backup
docker-compose exec -T postgres psql -U postgres ai_desk < backup.sql
```

### Backup Files

```bash
# Backup entire project
tar -czf ai-desk-backup.tar.gz ~/ai-desk/

# Store in S3
aws s3 cp ai-desk-backup.tar.gz s3://your-bucket/backups/
```

---

## Performance Tuning

### Database Optimization

```sql
-- Vacuum & analyze
VACUUM ANALYZE;

-- Check index usage
SELECT * FROM pg_stat_user_indexes;
```

### Redis Monitoring

```bash
docker-compose exec redis redis-cli info memory
docker-compose exec redis redis-cli dbsize
```

### Resource Limits

Edit `docker-compose.yml`:

```yaml
services:
  go-app:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
```

---

## Maintenance

### Update Code

```bash
cd ~/ai-desk
git pull origin main
docker-compose up -d --build
```

### Clean Docker

```bash
# Remove unused images
docker image prune -a

# Remove unused volumes
docker volume prune

# Restart services
docker-compose restart
```

### Health Checks

```bash
# Backend health
curl http://localhost:8000/health

# Frontend accessibility
curl http://localhost:5173

# Database connectivity
docker-compose exec postgres pg_isready -U postgres
```

---

## Support

If deployment fails:
1. Check all prerequisites are installed
2. Review logs: `docker-compose logs -f`
3. Verify .env files have correct values
4. Ensure ports 5432, 6379, 8000, 5173 are not in use
5. Try: `docker-compose down -v && docker-compose up -d`

---

## Quick Reference

| Command | Purpose |
|---------|---------|
| `docker-compose up -d` | Start all services |
| `docker-compose down` | Stop all services |
| `docker-compose logs -f` | View live logs |
| `go run cmd/main.go` | Run backend locally |
| `npm run dev` | Run frontend locally |
| `docker-compose ps` | Check service status |
| `docker-compose exec postgres psql -U postgres` | Access database |

---

**Last Updated:** June 1, 2025  
**Version:** 1.0  
**Status:** Production Ready
