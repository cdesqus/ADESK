# Email Integration Setup Guide

## Quick Start

### 1. Add Go Dependencies

Update your `go.mod` with the required email libraries:

```bash
go get github.com/emersion/go-imap@v1.2.1
go get github.com/google/uuid@v1.5.0
go mod tidy
```

Or manually add to `go.mod`:
```
require (
    github.com/emersion/go-imap v1.2.1
    github.com/google/uuid v1.5.0
)
```

### 2. Configure Environment Variables

Copy the example environment file:
```bash
cp .env.example .env
```

Edit `.env` and fill in email settings:

#### For Gmail
```
EMAIL_IMAP_HOST=imap.gmail.com
EMAIL_IMAP_PORT=993
EMAIL_USER=helpdesk@idesolusi.co.id
EMAIL_PASSWORD=<your-gmail-app-password>

SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=helpdesk@idesolusi.co.id
SMTP_PASSWORD=<your-gmail-app-password>
EMAIL_FROM_NAME=IDE SOLUSI INTEGRASI Support

EMAIL_POLLING_INTERVAL=5m
```

#### For Microsoft Outlook
```
EMAIL_IMAP_HOST=outlook.office365.com
EMAIL_IMAP_PORT=993
EMAIL_USER=helpdesk@idesolusi.co.id
EMAIL_PASSWORD=<your-outlook-password>

SMTP_HOST=smtp.office365.com
SMTP_PORT=587
SMTP_USER=helpdesk@idesolusi.co.id
SMTP_PASSWORD=<your-outlook-password>
EMAIL_FROM_NAME=IDE SOLUSI INTEGRASI Support

EMAIL_POLLING_INTERVAL=5m
```

#### For Other Email Providers
Contact your email provider for IMAP/SMTP server details.

### 3. Gmail-Specific Setup

If using Gmail, follow these steps:

1. Enable 2-Factor Authentication on your Google account
2. Generate an App Password:
   - Go to https://myaccount.google.com/apppasswords
   - Select "Mail" and "Windows Computer" (or your device)
   - Google will generate a 16-character password
   - Copy this password to EMAIL_PASSWORD in `.env`

### 4. Set Up Customer Domains

Ensure all customers have domain field populated:

```bash
# Via API
curl -X PUT http://localhost:8080/api/customers/1 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "mitsubishi.com"
  }'

# Or via database
UPDATE customers SET domain = 'mitsubishi.com' WHERE id = 1;
UPDATE customers SET domain = 'toyota.co.id' WHERE id = 2;
UPDATE customers SET domain = 'siemens.com' WHERE id = 3;
```

### 5. Database Migrations

Migrations run automatically on startup. The email_logs table will be created with:
- Indexes for fast queries
- Foreign keys to customers and tickets
- Status tracking (SUCCESS, FAILED, UNKNOWN_DOMAIN)

Manual migration if needed:
```bash
psql -U postgres -d ai_desk -f migrations/002_email_logs.sql
```

### 6. Start Application

```bash
go run ./cmd/main.go
```

You should see in logs:
```
Email poller started with interval 5m0s
[EmailPoller] Starting email polling cycle
```

## Testing the Integration

### Test 1: Send via Webhook

```bash
curl -X POST http://localhost:8080/api/email/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "admin@mitsubishi.com",
    "to": "helpdesk@idesolusi.co.id",
    "subject": "Test Ticket",
    "body": "This is a test email",
    "message_id": "<test@example.com>"
  }'
```

Expected response:
```json
{
  "ticket_id": 1,
  "status": "success",
  "message": "Ticket created from email"
}
```

### Test 2: Send Real Email

1. Send email from customer domain to `helpdesk@idesolusi.co.id`
2. Wait up to 5 minutes (next polling cycle)
3. Check if ticket was auto-created

### Test 3: Check Logs

```bash
# All recent logs
SELECT * FROM email_logs ORDER BY created_at DESC LIMIT 20;

# Successful processes only
SELECT * FROM email_logs WHERE status = 'SUCCESS' ORDER BY created_at DESC;

# Unknown domains
SELECT * FROM email_logs WHERE status = 'UNKNOWN_DOMAIN' ORDER BY created_at DESC;

# Failed processes
SELECT * FROM email_logs WHERE status = 'FAILED' ORDER BY created_at DESC;
```

### Test 4: Verify Ticket Creation

```bash
# Get recent tickets from email
SELECT id, title, customer_id, email_from, status 
FROM tickets 
WHERE source = 'EMAIL' 
ORDER BY created_at DESC 
LIMIT 10;
```

### Test 5: Check Auto-Reply

Check your email inbox for auto-reply from `helpdesk@idesolusi.co.id` with:
- "Re: [original subject]"
- Ticket ID (TK-123)
- Ticket status and expected response time

## Common Issues and Troubleshooting

### Issue: "Failed to connect IMAP"

**Causes:**
- Wrong IMAP server address
- Wrong port (should be 993 for TLS)
- Email/password incorrect
- Firewall blocking connection

**Solutions:**
1. Test IMAP server manually:
   ```bash
   nc -zv imap.gmail.com 993
   ```

2. Verify credentials in `.env`

3. Check firewall settings

4. For Gmail, ensure app password is generated (not regular password)

### Issue: "No unread emails"

This is not an error. The system is working correctly. Send a test email to verify.

### Issue: Ticket created but auto-reply not sent

**Causes:**
- SMTP credentials wrong
- SMTP server unreachable
- Email formatting issues

