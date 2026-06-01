package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/smtp"
	"strings"

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

// SendAutoReply sends an auto-reply email
func (sc *SMTPClient) SendAutoReply(toEmail, subject string, ticketID uint, customer *models.Customer) error {
	if toEmail == "" {
		return fmt.Errorf("recipient email is required")
	}

	// Build email body
	body := buildAutoReplyBody(ticketID, customer, sc.fromName)

	// Build reply subject
	replySubject := "Re: " + subject
	if !strings.HasPrefix(replySubject, "Re: Re: ") {
		replySubject = "Re: " + subject
	}

	// Create email message
	from := fmt.Sprintf("%s <%s>", sc.fromName, sc.fromEmail)
	to := toEmail

	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n",
		from,
		to,
		replySubject,
	)

	fullMessage := headers + body

	// Send email
	addr := fmt.Sprintf("%s:%d", sc.host, sc.port)

	// Use TLS for SMTP
	auth := smtp.PlainAuth("", sc.user, sc.password, sc.host)
	err := smtp.SendMail(addr, auth, sc.fromEmail, []string{toEmail}, []byte(fullMessage))
	if err != nil {
		log.Printf("Failed to send email to %s: %v", toEmail, err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Auto-reply sent successfully to %s for ticket %d", toEmail, ticketID)
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

// buildAutoReplyBody builds the auto-reply email body
func buildAutoReplyBody(ticketID uint, customer *models.Customer, companyName string) string {
	customerName := "Valued Customer"
	if customer != nil && customer.Name != "" {
		customerName = customer.Name
	}

	body := fmt.Sprintf(`Dear %s,

Thank you for contacting %s support.

We have created ticket ID: TK-%d for your request.
Our team will respond within 24 hours.

Ticket Details:
- Ticket ID: TK-%d
- Status: Open
- Priority: Medium

Please refer to the ticket ID in any future communication regarding this request.

Best regards,
%s Support Team

---
This is an automated response. Please do not reply to this email.
`, customerName, companyName, ticketID, ticketID, companyName)

	return body
}
