# AI-DESK Development Deployment Checklist

**Quick reference for deploying to dev server.**

---

## Before Deployment (Local Machine)

- [ ] Read `DEV_DEPLOYMENT.md` completely
- [ ] Have SSH access to dev server ready
- [ ] Have GitHub/GitLab repo setup (or use SCP)
- [ ] Verify all code is committed/pushed

```bash
cd D:\Demo\AI-DESK\backend
git status  # Should show clean working directory

cd D:\Demo\AI-DESK\frontend
git status  # Should show clean working directory
```

---

## On Development Server

### Phase 1: Infrastructure Setup (~10 minutes)

- [ ] SSH into dev server
- [ ] Create `/opt/ai-desk` directory
- [ ] Install Docker & Docker Compose
- [ ] Verify Docker is working: `docker --version`

```bash
# Copy-paste this entire block:
ssh ubuntu@dev-server-ip

sudo mkdir -p /opt/ai-desk
sudo chown $USER:$USER /opt/ai-desk
cd /opt/ai-desk

curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
exit

ssh ubuntu@dev-server-ip
docker --version
```

### Phase 2: Code Deployment (~5 minutes)

- [ ] Clone/upload code from Git or SCP
- [ ] Verify files are in place

```bash
cd /opt/ai-desk

# Option A: From Git
git clone https://github.com/your-org/ai-desk-backend.git backend
git clone https://github.com/your-org/ai-desk-frontend.git frontend

# Option B: From SCP (if no Git)
# Upload and extract tar.gz files
```

### Phase 3: Configuration (~5 minutes)

- [ ] Create `.env` in backend folder
- [ ] Update email credentials (if testing email)
- [ ] Update WhatsApp Waha Plus URL
- [ ] Create `.env` in frontend folder

```bash
cd /opt/ai-desk/backend

# Copy template and edit
cat > .env << 'EOF'
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=dev_password_123
DB_NAME=ai_desk
ENVIRONMENT=development
JWT_SECRET=dev_secret_key_minimum_32_characters_long!
EMAIL_IMAP_HOST=imap.gmail.com
EMAIL_IMAP_PORT=993
EMAIL_USER=your-email@gmail.com
EMAIL_PASSWORD=your-app-password
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
WAHA_API_URL=http://waha:3000
WAHA_WEBHOOK_URL=http://backend:8000/api/whatsapp/webhook
PORT=8000
EOF

cd /opt/ai-desk/frontend
cat > .env << 'EOF'
VITE_API_URL=http://dev-server-ip:8000/api
VITE_APP_NAME=AI-DESK DEV
EOF
```

### Phase 4: Start Backend Services (~2 minutes)

- [ ] Pull Waha Plus image
- [ ] Start all Docker services
- [ ] Wait for startup (60 seconds)
- [ ] Verify services are running

```bash
cd /opt/ai-desk/backend

# Login to Docker Hub (one-time)
docker login -u devlikeapro -p dckr_pat_7ea_I2rTo7h-QsbJBM1NAl5kMp0

# Pull Waha Plus
docker pull devlikeapro/waha-plus:latest

# Logout (security)
docker logout

# Start services
docker-compose up -d

# Wait
sleep 60

# Check status
docker-compose ps

# Expected: postgres, redis, waha, go-app all "Up"
```

### Phase 5: Start Frontend Services (~3 minutes)

- [ ] Install dependencies (first time, ~3 min)
- [ ] Build frontend
- [ ] Start server

```bash
cd /opt/ai-desk/frontend

# Install (first time only)
npm install

# Build
npm run build

# Start (choose one):

# Option A: Production mode
npm install -g serve
serve -s dist -l 5173

# Option B: Development mode (with hot reload)
npm run dev
```

### Phase 6: Verify Everything (~5 minutes)

- [ ] Backend health check
- [ ] Frontend loads
- [ ] Database tables created
- [ ] Can login with test credentials

```bash
# Health checks (in new terminal/SSH session)

# Backend
curl http://dev-server-ip:8000/health
# Expected: {"status":"ok"}

# Frontend
curl http://dev-server-ip:5173
# Expected: HTML response

# Check database
docker-compose exec postgres psql -U postgres -d ai_desk -c "\dt"
# Expected: List of tables (customers, engineers, tickets, etc)
```

**Login to Dashboard:**
- URL: `http://dev-server-ip:5173`
- Email: `admin@idesolusi.co.id`
- Password: `Admin123!`

---

## After Deployment

### Setup (First Time)

- [ ] Go to Settings → Email Settings
  - [ ] Configure email credentials
  - [ ] Test domain matching
  - [ ] Verify email polling is running

- [ ] Go to Settings → WhatsApp Settings
  - [ ] Add WhatsApp session
  - [ ] Scan QR code
  - [ ] Assign engineer phone numbers

### Testing

- [ ] Send test email to helpdesk@ → verify ticket created
- [ ] Send WhatsApp message to trigger ticket → verify auto-create
- [ ] Update ticket via engineer WhatsApp → verify status change
- [ ] Generate monthly report → verify metrics
- [ ] Download CSV/PDF → verify file downloads

### Share with Team

```
Development Server Access:
URL: http://dev-server-ip:5173
Email: admin@idesolusi.co.id
Password: Admin123!

Backend API: http://dev-server-ip:8000/api
Docs: DEV_DEPLOYMENT.md

Ready to test:
- Email integration (helpdesk@)
- WhatsApp integration (Waha Plus)
- Ticket management
- Monthly reports
```

---

## Troubleshooting Quick Guide

| Issue | Solution |
|-------|----------|
| Services won't start | Run `docker-compose logs -f` to see errors |
| Database connection error | Wait 60 seconds, check `docker-compose ps` |
| Port already in use | Kill process: `lsof -i :PORT` then `kill -9 PID` |
| Email not polling | Check `.env` email credentials, verify SMTP app password |
| WhatsApp not working | Check Waha logs: `docker-compose logs waha` |
| Frontend won't load | Check VITE_API_URL in `.env` matches backend IP |
| Can't login | Verify database tables created: `docker-compose exec postgres psql...` |

---

## Common Commands

```bash
# View logs
docker-compose logs -f go-app
docker-compose logs -f waha
docker-compose logs -f postgres

# Stop/start services
docker-compose stop
docker-compose start
docker-compose restart go-app

# Reset everything (WARNING: deletes data)
docker-compose down -v
docker-compose up -d

# Access database
docker-compose exec postgres psql -U postgres -d ai_desk

# Check resource usage
docker stats
```

---

## Timeline

| Step | Time | Task |
|------|------|------|
| 1 | 10 min | Docker setup |
| 2 | 5 min | Upload code |
| 3 | 5 min | Configure .env |
| 4 | 2 min | Start backend |
| 5 | 3 min | Start frontend |
| 6 | 5 min | Verify + test |
| **Total** | **~30 min** | **Fully deployed** |

---

## Success Criteria

✅ All Docker services running  
✅ Database initialized with tables  
✅ Can login to dashboard  
✅ Backend API responding  
✅ Frontend loads without errors  
✅ Email polling started  
✅ WhatsApp session can be added  

---

## Next Phases

Once dev deployment is stable:
1. **Gather feedback** from team
2. **Make adjustments** based on testing
3. **Plan production** deployment
4. **Setup CI/CD** (GitHub Actions, etc)
5. **Configure monitoring** (logs, metrics, alerting)

---

**Status:** Ready for deployment  
**Date:** June 1, 2025  
**Contact:** Your team lead for assistance