**Solutions:**
1. Check logs for "Failed to send email"
2. Verify SMTP settings in `.env`
3. Test SMTP connection:
   ```bash
   nc -zv smtp.gmail.com 587
   ```

### Issue: Domain matching not working

**Causes:**
- Customer domain field is empty
- Domain format inconsistency

**Solutions:**
1. Check customer domains are set:
   ```bash
   SELECT id, name, domain FROM customers;
   ```

2. Check email_logs for domain_matched value:
   ```bash
   SELECT sender_email, domain_matched, status FROM email_logs;
   ```

3. Verify domain format (should be like "mitsubishi.com" not "www.mitsubishi.com")

### Issue: Tickets created with "Unknown Customer"

**Solution:**
1. Check which domains are creating unknowns:
   ```bash
   SELECT DISTINCT domain_matched, COUNT(*) 
   FROM email_logs 
   WHERE status = 'UNKNOWN_DOMAIN' 
   GROUP BY domain_matched;
   ```

2. Add these domains to customers or create new customers

### Issue: Email polling not starting

**Causes:**
- EMAIL_USER and EMAIL_PASSWORD not set in `.env`
- Database connection failed

**Solutions:**
1. Ensure `.env` has EMAIL_USER and EMAIL_PASSWORD
2. Verify database is running and migrations completed
3. Check logs for specific error message

## Production Deployment

### Security Checklist

- [ ] Use app-specific passwords (not regular passwords)
- [ ] Store credentials in environment variables only
- [ ] Secure SMTP/IMAP passwords - never log them
- [ ] Add authentication to /api/email/webhook endpoint
- [ ] Use HTTPS for webhook endpoint in production
- [ ] Implement rate limiting on webhook
- [ ] Regular backup of email_logs table

### Performance Optimization

1. Adjust polling interval based on email volume:
   ```bash
   # High volume
   EMAIL_POLLING_INTERVAL=1m
   
   # Medium volume
   EMAIL_POLLING_INTERVAL=5m
   
   # Low volume
   EMAIL_POLLING_INTERVAL=15m
   ```

2. Monitor email_logs growth and archive old records:
   ```bash
   -- Archive logs older than 30 days
   DELETE FROM email_logs 
   WHERE created_at < NOW() - INTERVAL '30 days';
   ```

3. Add more database indexes if needed:
   ```bash
   CREATE INDEX idx_email_logs_status_date ON email_logs(status, created_at);
   CREATE INDEX idx_email_logs_customer_ticket ON email_logs(customer_id, ticket_id);
   ```

### Monitoring and Alerts

Monitor these in production:

1. Email poller connection health
2. Failed email processing rate
3. Email processing latency
4. Auto-reply delivery success rate
5. Unknown domain emails trending up

## Advanced Configuration

### Custom Email Filtering

To add custom email filtering before ticket creation, modify `email_poller.go`:

```go
// After parsing email, before creating ticket
if emailMsg.Subject == "" || len(emailMsg.Body) < 10 {
    log.Printf("[EmailPoller] Skipping invalid email")
    return // Skip processing
}

// Add spam detection, etc.
```

### Custom Domain Matching

To customize domain matching logic, modify `matcher.go`:

```go
// Add your own matching rules
if specialCase := checkSpecialCustomerRules(senderDomain); specialCase != nil {
    return specialCase
}
```

### Custom Auto-Reply Template

To customize auto-reply text, modify `sender.go` `buildAutoReplyBody()` function.

## Support and Debugging

Enable detailed logging:

```bash
# Watch all email poller logs
docker logs -f <container> 2>&1 | grep EmailPoller

# Watch database operations
GORM_LOGGER_LEVEL=debug go run ./cmd/main.go
```

Query for debugging:

```bash
-- Most recent email processing
SELECT * FROM email_logs ORDER BY created_at DESC LIMIT 1;

-- Failed emails with error details
SELECT sender_email, error_message FROM email_logs 
WHERE status = 'FAILED' 
ORDER BY created_at DESC;

-- Email to ticket mapping
SELECT 
  el.sender_email,
  el.domain_matched,
  t.id as ticket_id,
  t.title,
  c.name as customer_name
FROM email_logs el
LEFT JOIN tickets t ON el.ticket_id = t.id
LEFT JOIN customers c ON el.customer_id = c.id
WHERE el.created_at > NOW() - INTERVAL '1 day'
ORDER BY el.created_at DESC;
```

## File Structure

```
backend/
├── internal/
│   ├── email/
│   │   ├── types.go          # Data structures
│   │   ├── imap.go          # IMAP client
│   │   ├── parser.go        # Email parsing
│   │   ├── matcher.go       # Domain matching
│   │   ├── sender.go        # SMTP client
│   │   └── init.go          # Initialization helper
│   ├── handlers/
│   │   └── email.go         # HTTP handler
│   └── jobs/
│       └── email_poller.go  # Background job
├── cmd/
│   └── main.go             # Updated with email setup
├── migrations/
│   └── 002_email_logs.sql  # Database migration
├── config/
│   └── config.go           # Updated with email vars
├── .env.example            # Updated with email settings
└── EMAIL_INTEGRATION.md    # Complete documentation
```

## Next Steps

1. Run tests with `tests/email_integration_test.sh`
2. Monitor first few hours of operation
3. Set up production environment
4. Configure backup/archival strategy
5. Implement custom domain matching if needed
6. Add email filtering rules as needed
