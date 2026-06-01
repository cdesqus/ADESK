package email

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/mail"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	emersionMail "github.com/emersion/go-message/mail"
)

// ParseEmail parses an IMAP message into EmailMessage struct
func ParseEmail(msg *imap.Message) (*EmailMessage, error) {
	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}

	// Get raw email data
	var r io.Reader
	for _, body := range msg.Body {
		r = body
		break
	}
	if r == nil {
		return nil, fmt.Errorf("no body found in message")
	}

	mr, err := emersionMail.CreateReader(r)
	if err != nil {
		log.Printf("Failed to create message reader: %v", err)
		return nil, err
	}

	header := mr.Header

	// Parse basic headers
	from := header.Get("From")
	to := header.Get("To")
	subject := header.Get("Subject")
	messageID := header.Get("Message-ID")
	dateStr := header.Get("Date")

	// Decode subject if it's encoded
	dec := mime.WordDecoder{}
	decodedSubject, _ := dec.DecodeHeader(subject)
	if decodedSubject == "" {
		decodedSubject = subject
	}

	// Parse date
	var date time.Time
	if dateStr != "" {
		if parsedDate, err := mail.ParseDate(dateStr); err == nil {
			date = parsedDate
		} else {
			date = time.Now()
		}
	} else {
		date = time.Now()
	}

	// Extract text and HTML bodies
	textBody, htmlBody, attachments := parseBodyAndAttachments(mr)

	// Extract sender email from "From" header
	addr, err := mail.ParseAddress(from)
	senderEmail := from
	if err == nil && addr != nil {
		senderEmail = addr.Address
	}

	emailMsg := &EmailMessage{
		From:        senderEmail,
		To:          to,
		Subject:     decodedSubject,
		TextBody:    textBody,
		HTMLBody:    htmlBody,
		Body:        textBody, // Use text body as main body, fallback to HTML if empty
		Date:        date,
		MessageID:   messageID,
		Attachments: attachments,
		IsMultipart: len(attachments) > 0,
	}

	// If no text body but has HTML body, use HTML as body
	if emailMsg.Body == "" && htmlBody != "" {
		emailMsg.Body = stripHTML(htmlBody)
	}

	// Validate email
	if err := validateEmail(emailMsg); err != nil {
		log.Printf("Email validation error: %v", err)
		return nil, err
	}

	return emailMsg, nil
}

// parseBodyAndAttachments parses the multipart email body and extracts attachments
func parseBodyAndAttachments(mr *emersionMail.Reader) (string, string, []AttachmentMetadata) {
	var textBody, htmlBody string
	var attachments []AttachmentMetadata

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading part: %v", err)
			continue
		}

		switch h := part.Header.(type) {
		case *emersionMail.InlineHeader:
			body, _ := io.ReadAll(part.Body)
			mediaType, _, _ := h.ContentType()
			if strings.HasPrefix(mediaType, "text/plain") {
				textBody = string(body)
			} else if strings.HasPrefix(mediaType, "text/html") {
				htmlBody = string(body)
			}
		case *emersionMail.AttachmentHeader:
			filename, _ := h.Filename()
			body, _ := io.ReadAll(part.Body)
			mediaType, _, _ := h.ContentType()
			attachments = append(attachments, AttachmentMetadata{
				Filename: filename,
				MimeType: mediaType,
				Size:     len(body),
			})
		}
	}

	return textBody, htmlBody, attachments
}

// stripHTML removes HTML tags from text
func stripHTML(html string) string {
	// Simple HTML stripping - remove common tags
	replacements := [][2]string{
		{"<br>", "\n"},
		{"<br/>", "\n"},
		{"<br />", "\n"},
		{"<p>", ""},
		{"</p>", "\n"},
		{"<div>", ""},
		{"</div>", "\n"},
		{"<span>", ""},
		{"</span>", ""},
		{"<a href=\"", ""},
		{"</a>", ""},
		{"&nbsp;", " "},
		{"&lt;", "<"},
		{"&gt;", ">"},
		{"&amp;", "&"},
	}

	result := html
	for _, r := range replacements {
		result = strings.ReplaceAll(result, r[0], r[1])
	}

	// Remove remaining HTML tags with regex-like logic
	for {
		start := strings.Index(result, "<")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}

	return strings.TrimSpace(result)
}

// validateEmail validates parsed email
func validateEmail(email *EmailMessage) error {
	if email.From == "" {
		return fmt.Errorf("email must have From address")
	}

	if email.Subject == "" {
		return fmt.Errorf("email must have Subject")
	}

	if email.Body == "" && email.TextBody == "" && email.HTMLBody == "" {
		return fmt.Errorf("email must have body content")
	}

	// Basic email format validation
	if !strings.Contains(email.From, "@") {
		return fmt.Errorf("invalid sender email format")
	}

	return nil
}

// ExtractDomain extracts domain from email address
func ExtractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(parts[1]))
}
