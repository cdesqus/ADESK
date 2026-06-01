# Phase 2 Email Integration Implementation Guide

## Overview

This implementation provides complete email integration for AI-DESK, allowing automatic ticket creation from emails sent to `helpdesk@idesolusi.co.id` with domain-based customer auto-categorization.

## Components

### 1. Email Package (`internal/email/`)

#### `types.go`
- Defines core data structures:
  - `EmailMessage`: Parsed email with all header and body information
  - `AttachmentMetadata`: Email attachment metadata (filename, mime type, size)
  - `DomainMatchResult`: Result of domain matching to customer
  - `EmailLog`: Log entry for email processing

#### `imap.go` - IMAP Email Service
- Connects to email server via IMAP (IMAP4 with TLS)
- Fetches unread emails from INBOX
- Marks emails as read after processing
- Handles connection/reconnection logic
- Methods:
  - `Connect()`: Establish IMAP connection
  - `FetchUnreadEmails()`: Get all unread messages
  - `MarkAsRead(uid)`: Mark specific email as read
  - `Close()`: Close connection
  - `Reconnect()`: Reconnect if connection lost

#### `parser.go` - Email Parser
- Parses raw IMAP messages into structured EmailMessage
- Handles multipart MIME emails (plain text + HTML)
- Extracts attachment metadata (doesn't store actual files)
- Strips HTML tags from email content
- Validates email format
- Methods:
  - `ParseEmail(msg)`: Parse IMAP message
  - `ExtractDomain(email)`: Extract domain from email address
  - `stripHTML()`: Remove HTML tags from content

#### `matcher.go` - Domain Matcher
- Matches sender email domain to customer records
- Implements fuzzy matching logic:
  - Exact domain match (case-insensitive)
  - Subdomain matching (e.g., admin@sub.mitsubishi.com matches mitsubishi.com)
  - Base domain matching (different TLDs)
- Returns customer ID if match found, else "Unknown"
- Methods:
  - `MatchDomain(email)`: Find matching customer by domain
  - `fuzzyMatchDomain()`: Flexible domain comparison

#### `sender.go` - SMTP Email Service
- Sends auto-reply emails via SMTP
- Auto-reply template includes:
  - Ticket ID for reference
  - Ticket status and priority
  - Response time expectation
- Methods:
  - `SendAutoReply()`: Send reply to customer

### 2. Handlers (`internal/handlers/email.go`)

- `EmailHandler`: HTTP handler for email webhook and processing
- Methods:
  - `CreateTicketFromEmail()`: Convert email to ticket with domain matching
  - `ProcessEmailWithLogging()`: Create ticket and log process
  - `ProcessEmailWebhook()`: HTTP endpoint for webhook (POST /api/email/webhook)

- Workflow:
  1. Parse email from request
  2. Match sender domain to customer
  3. Create ticket record
  4. Log processing in email_logs table
  5. Send auto-reply email

### 3. Background Job (`internal/jobs/email_poller.go`)

- Runs periodically (default: every 5 minutes)
- Fetches unread emails from IMAP
- For each email:
  - Parse content
  - Match domain to customer
  - Create ticket
  - Send auto-reply
  - Mark as read
- Graceful shutdown support
- Automatic reconnection on connection loss
- Detailed logging of all actions

### 4. Database Models

#### EmailLog Model (`internal/models/models.go`)
```
Fields:
- id: UUID primary key
- email_message_id: Message-ID from email header
- sender_email: From address
- domain_matched: Extracted domain
- customer_id: Matched customer (nullable)
- ticket_id: Created ticket ID (nullable)
- status: SUCCESS, FAILED, UNKNOWN_DOMAIN
- error_message: Error details if failed
- created_at, updated_at: Timestamps
```

#### Updated Ticket Model
- Added `email_message_id` field for tracking email source

### 5. Configuration

Environment variables (in `.env`):
```
# Email IMAP
EMAIL_IMAP_HOST=imap.gmail.com
EMAIL_IMAP_PORT=993
EMAIL_USER=helpdesk@idesolusi.co.id
EMAIL_PASSWORD=your-app-password

# Email SMTP
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=helpdesk@idesolusi.co.id
SMTP_PASSWORD=your-app-password
EMAIL_FROM_NAME=IDE SOLUSI INTEGRASI Support

# Email Polling
EMAIL_POLLING_INTERVAL=5m
```

## API Endpoints

### Email Webhook (Internal)
```
POST /api/email/webhook

Request Body:
{
  "from": "admin@mitsubishi.com",
  "to": "helpdesk@idesolusi.co.id",
  "subject": "System Error",
  "body": "We are experiencing an issue...",
  "text_body": "...",
  "html_body": "...",
  "message_id": "<unique-id@example.com>",
  "date": "2024-01-15T10:30:00Z",
  "attachments": [
    {
      "filename": "error.log",
      "mime_type": "text/plain",
      "size": 1024
    }
  ]
}

Response:
{
  "ticket_id": 123,
  "status": "success",
  "message": "Ticket created from email"
}
```

## Processing Flow

### Automatic (Background Job)
1. Every 5 minutes, email poller connects to IMAP
2. Fetches all unread emails
3. For each email:
   - Parse headers and body
   - Extract sender domain
   - Query customers for matching domain
   - Create ticket with matched customer_id
   - Send auto-reply email
   - Log processing result
   - Mark email as read

### Manual (Webhook)
1. External system sends POST to /api/email/webhook
2. Handler processes email
3. Creates ticket with auto-categorization
4. Sends auto-reply
5. Returns ticket ID

## Domain Matching Logic

### Example Scenarios

1. **Exact Match**
   - Email from: admin@mitsubishi.com
   - Customer domain: mitsubishi.com
   - Result: MATCH → Use Mitsubishi customer

2. **Subdomain Match**
   - Email from: support@asia.toyota.co.id
   - Customer domain: toyota.co.id
   - Result: MATCH → Use Toyota customer

3. **Different TLD**
   - Email from: info@siemens.de
   - Customer domain: siemens.com
   - Result: MATCH → Use Siemens customer (base domain match)

4. **Unknown Domain**
   - Email from: unknown@example.com
   - No matching customer found
   - Result: Create ticket under "Unknown Customer"

## Error Handling

- **IMAP Connection Failure**: Logs error, retries on next cycle
- **Email Parse Error**: Logs and skips email, marks as read
- **Domain Match Failure**: Creates ticket with "Unknown Customer"
- **Ticket Creation Failure**: Logs error with details
- **Auto-reply Failure**: Logs warning, doesn't fail ticket creation

## Logging

All email processing is logged to `email_logs` table:
- SUCCESS: Ticket created successfully
- FAILED: Ticket creation failed
- UNKNOWN_DOMAIN: No matching customer found

Query logs:
```sql
SELECT * FROM email_logs 
WHERE created_at > NOW() - INTERVAL '1 day'
ORDER BY created_at DESC;

SELECT status, COUNT(*) FROM email_logs 
GROUP BY status;

SELECT domain_matched, COUNT(*) FROM email_logs 
WHERE status = 'UNKNOWN_DOMAIN'
GROUP BY domain_matched;
```

## Setup Instructions

### 1. Dependencies

Add to go.mod:
```
require (
  github.com/emersion/go-imap v1.2.1
  github.com/google/uuid v1.5.0
)
```

Run:
```bash
go mod tidy
```

### 2. Environment Configuration

Copy and fill `.env`:
```bash
cp .env.example .env
```

For Gmail:
- Create app password: https://myaccount.google.com/apppasswords
- Use app password in EMAIL_PASSWORD and SMTP_PASSWORD

### 3. Database Migration

Migrations run automatically on startup via GORM AutoMigrate:
- Creates email_logs table
- Adds email_message_id to tickets
- Creates indexes

Manual SQL migration also available in `migrations/002_email_logs.sql`

### 4. Start Application

```bash
go run ./cmd/main.go
```

Email poller starts automatically if EMAIL_USER and EMAIL_PASSWORD are configured.

## Testing

### Send Test Email
1. Send email to helpdesk@idesolusi.co.id from customer domain
2. Wait up to 5 minutes (or next polling cycle)
3. Check if ticket created with correct customer

### Monitor Logs
```bash
tail -f app.log | grep EmailPoller
```

### Query Email Processing Status
```sql
SELECT * FROM email_logs 
WHERE sender_email = 'admin@mitsubishi.com'
ORDER BY created_at DESC
LIMIT 10;
```

### Manual Webhook Test
```bash
curl -X POST http://localhost:8080/api/email/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "test@example.com",
    "subject": "Test Issue",
    "body": "This is a test email",
    "message_id": "<test@example.com>"
  }'
```

## Production Considerations

1. **Security**
   - Webhook endpoint should be secured (add API key or JWT)
   - SMTP/IMAP credentials in environment only, never hardcoded
   - Consider rate limiting on webhook

2. **Reliability**
   - Email poller runs as background goroutine
   - Automatic reconnection on connection loss
   - Graceful shutdown waits for current processing

3. **Performance**
   - Batch email processing possible in future
   - Database indexes on email_logs for fast queries
   - Consider limiting IMAP batch size

4. **Monitoring**
   - Query email_logs for failure analysis
   - Monitor log output for IMAP connection issues
   - Track unknown domain emails for new customers

## Troubleshooting

### Emails Not Being Processed
1. Check EMAIL_USER and EMAIL_PASSWORD are correct
2. Verify IMAP server connectivity: `telnet imap.gmail.com 993`
3. Check logs for "Failed to connect IMAP"
4. Ensure database email_logs table exists

### Domain Matching Not Working
1. Verify customer.domain values are set
2. Check email_logs.domain_matched vs customer.domain
3. Test domain matching logic manually

### Auto-reply Not Sent
1. Verify SMTP credentials are correct
2. Check SMTP server connectivity
3. Look for "Failed to send email" in logs

### Tickets Not Created
1. Check database connectivity
2. Verify customer exists for matched domain
3. Check error_message in email_logs

## Future Enhancements

- [ ] Support for email attachments (store in S3/local)
- [ ] Advanced spam/phishing detection
- [ ] Email threading (group related emails)
- [ ] Support for multiple helpdesk emails
- [ ] Email signature/quote stripping
- [ ] Encryption for sensitive data
- [ ] Web UI for email logs
- [ ] Webhook retries with backoff
- [ ] Email filtering rules
- [ ] Support for multiple email accounts per customer
