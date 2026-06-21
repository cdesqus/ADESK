package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"ai-desk/internal/ai"
	"ai-desk/internal/models"
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

	htmlBody := buildAutoReplyHTML(customerName, body, ticketNum, ticketURL, sc.fromName)

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
func (sc *SMTPClient) SendHTMLEmail(toEmail, ccEmail, subject, htmlBody string) error {
	if toEmail == "" {
		return fmt.Errorf("recipient email is required")
	}

	from := fmt.Sprintf("%s <%s>", sc.fromName, sc.fromEmail)

	ccHeader := ""
	recipients := []string{toEmail}
	if ccEmail != "" {
		ccHeader = fmt.Sprintf("Cc: %s\r\n", ccEmail)
		recipients = append(recipients, ccEmail)
	}

	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\n%sSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n",
		from,
		toEmail,
		ccHeader,
		subject,
	)

	fullMessage := headers + htmlBody

	addr := fmt.Sprintf("%s:%d", sc.host, sc.port)
	auth := smtp.PlainAuth("", sc.user, sc.password, sc.host)

	err := smtp.SendMail(addr, auth, sc.fromEmail, recipients, []byte(fullMessage))
	if err != nil {
		log.Printf("Failed to send HTML email to %s (CC: %s): %v", toEmail, ccEmail, err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("HTML email sent successfully to %s (CC: %s)", toEmail, ccEmail)
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
func buildAutoReplyHTML(customerName string, replyBody string, ticketNum string, ticketURL string, companyName string) string {
	// Convert newlines to <br> for HTML
	htmlReply := strings.ReplaceAll(replyBody, "\n", "<br>")

	greeting := ""
	if customerName != "" {
		greeting = fmt.Sprintf("<p>Dear %s,</p>", customerName)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; color: #333333; line-height: 1.6; font-size: 14px; padding: 10px;">
	<div>
		%s
		<p>%s</p>
		<p>Berikut adalah no ticket anda: <strong>%s</strong></p>
		<br>
		<p>Salam,</p>
		<p>Best Regards,</p>
		<p>Helpdesk IDE Solusi</p>
	</div>
</body>
</html>`, greeting, htmlReply, ticketNum)
}

// BuildEngineerNotificationHTML builds a professional HTML email for engineer notification
func BuildEngineerNotificationHTML(engineerName string, ticketID uint, title, priority, source, senderEmail, createdAt, description, ticketURL, companyName, category string) string {
	// Priority styling
	priorityColor := "#f59e0b" // MEDIUM
	priorityBg := "#fef3c7"
	switch priority {
	case "LOW":
		priorityColor = "#16a34a"
		priorityBg = "#dcfce7"
	case "HIGH":
		priorityColor = "#ea580c"
		priorityBg = "#ffedd5"
	case "URGENT":
		priorityColor = "#dc2626"
		priorityBg = "#fee2e2"
	}

	// Category emoji
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
	if len(descPreview) > 1000 {
		descPreview = descPreview[:1000] + "..."
	}
	descHTML := strings.ReplaceAll(descPreview, "\n", "<br>")

	viewButton := ""
	if ticketURL != "" {
		viewButton = fmt.Sprintf(`
		<tr>
			<td colspan="2" align="center" style="padding: 30px 20px;">
				<a href="%s" style="display: inline-block; background-color: #2563eb; color: #ffffff; text-decoration: none; padding: 14px 32px; border-radius: 6px; font-weight: 600; font-size: 14px;">
					🔍 Buka Tiket di Dashboard
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
<table width="600" cellpadding="0" cellspacing="0" style="background-color: #ffffff; border-radius: 12px; box-shadow: 0 2px 12px rgba(0,0,0,0.08); overflow: hidden; border: 1px solid #e5e7eb; text-align: left;">
	<!-- Header -->
	<tr>
		<td style="background-color: #1e3a8a; padding: 24px 30px;">
			<h1 style="color: #ffffff; margin: 0; font-size: 20px;">Tiket Baru Ditugaskan</h1>
			<p style="color: #bfdbfe; margin: 5px 0 0; font-size: 14px;">Kepada Yth. %s</p>
		</td>
	</tr>

	<!-- Content -->
	<tr>
		<td style="padding: 30px;">
			<p style="color: #374151; font-size: 14px; margin: 0 0 20px 0; line-height: 1.6;">
				Anda telah ditugaskan untuk menangani tiket layanan berikut. Mohon untuk segera ditindaklanjuti.
			</p>

			<!-- Detail Box -->
			<table width="100%%" cellpadding="10" cellspacing="0" style="background-color: #f9fafb; border: 1px solid #e5e7eb; border-radius: 6px; margin-bottom: 20px; font-size: 14px; color: #374151;">
				<tr><td width="30%%" style="font-weight: bold;">Judul Tiket</td><td>: %s</td></tr>
				<tr><td style="font-weight: bold;">ID Tiket</td><td>: %d</td></tr>
				<tr><td style="font-weight: bold;">Pelanggan</td><td>: %s</td></tr>
				<tr><td style="font-weight: bold;">Sumber</td><td>: %s</td></tr>
				<tr><td style="font-weight: bold;">Waktu Dibuat</td><td>: %s</td></tr>
				<tr><td style="font-weight: bold;">Kategori</td><td>: %s %s</td></tr>
				<tr><td style="font-weight: bold;">Prioritas</td><td>: <span style="background-color: %s; color: %s; padding: 2px 8px; border-radius: 4px; font-size: 12px; font-weight: bold;">%s</span></td></tr>
			</table>

			<h3 style="color: #111827; font-size: 16px; margin: 0 0 10px 0;">Deskripsi Pesan</h3>
			<table width="100%%" cellpadding="15" cellspacing="0" style="border: 1px solid #e5e7eb; border-radius: 6px; background-color: #ffffff;">
				<tr>
					<td style="color: #374151; font-size: 14px; line-height: 1.6;">
						%s
					</td>
				</tr>
			</table>

			%s
		</td>
	</tr>

	<!-- Footer -->
	<tr>
		<td style="background-color: #f9fafb; padding: 20px 30px; border-top: 1px solid #e5e7eb; text-align: center;">
			<p style="color: #6b7280; font-size: 12px; margin: 0; line-height: 1.5;">
				Email ini otomatis dibuat oleh sistem %s.<br>
				Mohon tidak membalas langsung ke email ini.
			</p>
		</td>
	</tr>
</table>
</td></tr>
</table>
</body>
</html>`,
		engineerName,
		title,
		ticketID,
		senderEmail,
		source,
		createdAt,
		catEmoji, category,
		priorityBg, priorityColor, priority,
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
