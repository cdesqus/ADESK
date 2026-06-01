package email

import "time"

// EmailMessage represents a parsed email
type EmailMessage struct {
	From            string
	To              string
	Subject         string
	Body            string
	TextBody        string
	HTMLBody        string
	Date            time.Time
	MessageID       string
	Attachments     []AttachmentMetadata
	IsMultipart     bool
}

// AttachmentMetadata contains metadata about an attachment without storing the file
type AttachmentMetadata struct {
	Filename string
	MimeType string
	Size     int
}

// DomainMatchResult represents the result of domain matching
type DomainMatchResult struct {
	CustomerID   uint
	CustomerName string
	Domain       string
	IsMatch      bool
}

// EmailLog represents a log entry for email processing
type EmailLog struct {
	ID            string
	EmailMessageID string
	SenderEmail   string
	DomainMatched string
	CustomerID    *uint
	TicketID      *uint
	Status        string // SUCCESS, FAILED, UNKNOWN_DOMAIN
	ErrorMessage  string
	CreatedAt     time.Time
}
