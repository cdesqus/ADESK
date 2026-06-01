package reports

import (
	"context"
	"fmt"
	"time"

	"ai-desk/internal/email"
	"ai-desk/internal/models"
)

type ReportMailer struct {
	smtpClient *email.SMTPClient
	maxRetries int
}

func NewReportMailer(smtpClient *email.SMTPClient) *ReportMailer {
	return &ReportMailer{
		smtpClient: smtpClient,
		maxRetries: 3,
	}
}

func (rm *ReportMailer) SendReportEmail(ctx context.Context, customerEmail, customerName string,
	csvBytes, pdfBytes []byte, metrics *models.ReportMetrics, monthName string) error {

	if customerEmail == "" {
		return fmt.Errorf("customer email is required")
	}

	subject := fmt.Sprintf("%s Support Report - %s", monthName, customerName)

	body := fmt.Sprintf(`Dear %s,

Attached is your monthly support report for %s.

Key Metrics:
- Total Tickets: %d
- Resolved: %d (%.1f%%)
- Average Resolution Time: %.2f hours
- SLA Compliance: %.1f%%
- Open Tickets: %d
- In Progress: %d

The detailed report includes:
- Ticket breakdown by status, priority, and source
- Engineer performance metrics and statistics
- Complete ticket list with timestamps and resolution times

For questions or additional details, please contact our support team at support@idesolusi.co.id

Best regards,
IDE SOLUSI INTEGRASI Support Team
`, customerName, monthName,
		metrics.TotalTickets,
		metrics.ResolvedTickets,
		calculatePercentage(metrics.ResolvedTickets, metrics.TotalTickets),
		metrics.AverageResolutionTime,
		metrics.SLACompliance,
		metrics.OpenTickets,
		metrics.InProgressTickets,
	)

	var lastErr error
	for attempt := 1; attempt <= rm.maxRetries; attempt++ {
		if err := rm.smtpClient.SendEmailWithAttachments(
			customerEmail,
			subject,
			body,
			[]email.Attachment{
				{
					Filename: fmt.Sprintf("report_%s.xlsx", monthName),
					Data:     csvBytes,
					MimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				},
				{
					Filename: fmt.Sprintf("report_%s.pdf", monthName),
					Data:     pdfBytes,
					MimeType: "application/pdf",
				},
			},
		); err == nil {
			return nil
		} else {
			lastErr = err
			if attempt < rm.maxRetries {
				// Exponential backoff
				backoff := time.Duration(attempt*attempt) * time.Second
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}

	return fmt.Errorf("failed to send report email after %d attempts: %w", rm.maxRetries, lastErr)
}

func calculatePercentage(value, total int) float64 {
	if total == 0 {
		return 0
	}
	return (float64(value) / float64(total)) * 100
}
