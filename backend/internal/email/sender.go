package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/smtp"
	"regexp"
	"strings"
	"time"

	"ai-desk/internal/ai"
	"ai-desk/internal/models"
	"github.com/jordan-wright/email"
)

type Attachment struct {
	Filename string
	Data     []byte
	MimeType string
}

type SMTPClient struct {
	host      string
	port      int
	user      string
	password  string
	fromEmail string
	fromName  string
}

// NewSMTPClient creates a new SMTP client
func NewSMTPClient(host string, port int, user, password, fromEmail, fromName string) *SMTPClient {
	return &SMTPClient{
		host:      host,
		port:      port,
		user:      user,
		password:  password,
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

// SendAutoReplyWithClassification sends an auto-reply email using AI classification
// Returns the classification result so the caller can update the ticket
func (sc *SMTPClient) SendAutoReplyWithClassification(toEmail, toName, subject string, ticketNum string, ticketID uint, customer *models.Customer, emailBody, openAIKey, frontendURL string) (*ai.AIClassification, error) {
	if toEmail == "" {
		return nil, fmt.Errorf("recipient email is required")
	}

	var classification *ai.AIClassification

	// Default fallback body
	body := buildAutoReplyBody(ticketNum, customer, toName, sc.fromName)

	// Try AI classification + reply
	if openAIKey != "" {
		customerName := "Valued Customer"
		if toName != "" {
			customerName = toName
		} else if customer != nil && customer.Name != "" && customer.Name != "Unknown Customer" {
			customerName = customer.Name
		}

		aiClient := ai.NewOpenAIClient(openAIKey)
		var err error
		classification, err = aiClient.ClassifyAndReply(emailBody, customerName, sc.fromName, ticketNum)
		if err == nil && classification != nil && classification.Reply != "" {
			body = classification.Reply
		} else {
			log.Printf("AI classification failed, falling back to default template: %v", err)
		}
	}

	// Build reply subject
	var replySubject string
	if strings.HasPrefix(strings.ToLower(subject), "re:") {
		replySubject = subject
	} else {
		replySubject = "Re: " + subject
	}

	// Ensure subject has ticket format
	ticketTag := fmt.Sprintf("[%s]", ticketNum)
	if !strings.Contains(replySubject, ticketTag) {
		replySubject = strings.Replace(replySubject, "Re: ", "Re: "+ticketTag+" ", 1)
	}

	// Build ticket URL
	ticketURL := ""
	if frontendURL != "" {
		ticketURL = fmt.Sprintf("%s/tickets/%d", strings.TrimRight(frontendURL, "/"), ticketID)
	}

	// Create email message with HTML
	from := fmt.Sprintf("%s <%s>", sc.fromName, sc.fromEmail)

	htmlBody := buildAutoReplyHTML(body, ticketNum, ticketURL, sc.fromName)

	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n",
		from,
		toEmail,
		replySubject,
	)

	fullMessage := headers + htmlBody

	// Send email
	addr := fmt.Sprintf("%s:%d", sc.host, sc.port)
	auth := smtp.PlainAuth("", sc.user, sc.password, sc.host)
	err := smtp.SendMail(addr, auth, sc.fromEmail, []string{toEmail}, []byte(fullMessage))
	if err != nil {
		log.Printf("Failed to send email to %s: %v", toEmail, err)
		return classification, fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Auto-reply sent successfully to %s for ticket %s", toEmail, ticketNum)
	return classification, nil
}

// SendAutoReply is backward-compatible wrapper (without classification return)
func (sc *SMTPClient) SendAutoReply(toEmail, toName, subject string, ticketNum string, ticketID uint, customer *models.Customer, emailBody, openAIKey string) error {
	_, err := sc.SendAutoReplyWithClassification(toEmail, toName, subject, ticketNum, ticketID, customer, emailBody, openAIKey, "")
	return err
}

// SendHTMLEmail sends an HTML email
func (sc *SMTPClient) SendHTMLEmail(toEmail, subject, htmlBody string) error {
	if toEmail == "" {
		return fmt.Errorf("recipient email is required")
	}

	from := fmt.Sprintf("%s <%s>", sc.fromName, sc.fromEmail)

	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n",
		from,
		toEmail,
		subject,
	)

	fullMessage := headers + htmlBody

	addr := fmt.Sprintf("%s:%d", sc.host, sc.port)
	auth := smtp.PlainAuth("", sc.user, sc.password, sc.host)

	err := smtp.SendMail(addr, auth, sc.fromEmail, []string{toEmail}, []byte(fullMessage))
	if err != nil {
		log.Printf("Failed to send HTML email to %s: %v", toEmail, err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("HTML email sent successfully to %s", toEmail)
	return nil
}

// SendEmailWithAttachments sends an email with attachments
func (sc *SMTPClient) SendEmailWithAttachments(toEmail, subject, body string, attachments []Attachment) error {
	if toEmail == "" {
		return fmt.Errorf("recipient email is required")
	}

	from := fmt.Sprintf("%s <%s>", sc.fromName, sc.fromEmail)
	boundary := "boundary123456789"

	// Build message with attachments
	var message bytes.Buffer

	// Write headers
	message.WriteString(fmt.Sprintf("From: %s\r\n", from))
	message.WriteString(fmt.Sprintf("To: %s\r\n", toEmail))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", boundary))

	// Write text part
	message.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	message.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	message.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	message.WriteString(body)
	message.WriteString("\r\n")

	// Write attachments
	for _, att := range attachments {
		message.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		message.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", att.MimeType, att.Filename))
		message.WriteString("Content-Transfer-Encoding: base64\r\n")
		message.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", att.Filename))

		encoded := base64.StdEncoding.EncodeToString(att.Data)
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			message.WriteString(encoded[i:end])
			message.WriteString("\r\n")
		}
		message.WriteString("\r\n")
	}

	// Write closing boundary
	message.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	// Send email
	addr := fmt.Sprintf("%s:%d", sc.host, sc.port)
	auth := smtp.PlainAuth("", sc.user, sc.password, sc.host)

	err := smtp.SendMail(addr, auth, sc.fromEmail, []string{toEmail}, message.Bytes())
	if err != nil {
		log.Printf("Failed to send email with attachments to %s: %v", toEmail, err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Email with attachments sent successfully to %s", toEmail)
	return nil
}

// buildAutoReplyHTML wraps the AI reply text in a clean HTML email template
func buildAutoReplyHTML(replyBody string, ticketNum string, ticketURL string, companyName string) string {
	// Convert newlines to <br> for HTML
	htmlReply := strings.ReplaceAll(replyBody, "\n", "<br>")

	trackingButton := ""
	if ticketURL != "" {
		trackingButton = fmt.Sprintf(`
		<tr>
			<td style="padding: 24px 30px 10px;">
				<a href="%s" style="display: inline-block; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: #ffffff; text-decoration: none; padding: 12px 28px; border-radius: 8px; font-weight: 600; font-size: 14px;">
					📋 Lihat Tiket %s
				</a>
			</td>
		</tr>`, ticketURL, ticketNum)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin: 0; padding: 0; background-color: #f4f7fa; font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color: #f4f7fa; padding: 32px 0;">
<tr><td align="center">
<table width="600" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 12px; box-shadow: 0 2px 12px rgba(0,0,0,0.08); overflow: hidden;">
	<!-- Header -->
	<tr>
		<td style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 28px 30px; text-align: center;">
			<h1 style="color: #ffffff; margin: 0; font-size: 22px; font-weight: 700;">%s</h1>
			<p style="color: rgba(255,255,255,0.85); margin: 6px 0 0; font-size: 13px;">Helpdesk Support System</p>
		</td>
	</tr>
	<!-- Ticket Badge -->
	<tr>
		<td style="padding: 20px 30px 0;">
			<span style="display: inline-block; background: #eef2ff; color: #4338ca; padding: 6px 16px; border-radius: 20px; font-size: 13px; font-weight: 600;">
				🎫 Ticket ID: %s
			</span>
		</td>
	</tr>
	<!-- Body -->
	<tr>
		<td style="padding: 20px 30px; color: #374151; font-size: 15px; line-height: 1.7;">
			%s
		</td>
	</tr>
	<!-- Button -->
	%s
	<!-- Footer -->
	<tr>
		<td style="padding: 24px 30px; border-top: 1px solid #e5e7eb; margin-top: 16px;">
			<p style="color: #9ca3af; font-size: 12px; margin: 0; line-height: 1.6;">
				Email ini dikirim oleh %s Helpdesk System.<br>
				Anda menerima email ini karena ada tiket support yang terkait dengan alamat email Anda.
			</p>
		</td>
	</tr>
</table>
</td></tr>
</table>
</body>
</html>`, companyName, ticketNum, htmlReply, trackingButton, companyName)
}

// BuildEngineerNotificationHTML builds a professional HTML email for engineer notification
func BuildEngineerNotificationHTML(engineerName string, ticketID uint, title, priority, source, senderEmail, createdAt, description, ticketURL, companyName, category string) string {
	// Priority color mapping
	priorityColors := map[string]string{
		"LOW":    "#22c55e",
		"MEDIUM": "#f59e0b",
		"HIGH":   "#ef4444",
		"URGENT": "#dc2626",
	}
	priorityColor := priorityColors[priority]
	if priorityColor == "" {
		priorityColor = "#6b7280"
	}

	// Category emoji mapping
	categoryEmoji := map[string]string{
		"PROBLEM":  "🔴",
		"REQUEST":  "🔵",
		"INQUIRY":  "🟡",
		"FEEDBACK": "🟢",
	}
	catEmoji := categoryEmoji[category]
	if catEmoji == "" {
		catEmoji = "📋"
	}

	// Truncate description
	descPreview := description
	if len(descPreview) > 500 {
		descPreview = descPreview[:500] + "..."
	}
	descHTML := strings.ReplaceAll(descPreview, "\n", "<br>")

	viewButton := ""
	if ticketURL != "" {
		viewButton = fmt.Sprintf(`
		<tr>
			<td style="padding: 20px 30px 10px;">
				<a href="%s" style="display: inline-block; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: #ffffff; text-decoration: none; padding: 14px 32px; border-radius: 8px; font-weight: 600; font-size: 14px;">
					🔗 Buka Tiket di Dashboard
				</a>
			</td>
		</tr>`, ticketURL)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin: 0; padding: 0; background-color: #f4f7fa; font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color: #f4f7fa; padding: 32px 0;">
<tr><td align="center">
<table width="600" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 12px; box-shadow: 0 2px 12px rgba(0,0,0,0.08); overflow: hidden;">
	<!-- Header -->
	<tr>
		<td style="background: linear-gradient(135deg, #1e3a5f 0%%, #2d5a87 100%%); padding: 28px 30px;">
			<h1 style="color: #ffffff; margin: 0; font-size: 20px; font-weight: 700;">🔔 Tiket Baru Ditugaskan</h1>
			<p style="color: rgba(255,255,255,0.8); margin: 6px 0 0; font-size: 13px;">Halo %s, ada tiket baru untuk Anda</p>
		</td>
	</tr>
	<!-- Ticket Info Card -->
	<tr>
		<td style="padding: 24px 30px 0;">
			<table width="100%%" cellpadding="0" cellspacing="0" style="background: #f8fafc; border-radius: 10px; border: 1px solid #e2e8f0;">
				<tr>
					<td style="padding: 20px;">
						<h2 style="margin: 0 0 12px; font-size: 17px; color: #1e293b;">%d — %s</h2>
						<table cellpadding="0" cellspacing="0">
							<tr>
								<td style="padding: 3px 0;">
									<span style="display: inline-block; background: %s; color: #fff; padding: 3px 12px; border-radius: 12px; font-size: 11px; font-weight: 600;">%s</span>
									<span style="display: inline-block; background: #eef2ff; color: #4338ca; padding: 3px 12px; border-radius: 12px; font-size: 11px; font-weight: 600; margin-left: 6px;">%s %s</span>
								</td>
							</tr>
						</table>
						<table width="100%%" cellpadding="0" cellspacing="0" style="margin-top: 14px; font-size: 13px; color: #64748b;">
							<tr><td style="padding: 4px 0;">📧 Dari: <strong style="color: #334155;">%s</strong></td></tr>
							<tr><td style="padding: 4px 0;">📡 Sumber: <strong style="color: #334155;">%s</strong></td></tr>
							<tr><td style="padding: 4px 0;">🕐 Waktu: <strong style="color: #334155;">%s</strong></td></tr>
						</table>
					</td>
				</tr>
			</table>
		</td>
	</tr>
	<!-- Description -->
	<tr>
		<td style="padding: 20px 30px 0;">
			<h3 style="margin: 0 0 8px; font-size: 14px; color: #64748b; text-transform: uppercase; letter-spacing: 0.5px;">📝 Isi Pesan</h3>
			<div style="background: #fefefe; border-left: 4px solid #667eea; padding: 16px; border-radius: 0 8px 8px 0; color: #374151; font-size: 14px; line-height: 1.6;">
				%s
			</div>
		</td>
	</tr>
	<!-- Button -->
	%s
	<!-- Footer -->
	<tr>
		<td style="padding: 24px 30px; border-top: 1px solid #e5e7eb; margin-top: 16px;">
			<p style="color: #9ca3af; font-size: 12px; margin: 0; line-height: 1.6;">
				Silakan segera tindak lanjuti tiket ini.<br>
				%s Helpdesk System
			</p>
		</td>
	</tr>
</table>
</td></tr>
</table>
</body>
</html>`,
		engineerName,
		ticketID, title,
		priorityColor, priority,
		catEmoji, category,
		senderEmail,
		source,
		createdAt,
		descHTML,
		viewButton,
		companyName,
	)
}

// buildAutoReplyBody builds the fallback auto-reply email body (plain text)
func buildAutoReplyBody(ticketNum string, customer *models.Customer, senderName string, companyName string) string {
	customerName := "Valued Customer"

	if senderName != "" {
		customerName = senderName
	} else if customer != nil && customer.Name != "" && customer.Name != "Unknown Customer" {
		customerName = customer.Name
	}

	body := fmt.Sprintf(`Halo %s,

Terima kasih sudah menghubungi %s Support.

Kami telah membuat tiket %s untuk permintaan Anda.
Tim kami akan merespons dalam waktu 24 jam.

Detail Tiket:
- Ticket ID: %s
- Status: Open
- Prioritas: Medium

Gunakan nomor tiket ini untuk komunikasi selanjutnya.

Salam,
%s Support Team
`, customerName, companyName, ticketNum, ticketNum, companyName)

	return body
}
