package handlers

import (
	"errors"
	"log"
	"net/http"
	"time"

	"ai-desk/internal/email"
	"ai-desk/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EmailHandler struct {
	db              *gorm.DB
	domainMatcher   *email.DomainMatcher
	smtpClient      *email.SMTPClient
}

// NewEmailHandler creates a new email handler
func NewEmailHandler(db *gorm.DB, domainMatcher *email.DomainMatcher, smtpClient *email.SMTPClient) *EmailHandler {
	return &EmailHandler{
		db:            db,
		domainMatcher: domainMatcher,
		smtpClient:    smtpClient,
	}
}

// CreateTicketFromEmail creates a ticket from parsed email data
// POST /api/email/webhook (internal only)
func (h *EmailHandler) CreateTicketFromEmail(emailMsg *email.EmailMessage) (*models.Ticket, *models.EmailLog, error) {
	log.Printf("Processing email from %s with subject: %s", emailMsg.From, emailMsg.Subject)

	// Initialize log entry
	logEntry := &models.EmailLog{
		ID:             uuid.New().String(),
		EmailMessageID: emailMsg.MessageID,
		SenderEmail:    emailMsg.From,
		CreatedAt:      time.Now(),
	}

	// Match domain to customer
	matchResult := h.domainMatcher.MatchDomain(emailMsg.From)
	if !matchResult.IsMatch {
		logEntry.Status = "UNKNOWN_DOMAIN"
		logEntry.DomainMatched = email.ExtractDomain(emailMsg.From)
		logEntry.ErrorMessage = "No matching customer found for domain"
		log.Printf("Domain match failed for email: %s", emailMsg.From)

		// Try to find or create unknown customer
		var unknownCustomer models.Customer
		if err := h.db.Where("name = ?", "Unknown Customer").First(&unknownCustomer).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Create unknown customer if doesn't exist
				unknownCustomer = models.Customer{
					Name:     "Unknown Customer",
					Domain:   "",
					IsActive: true,
				}
				if err := h.db.Create(&unknownCustomer).Error; err != nil {
					logEntry.Status = "FAILED"
					logEntry.ErrorMessage = "Failed to create unknown customer: " + err.Error()
					return nil, logEntry, err
				}
			} else {
				logEntry.Status = "FAILED"
				logEntry.ErrorMessage = "Database error: " + err.Error()
				return nil, logEntry, err
			}
		}

		matchResult.CustomerID = unknownCustomer.ID
		matchResult.CustomerName = unknownCustomer.Name
		logEntry.DomainMatched = "UNKNOWN"
	}

	logEntry.DomainMatched = email.ExtractDomain(emailMsg.From)
	logEntry.CustomerID = &matchResult.CustomerID

	// Create ticket
	ticket := models.Ticket{
		CustomerID:  matchResult.CustomerID,
		Title:       emailMsg.Subject,
		Description: emailMsg.Body,
		Status:      "OPEN",
		Priority:    "MEDIUM",
		Source:      "EMAIL",
		EmailFrom:   emailMsg.From,
		Category:    "General",
	}

	if err := h.db.Create(&ticket).Error; err != nil {
		logEntry.Status = "FAILED"
		logEntry.ErrorMessage = "Failed to create ticket: " + err.Error()
		log.Printf("Failed to create ticket: %v", err)
		return nil, logEntry, err
	}

	ticketID := ticket.ID
	logEntry.TicketID = &ticketID
	logEntry.Status = "SUCCESS"

	log.Printf("Ticket created: TK-%d for customer %s from email %s", ticket.ID, matchResult.CustomerName, emailMsg.From)

	return &ticket, logEntry, nil
}

// ProcessEmailWithLogging creates ticket and logs the process
func (h *EmailHandler) ProcessEmailWithLogging(emailMsg *email.EmailMessage) (*models.Ticket, error) {
	ticket, logEntry, err := h.CreateTicketFromEmail(emailMsg)

	// Save log entry
	if logEntry != nil {
		if err := h.db.Create(logEntry).Error; err != nil {
			log.Printf("Failed to save email log: %v", err)
		}
	}

	if err != nil {
		return nil, err
	}

	// Send auto-reply if ticket created successfully
	if ticket != nil {
		var customer models.Customer
		if err := h.db.First(&customer, ticket.CustomerID).Error; err == nil {
			if err := h.smtpClient.SendAutoReply(emailMsg.From, emailMsg.Subject, ticket.ID, &customer); err != nil {
				log.Printf("Failed to send auto-reply: %v", err)
				// Don't fail if auto-reply fails, just log it
			}
		}
	}

	return ticket, nil
}

// EmailWebhookRequest represents the webhook request payload
type EmailWebhookRequest struct {
	From       string `json:"from" binding:"required"`
	To         string `json:"to"`
	Subject    string `json:"subject" binding:"required"`
	Body       string `json:"body" binding:"required"`
	TextBody   string `json:"text_body"`
	HTMLBody   string `json:"html_body"`
	MessageID  string `json:"message_id"`
	Date       string `json:"date"`
	Attachments []struct {
		Filename string `json:"filename"`
		MimeType string `json:"mime_type"`
		Size     int    `json:"size"`
	} `json:"attachments"`
}

// ProcessEmailWebhook handles incoming email webhook
// POST /api/email/webhook
func (h *EmailHandler) ProcessEmailWebhook(c *gin.Context) {
	var req EmailWebhookRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Convert request to EmailMessage
	emailMsg := &email.EmailMessage{
		From:      req.From,
		To:        req.To,
		Subject:   req.Subject,
		Body:      req.Body,
		TextBody:  req.TextBody,
		HTMLBody:  req.HTMLBody,
		MessageID: req.MessageID,
	}

	if req.Date != "" {
		if t, err := time.Parse(time.RFC3339, req.Date); err == nil {
			emailMsg.Date = t
		}
	}

	// Convert attachments
	for _, att := range req.Attachments {
		emailMsg.Attachments = append(emailMsg.Attachments, email.AttachmentMetadata{
			Filename: att.Filename,
			MimeType: att.MimeType,
			Size:     att.Size,
		})
	}

	// Process email
	ticket, err := h.ProcessEmailWithLogging(emailMsg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create ticket",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"ticket_id": ticket.ID,
		"status":    "success",
		"message":   "Ticket created from email",
	})
}
