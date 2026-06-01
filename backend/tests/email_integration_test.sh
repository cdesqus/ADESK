#!/bin/bash

# Email Integration Testing Script
# This script provides examples for testing the email integration

set -e

API_URL="http://localhost:8080"
CONTENT_TYPE="Content-Type: application/json"

echo "====== AI-DESK Email Integration Testing ======"
echo ""

# Test 1: Health check
echo "Test 1: Health Check"
curl -X GET "${API_URL}/health" | jq .
echo ""

# Test 2: Send test email via webhook (Mitsubishi)
echo "Test 2: Create Ticket from Email (Mitsubishi)"
RESPONSE=$(curl -s -X POST "${API_URL}/api/email/webhook" \
  -H "${CONTENT_TYPE}" \
  -d '{
    "from": "admin@mitsubishi.com",
    "to": "helpdesk@idesolusi.co.id",
    "subject": "System Integration Issue",
    "body": "We are experiencing an issue with the integration. Please help us resolve this.",
    "text_body": "We are experiencing an issue with the integration. Please help us resolve this.",
    "html_body": "<p>We are experiencing an issue with the integration. Please help us resolve this.</p>",
    "message_id": "<mitsubishi-001@mitsubishi.com>",
    "date": "2024-01-15T10:30:00Z",
    "attachments": []
  }')

echo "$RESPONSE" | jq .
TICKET_ID=$(echo "$RESPONSE" | jq -r '.ticket_id // empty')
echo ""

if [ -n "$TICKET_ID" ]; then
    echo "Created Ticket ID: $TICKET_ID"

    # Test 3: Verify ticket was created
    echo "Test 3: Verify Ticket Creation"
    curl -s -X GET "${API_URL}/api/tickets/${TICKET_ID}" \
      -H "Authorization: Bearer YOUR_JWT_TOKEN" | jq .
    echo ""

    # Test 4: Check email logs
    echo "Test 4: Verify Email Log Entry"
    curl -s -X GET "${API_URL}/api/email/logs?ticket_id=${TICKET_ID}" \
      -H "Authorization: Bearer YOUR_JWT_TOKEN" | jq .
    echo ""
fi

# Test 5: Test with unknown domain
echo "Test 5: Create Ticket from Email (Unknown Domain)"
RESPONSE=$(curl -s -X POST "${API_URL}/api/email/webhook" \
  -H "${CONTENT_TYPE}" \
  -d '{
    "from": "contact@unknowndomain.com",
    "to": "helpdesk@idesolusi.co.id",
    "subject": "General Inquiry",
    "body": "I have a question about your services.",
    "message_id": "<unknown-001@unknowndomain.com>",
    "date": "2024-01-15T11:00:00Z",
    "attachments": []
  }')

echo "$RESPONSE" | jq .
TICKET_ID=$(echo "$RESPONSE" | jq -r '.ticket_id // empty')

if [ -n "$TICKET_ID" ]; then
    echo "Created Ticket ID (Unknown): $TICKET_ID"
fi
echo ""

# Test 6: Test with subdomain
echo "Test 6: Create Ticket from Email (Subdomain)"
RESPONSE=$(curl -s -X POST "${API_URL}/api/email/webhook" \
  -H "${CONTENT_TYPE}" \
  -d '{
    "from": "tech@asia.toyota.co.id",
    "to": "helpdesk@idesolusi.co.id",
    "subject": "Technical Support Needed",
    "body": "Our system is having technical issues.",
    "message_id": "<toyota-asia-001@toyota.co.id>",
    "date": "2024-01-15T11:30:00Z",
    "attachments": []
  }')

echo "$RESPONSE" | jq .
echo ""

echo "====== Manual Testing Steps ======"
echo ""
echo "1. Check Email Logs Table:"
echo "   docker exec -it ai-desk-db psql -U postgres -d ai_desk -c \"SELECT * FROM email_logs ORDER BY created_at DESC LIMIT 10;\""
echo ""

echo "2. Check Tickets Created:"
echo "   docker exec -it ai-desk-db psql -U postgres -d ai_desk -c \"SELECT id, title, customer_id, email_from, status FROM tickets WHERE source = 'EMAIL' ORDER BY created_at DESC LIMIT 10;\""
echo ""

echo "3. Monitor Email Poller (in docker logs):"
echo "   docker logs -f ai-desk --tail 100 | grep EmailPoller"
echo ""

echo "4. Send Real Email to helpdesk@idesolusi.co.id:"
echo "   - From an email with a matching customer domain"
echo "   - Wait up to 5 minutes for auto-processing"
echo "   - Check database for created ticket"
echo ""

echo "5. Verify Auto-Reply Received:"
echo "   - Check your email for reply from IDE SOLUSI INTEGRASI Support"
echo "   - Should include Ticket ID: TK-[ticket_number]"
echo ""

echo "====== Database Queries ======"
echo ""

echo "Query all email logs:"
echo "SELECT id, sender_email, domain_matched, status, created_at FROM email_logs ORDER BY created_at DESC;"
echo ""

echo "Count by status:"
echo "SELECT status, COUNT(*) as count FROM email_logs GROUP BY status;"
echo ""

echo "Find unknown domains:"
echo "SELECT DISTINCT domain_matched, COUNT(*) as count FROM email_logs WHERE status = 'UNKNOWN_DOMAIN' GROUP BY domain_matched;"
echo ""

echo "Check email to ticket mapping:"
echo "SELECT el.sender_email, el.domain_matched, el.ticket_id, t.title, t.customer_id FROM email_logs el LEFT JOIN tickets t ON el.ticket_id = t.id ORDER BY el.created_at DESC LIMIT 10;"
echo ""

echo "====== Testing Complete ======"
