# AI-DESK Backend Deployment Guide

## Local Development Setup

### Prerequisites
- Go 1.21+
- PostgreSQL 16
- Redis 7
- Docker & Docker Compose (for containerized setup)

### Quick Start with Docker Compose

1. Clone/navigate to the backend directory:
```bash
cd backend
```

2. Create `.env` file from `.env.example`:
```bash
cp .env.example .env
```

3. Update `.env` with your configuration (optional for local dev):
```
SERVER_PORT=8080
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=ai_desk
REDIS_HOST=redis
REDIS_PORT=6379
JWT_SECRET=dev-secret-key-change-in-production
ENVIRONMENT=development
```

4. Start all services:
```bash
docker-compose up -d
```

5. Verify services are running:
```bash
docker ps
```

6. Check API health:
```bash
curl http://localhost:8080/health
```

### Local Development Without Docker

1. Install dependencies:
```bash
go mod download
```

2. Set up PostgreSQL and Redis locally

3. Create `.env` file pointing to local services:
```
DB_HOST=localhost
REDIS_HOST=localhost
```

4. Run migrations manually:
```bash
psql -h localhost -U postgres -d ai_desk -f migrations/001_init.sql
```

5. Run the application:
```bash
go run ./cmd/main.go
```

## API Endpoints

### Authentication
- **POST** `/api/auth/login` - Login and get JWT token
  - Body: `{"email": "admin@aidesK.local", "password": "admin123"}`

### Customers (Protected)
- **GET** `/api/customers` - List customers
- **POST** `/api/customers` - Create customer
- **GET** `/api/customers/:id` - Get customer details
- **PUT** `/api/customers/:id` - Update customer
- **DELETE** `/api/customers/:id` - Delete customer

### Engineers (Protected)
- **GET** `/api/engineers` - List engineers
- **POST** `/api/engineers` - Create engineer
- **GET** `/api/engineers/:id` - Get engineer details
- **PUT** `/api/engineers/:id` - Update engineer
- **DELETE** `/api/engineers/:id` - Delete engineer

### Tickets (Protected)
- **GET** `/api/tickets` - List tickets with filters
- **POST** `/api/tickets` - Create ticket
- **GET** `/api/tickets/:id` - Get ticket details
- **PUT** `/api/tickets/:id` - Update ticket
- **DELETE** `/api/tickets/:id` - Delete ticket

### Updates/Comments (Protected)
- **GET** `/api/tickets/:ticket_id/updates` - Get ticket updates
- **POST** `/api/tickets/:ticket_id/updates` - Add update to ticket
- **DELETE** `/api/updates/:id` - Delete update

## Authentication

All protected endpoints require JWT token in Authorization header:
```bash
Authorization: Bearer <token>
```

## Docker Compose Services

- **postgres** - PostgreSQL database (port 5432)
- **redis** - Redis cache (port 6379)
- **app** - Go application (port 8080)

## Logs

View application logs:
```bash
docker-compose logs -f app
```

## Stopping Services

```bash
docker-compose down
```

Remove volumes:
```bash
docker-compose down -v
```

## Production Deployment

1. Update `.env` with production values:
   - Set `ENVIRONMENT=production`
   - Use strong `JWT_SECRET`
   - Update database credentials
   - Use `DB_SSL_MODE=require` if SSL is available

2. Build the Docker image:
```bash
docker build -t ai-desk-backend:latest .
```

3. Push to your container registry (e.g., Docker Hub, ECR)

4. Deploy using Kubernetes, ECS, or other orchestration platform

5. Ensure environment variables are securely managed (use Secrets)

## Database Migrations

Migrations are automatically applied when the app starts (via GORM AutoMigrate).

For manual migration (if needed):
```bash
psql -h <host> -U <user> -d <dbname> -f migrations/001_init.sql
```

## Troubleshooting

### Connection Refused
- Ensure PostgreSQL and Redis are running
- Verify correct host/port in `.env`

### Database Locked
- Check for other connections: `select * from pg_stat_activity;`

### JWT Token Issues
- Ensure JWT_SECRET is set consistently
- Token expires in 24 hours by default

### Port Already in Use
- Change port in `.env` or kill existing process
