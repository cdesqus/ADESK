# WhatsApp Integration - Phase 3

## Overview
AI-DESK Phase 3 integrates WhatsApp via Waha Plus to enable customers and engineers to manage support tickets through WhatsApp messages.

## Architecture Components

### 1. Waha Plus Service
- Container: devlikeapro/waha-plus:latest
- Port: 3000
- Role: WhatsApp API gateway for session management and message handling
- Communicates via webhooks with the backend

### 2. Go Backend Components

#### internal/whatsapp/client.go
- WahaClient: HTTP client communicating with Waha Plus API
- Key methods:
  * CreateSession(name) - Initialize new WhatsApp session
  * GetSessionQR(sessionName) - Retrieve QR code for scanning
  * SendMessage(sessionName, to, message) - Send text messages
  * GetSessions() - List all active sessions
  * DeleteSession(sessionName) - Disconnect session
  * CheckSessionStatus(sessionName) - Verify connection status

#### internal/whatsapp/parser.go
- ParseMessage(body) - Extract action from message text
- Recognized patterns:
  * "tolong buatin tiket ..." → create_ticket
  * "TK-xxx progress: ..." → update ticket status
  * "TK-xxx close: ..." → close ticket
  * "TK-xxx reopen" → reopen ticket
  * "status TK-xxx" → check ticket status

#### internal/whatsapp/sender.go
- MessageSender: Handles outbound message delivery
- Features:
  * Automatic retry logic (3 attempts with exponential backoff)
  * Message logging for audit trail
  * Support for both direct and group messages

#### internal/whatsapp/actions.go
- ActionHandler: Processes parsed message actions
- Methods:
  * HandleCreateTicket() - Auto-create tickets from customer messages
  * HandleTicketUpdate() - Update ticket status from engineer messages
  * HandleTicketClose() - Close tickets with resolution
  * HandleTicketReopen() - Allow customers to reopen tickets
  * HandleStatusCheck() - Send ticket status to customer

#### internal/handlers/whatsapp.go
- WhatsAppHandler: REST API endpoints for WhatsApp management
- Endpoints:
  * POST /api/whatsapp/sessions - Create new session
  * GET /api/whatsapp/sessions - List sessions
  * GET /api/whatsapp/sessions/:id/qr - Get QR code
  * POST /api/whatsapp/sessions/:id/verify - Check connection
  * DELETE /api/whatsapp/sessions/:id - Disconnect session
  * POST /api/whatsapp/engineers/:id/phone - Link engineer phone
  * GET /api/whatsapp/logs - View message history
  * POST /api/whatsapp/webhook - Incoming message handler (no auth)

### 3. Database Models

#### WhatsAppSession
- Tracks WhatsApp business account sessions
- Fields: id, session_name, phone_number, status, qr_code
- Status: PENDING, CONNECTED, DISCONNECTED

#### EngineerWAPhone
- Links engineers to WhatsApp numbers for notifications
- Fields: id, engineer_id, phone_number, group_id, is_active
- Enables targeting specific engineers or groups

#### WhatsAppLog
- Complete audit trail of all WhatsApp messages
- Fields: id, session_name, from_phone, to_phone, body, direction, status
- Supports inbound/outbound tracking

#### Ticket (updated)
- New fields: whatsapp_from, whatsapp_session_id
- Source: WHATSAPP support type

## Setup Instructions

### 1. Docker Configuration
Update docker-compose.yml includes:
- Waha Plus service on port 3000
- Backend service environment variables for Waha connection
- Network configuration for inter-service communication

### 2. Environment Variables
```
WAHA_API_URL=http://waha:3000
WAHA_API_PORT=3000
WAHA_WEBHOOK_URL=http://app:8080/api/whatsapp/webhook
```

### 3. Database Migrations
Run migration 003_whatsapp.sql which creates:
- whatsapp_sessions table
- engineer_wa_phones table
- whatsapp_logs table
- Indexes for performance

