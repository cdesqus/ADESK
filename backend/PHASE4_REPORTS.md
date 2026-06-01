# Phase 4: Monthly Reports System

## Overview
Phase 4 implements a production-ready monthly reporting system for AI-DESK that automatically generates, archives, and emails monthly support reports to customers on the 1st of each month at 8:00 AM.

## Components

### 1. Data Models (`internal/models/models.go`)
- **MonthlyReport**: Stores generated reports with customer reference, month/year, CSV/PDF data, sent timestamps, and recipient emails
- **ReportMetrics**: Aggregates all metrics for a month (ticket counts, resolution times, SLA compliance)
- **EngineerStat**: Per-engineer performance (tickets handled, avg time, resolution rate)
- **TicketSummary**: Ticket details for report display
- **ReportData**: Complete report data structure for rendering

### 2. Report Generation (`internal/reports/generator.go`)
- **ReportGenerator**: Queries all tickets for a customer in a given month/year
- Calculates:
  - Total, resolved, open, in-progress ticket counts
  - Average resolution time (hours)
  - Ticket breakdown by status, priority, source
  - Engineer performance metrics
  - SLA compliance percentage (configurable SLA window)
- Handles edge cases: no tickets, partial months, missing engineer data

### 3. CSV Export (`internal/reports/csv.go`)
- Generates Excel workbooks (XLSX format) using excelize
- Three sheets:
  1. **Summary**: Key metrics, breakdowns by status/priority/source
  2. **Engineers**: Engineer performance table
  3. **Tickets**: Full ticket details with all fields
- Professional formatting with column widths and proper data types

### 4. PDF Export (`internal/reports/pdf.go`)
- Uses fpdf library for PDF generation
- Four pages:
  1. **Cover Page**: Logo, title, customer name, month, generation date
  2. **Executive Summary**: Key metrics in boxes, status/priority breakdowns
  3. **Performance**: Engineer stats, source breakdown
  4. **Ticket Details**: Multi-page ticket table with pagination