### 4. Starting the Integration

1. Ensure Docker credentials for devlikeapro/waha-plus are configured
2. Start backend: `docker-compose up`
3. Backend auto-migrates database on startup
4. WhatsApp components initialized in cmd/main.go

## Usage Workflow

### Customer Initiation
1. Customer sends: "tolong buatin tiket databasenya error"
2. Parser detects create_ticket action
3. ActionHandler:
   - Creates OPEN ticket in database
   - Assigns to available engineer (round-robin)
   - Sends confirmation: "Terima kasih! Kami buat tiket TK-123..."
   - Notifies engineer via WhatsApp

### Engineer Response
1. Engineer sends: "TK-123 progress: sedang menganalisis masalah"
2. Parser detects update action
3. ActionHandler:
   - Updates ticket to IN_PROGRESS
   - Adds comment to audit trail
   - Sends update to customer: "Update TK-123: sedang menganalisis masalah"

### Ticket Resolution
1. Engineer sends: "TK-123 close: masalah sudah diperbaiki"
2. Parser detects close action
3. ActionHandler:
   - Updates ticket to RESOLVED
   - Sends closure message to customer
   - Requests confirmation: "Reply 'setuju' jika Anda puas"

### Customer Reopen
1. Customer sends: "TK-123 reopen"
2. Parser detects reopen action
3. ActionHandler:
   - Changes ticket to REOPENED
   - Notifies assigned engineer
   - Prevents duplicate ticket creation

## API Examples

### Create WhatsApp Session
```bash
curl -X POST http://localhost:8080/api/whatsapp/sessions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"session_name": "support_team"}'
```

Response:
```json
{
  "id": "uuid",
  "session_name": "support_team",
  "status": "PENDING",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Get QR Code
```bash
curl http://localhost:8080/api/whatsapp/sessions/{id}/qr \
  -H "Authorization: Bearer <token>"
```

Response:
```json
{
  "qr_code": "base64-encoded-qr-image"
}
```

### Add Engineer Phone
```bash
curl -X POST http://localhost:8080/api/whatsapp/engineers/1/phone \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d {
    "phone_number": "+62812345678",
    "group_id": "engineer_group@g.us"
  }
```

### View Message Logs
```bash
curl "http://localhost:8080/api/whatsapp/logs?session_name=support_team&direction=inbound&page=1&limit=50" \
  -H "Authorization: Bearer <token>"
```

## Security Considerations

1. **Webhook Verification**: Webhook endpoint (`/api/whatsapp/webhook`) accepts unauthenticated requests from Waha Plus. Consider adding HMAC verification in production.

2. **Phone Number Matching**: Engineer phone numbers should be validated when linking to prevent unauthorized access.

3. **Message Content**: Sanitize customer messages before storing in database to prevent injection attacks.

4. **Rate Limiting**: Consider adding rate limiting to prevent spam via WhatsApp.

5. **Audit Trail**: All messages logged with timestamps for compliance and debugging.

## Error Handling

- Failed message sends retry 3 times with exponential backoff
- Unrecognized message patterns logged but don't create tickets
- Engineer authorization checks prevent unauthorized ticket modifications
- Phone number lookup validates engineer-customer relationships

## Testing Checklist

1. Create WhatsApp session and scan QR code
2. Send "tolong buatin tiket test message" - verify ticket created
3. Send "TK-123 progress: working on it" - verify status updated
4. Send "TK-123 close: fixed" - verify ticket closed
5. Send "status TK-123" - verify status response
6. Check logs: `/api/whatsapp/logs` should show all interactions
7. Verify audit trail in ticket updates

## Production Deployment

1. Update WAHA_WEBHOOK_URL to actual domain
2. Configure database backups for WhatsApp logs
3. Monitor Waha Plus logs for connection issues
4. Set up alerting for message delivery failures
5. Use HTTPS for webhook endpoint
6. Rotate database credentials regularly