- Professional styling:
  - Blue headers (#2E75B6)
  - Alternating row colors
  - Proper margins, fonts, page numbers
  - Sortable tickets by date

### 5. Email Delivery (`internal/reports/mailer.go`)
- **ReportMailer**: Sends reports via SMTP with attachments
- Email body includes key metrics summary
- Attaches both CSV and PDF files
- Retry logic: 3 attempts with exponential backoff
- Proper error handling and logging
- Context-aware timeout support

### 6. Email Integration (`internal/email/sender.go`)
- **SendEmailWithAttachments**: New method for multipart emails with attachments
- Handles base64 encoding of binary files
- Proper MIME headers for attachments
- TLS/SMTP authentication

### 7. Report Repository (`internal/reports/repository.go`)
- **ReportRepository**: Database operations
- CRUD operations for MonthlyReport
- UPSERT support (handles duplicate month/year)
- List with pagination, filtering by customer
- Mark as sent tracking

### 8. Report Scheduler (`internal/jobs/report_scheduler.go`)
- **ReportScheduler**: Runs report generation monthly
- Executes on 1st of month at 8:00 AM (configurable)
- Iterates all active customers
- Generates for previous month
- Handles failures gracefully (continues on customer failure)
- Logs success/failure counts
- Context-aware with graceful shutdown

### 9. API Endpoints (`internal/handlers/reports.go`)
- **POST /api/reports/generate**: On-demand report generation
  - Input: customerID, month, year
  - Returns: ReportData (JSON)
  
- **GET /api/reports**: List all reports
  - Pagination: limit, offset
  - Filter: customer_id
  - Returns: Report metadata with sent status
  
- **GET /api/reports/:id**: Get single report details
  
- **GET /api/reports/:id/download**: Download CSV or PDF
  - Format: csv or pdf parameter
  - Streams binary file with proper headers
  
- **POST /api/reports/:id/resend**: Resend report email
  - Input: email address
  - Regenerates email and marks as sent
  
- **DELETE /api/reports/:id**: Archive report (soft delete)

### 10. Database Migration (`migrations/004_reports.sql`)
- Creates monthly_reports table with:
  - UUID primary key
  - Customer foreign key with cascade delete
  - Month/year with unique constraint per customer
  - CSV and PDF binary data columns
  - Timestamps: generated_at, sent_at, created_at, updated_at
  - JSONB array for sent_to_emails
- Optimized indexes on customer_id, month/year, timestamps

### 11. Configuration (`config/config.go`)
- New config fields:
  - REPORT_GENERATION_TIME: Time to run scheduler (e.g., "08:00")
  - REPORT_MONTH_DAY: Day of month to generate (default: 1)
  - SLA_HOURS: Hours for SLA compliance calculation (default: 24)
  - REPORT_RETENTION_DAYS: Archive retention period (default: 365)

### 12. Startup Integration (`cmd/main.go`)
- Initializes ReportGenerator, ReportMailer, ReportRepository
- Creates and starts ReportScheduler on app startup
- Integrates graceful shutdown for scheduler
- Registers all report API endpoints

## Environment Variables

```
# Report Configuration
REPORT_GENERATION_TIME=08:00        # 8 AM (24-hour format)
REPORT_MONTH_DAY=1                  # 1st of month
SLA_HOURS=24                        # 24-hour SLA window
REPORT_RETENTION_DAYS=365           # 1 year retention

# SMTP for report delivery
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
EMAIL_FROM_NAME=IDE SOLUSI INTEGRASI Support
```

## Dependencies

Required Go packages:
```
github.com/go-pdf/fpdf v1.4.2              # PDF generation
github.com/xuri/excelize/v2 v2.8.0         # Excel generation
github.com/google/uuid v1.5.0               # UUID generation
gorm.io/gorm                                # ORM (existing)
gorm.io/clause                              # UPSERT support
```

## Features

1. **Automatic Scheduling**: Runs monthly without manual intervention
2. **Comprehensive Metrics**: Ticket counts, resolution times, SLA compliance, engineer stats
3. **Multiple Formats**: CSV (Excel) and PDF with professional styling
4. **Email Delivery**: Automatic sending on 1st of month with retry logic
5. **On-Demand Generation**: Manual API for immediate report generation
6. **Archive Storage**: All reports stored in database with audit trail
7. **Resend Capability**: Resend any archived report via email
8. **Pagination**: Efficient listing with limit/offset
9. **Error Handling**: Graceful failures, retries, comprehensive logging
10. **Production-Ready**: No hardcoded values, configurable via env

## Usage Examples

### Generate Report On-Demand
```bash
curl -X POST http://localhost:8080/api/reports/generate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": 1,
    "month": 6,
    "year": 2026
  }'
```

### List Reports
```bash
curl http://localhost:8080/api/reports?customer_id=1&limit=10&offset=0 \
  -H "Authorization: Bearer <token>"
```

### Download PDF Report
```bash
curl http://localhost:8080/api/reports/abc-123-def/download?format=pdf \
  -H "Authorization: Bearer <token>" \
  -o report.pdf
```

### Resend Report Email
```bash
curl -X POST http://localhost:8080/api/reports/abc-123-def/resend \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "customer@example.com"
  }'
```

## Testing Checklist

- [ ] Generate report via API (POST /api/reports/generate)
- [ ] Verify metrics calculation: ticket counts, averages, percentages
- [ ] Download CSV file and verify structure
- [ ] Download PDF file and verify formatting
- [ ] Test email delivery with test SMTP service
- [ ] Verify report saved in database
- [ ] Test scheduled job (set cron to 1 min from now, verify runs)
- [ ] Test edge cases: no tickets, single ticket, many engineers
- [ ] Test resend email functionality
- [ ] Test pagination (list reports with limit/offset)
- [ ] Verify SLA compliance calculation
- [ ] Test graceful shutdown

## Database Schema

```sql
CREATE TABLE monthly_reports (
    id VARCHAR(36) PRIMARY KEY,
    customer_id INTEGER NOT NULL REFERENCES customers(id),
    month INTEGER NOT NULL,
    year INTEGER NOT NULL,
    csv_data BYTEA,
    pdf_data BYTEA,
    generated_at TIMESTAMP,
    sent_at TIMESTAMP NULL,
    sent_to_emails JSONB DEFAULT '[]',
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    UNIQUE(customer_id, month, year)
);
```

## Performance Considerations

1. **Indexed Queries**: Efficient retrieval by customer_id, month/year, timestamps
2. **Pagination**: All list operations support limit/offset
3. **Streaming**: Large PDF/CSV files streamed to HTTP response
4. **Context Timeouts**: 5-min timeout for report generation, 30-sec for DB queries
5. **Batch Processing**: Monthly scheduler processes all customers sequentially

## Error Handling

- Missing customer: HTTP 404
- Invalid month: HTTP 400 with clear error message
- SMTP failure: 3 retry attempts with exponential backoff
- Database errors: Wrapped with context, proper HTTP status codes
- Scheduler failures: Logged but continue processing other customers

## Logging

All operations logged to stdout:
- Report generation start/completion
- Metric calculations
- Email sending (success/failure/retry)
- Scheduler ticks and customer processing
- Database operations (via gorm logger)

## Future Enhancements

1. Bulk email delivery with queue
2. Report customization per customer (branding, metrics selection)
3. Scheduled delivery (allow custom month/time per customer)
4. Multi-format support (HTML email inline)
5. Report templates for different customer types
6. Dashboard for report delivery status
7. Webhook notifications on report generation
8. Report comparison (month-over-month metrics)
