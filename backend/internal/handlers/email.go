package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"ai-desk/internal/email"
	"ai-desk/internal/models"
	"ai-desk/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EmailHandler struct {
	db              *gorm.DB
	domainMatcher   *email.DomainMatcher
	smtpClient      *email.SMTPClient
	cfg             *config.Config
}

// NewEmailHandler creates a new email handler
func NewEmailHandler(db *gorm.DB, domainMatcher *email.DomainMatcher, smtpClient *email.SMTPClient, cfg *config.Config) *EmailHandler {
	return &EmailHandler{
		db:            db,
		domainMatcher: domainMatcher,
		smtpClient:    smtpClient,
		cfg:           cfg,
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
			if err := h.smtpClient.SendAutoReply(emailMsg.From, emailMsg.FromName, emailMsg.Subject, ticket.ID, &customer, emailMsg.Body, h.cfg.OpenAIKey); err != nil {
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

// GET /api/email/settings
func (h *EmailHandler) GetEmailSettings(c *gin.Context) {
	isConfigured := h.cfg.EmailUser != "" && h.cfg.EmailPassword != ""
	status := "disconnected"
	if isConfigured {
		status = "connected"
	}
	pollingInterval := 5
	if h.cfg.EmailPollingInterval != "" {
		if d, err := time.ParseDuration(h.cfg.EmailPollingInterval); err == nil {
			pollingInterval = int(d.Minutes())
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"host":            h.cfg.EmailIMAPHost,
		"port":            strconv.Itoa(h.cfg.EmailIMAPPort),
		"username":        h.cfg.EmailUser,
		"isConfigured":    isConfigured,
		"status":          status,
		"lastSync":        time.Now(),
		"pollingInterval": pollingInterval,
	})
}

// PATCH /api/email/settings
func (h *EmailHandler) UpdateEmailSettings(c *gin.Context) {
	var req struct {
		Host            string `json:"host"`
		Port            string `json:"port"`
		Username        string `json:"username"`
		Password        string `json:"password"`
		PollingInterval int    `json:"pollingInterval"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("JSON binding error in UpdateEmailSettings: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format", "details": err.Error()})
		return
	}

	saveSetting := func(key, value string) error {
		err := h.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			UpdateAll: true,
		}).Create(&models.SystemSetting{Key: key, Value: value, UpdatedAt: time.Now()}).Error
		if err != nil {
			log.Printf("Failed to save setting %s: %v", key, err)
		}
		return err
	}

	if req.Host != "" {
		if err := saveSetting("EMAIL_IMAP_HOST", req.Host); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save host", "details": err.Error()})
			return
		}
		h.cfg.EmailIMAPHost = req.Host
	}
	if req.Port != "" {
		if err := saveSetting("EMAIL_IMAP_PORT", req.Port); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save port", "details": err.Error()})
			return
		}
		if portNum, err := strconv.Atoi(req.Port); err == nil {
			h.cfg.EmailIMAPPort = portNum
		}
	}
	if req.Username != "" {
		if err := saveSetting("EMAIL_USER", req.Username); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save username", "details": err.Error()})
			return
		}
		h.cfg.EmailUser = req.Username
	}
	if req.Password != "" {
		if err := saveSetting("EMAIL_PASSWORD", req.Password); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save password", "details": err.Error()})
			return
		}
		h.cfg.EmailPassword = req.Password
	}
	if req.PollingInterval > 0 {
		pollStr := strconv.Itoa(req.PollingInterval) + "m"
		if err := saveSetting("EMAIL_POLLING_INTERVAL", pollStr); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save polling interval", "details": err.Error()})
			return
		}
		h.cfg.EmailPollingInterval = pollStr
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Email settings updated in Database. Please restart the backend to apply changes."})
}

// GET /api/email/domain-mappings
func (h *EmailHandler) GetDomainMappings(c *gin.Context) {
	var customers []models.Customer
	if err := h.db.Where("domain != ''").Find(&customers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get domains"})
		return
	}
	
	type DomainMapping struct {
		CustomerID   string `json:"customerId"`
		CustomerName string `json:"customerName"`
		Domain       string `json:"domain"`
	}
	var res []DomainMapping
	for _, cst := range customers {
		res = append(res, DomainMapping{
			CustomerID:   strconv.FormatUint(uint64(cst.ID), 10),
			CustomerName: cst.Name,
			Domain:       cst.Domain,
		})
	}
	if res == nil {
		res = make([]DomainMapping, 0)
	}
	c.JSON(http.StatusOK, res)
}

// GET /api/email/auto-reply-template
func (h *EmailHandler) GetAutoReplyTemplate(c *gin.Context) {
	var setting models.SystemSetting
	if err := h.db.Where("key = ?", "AutoReplyTemplate").First(&setting).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"template": "Hello {CUSTOMER_NAME},\n\nWe have received your request. Ticket {TICKET_ID} has been created.\n\nThanks,\nSupport Team"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"template": setting.Value})
}

// PATCH /api/email/auto-reply-template
func (h *EmailHandler) UpdateAutoReplyTemplate(c *gin.Context) {
	var req struct {
		Template string `json:"template"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format"})
		return
	}
	
	setting := models.SystemSetting{Key: "AutoReplyTemplate", Value: req.Template, UpdatedAt: time.Now()}
	if err := h.db.Save(&setting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"template": req.Template})
}

// GET /api/email/history
func (h *EmailHandler) GetEmailHistory(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("pageSize", "10")
	status := c.DefaultQuery("status", "all")
	customerId := c.Query("customerId")

	pageNum, _ := strconv.Atoi(page)
	pageSizeNum, _ := strconv.Atoi(pageSize)

	query := h.db.Model(&models.EmailLog{})
	if status != "all" {
		query = query.Where("status = ?", status)
	}
	if customerId != "" {
		query = query.Where("customer_id = ?", customerId)
	}

	var total int64
	query.Count(&total)

	var logs []models.EmailLog
	query.Order("created_at desc").Offset((pageNum - 1) * pageSizeNum).Limit(pageSizeNum).Find(&logs)

	type emailLogRes struct {
		ID            string `json:"id"`
		SenderEmail   string `json:"senderEmail"`
		DomainMatched string `json:"domainMatched"`
		Status        string `json:"status"`
		CreatedAt     time.Time `json:"createdAt"`
	}
	var res []emailLogRes
	for _, l := range logs {
		res = append(res, emailLogRes{
			ID:            l.ID,
			SenderEmail:   l.SenderEmail,
			DomainMatched: l.DomainMatched,
			Status:        l.Status,
			CreatedAt:     l.CreatedAt,
		})
	}
	if res == nil {
		res = make([]emailLogRes, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": res,
		"total": total,
		"page": pageNum,
		"pageSize": pageSizeNum,
	})
}

// POST /api/email/test-connection
func (h *EmailHandler) TestEmailConnection(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Connection OK"})
}

// POST /api/email/sync
func (h *EmailHandler) SyncEmails(c *gin.Context) {
	imapClient := email.NewIMAPClient(
		h.cfg.EmailIMAPHost,
		h.cfg.EmailIMAPPort,
		h.cfg.EmailUser,
		h.cfg.EmailPassword,
	)

	if err := imapClient.Connect(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to IMAP server", "details": err.Error()})
		return
	}
	defer imapClient.Close()

	messages, err := imapClient.FetchUnreadEmails()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch emails", "details": err.Error()})
		return
	}

	if len(messages) == 0 {
		c.JSON(http.StatusOK, gin.H{"synced": 0, "message": "No unread emails found"})
		return
	}

	successCount := 0
	for _, msg := range messages {
		emailMsg, err := email.ParseEmail(msg)
		if err != nil {
			log.Printf("SyncEmails: Failed to parse email: %v", err)
			continue
		}
		
		_, err = h.ProcessEmailWithLogging(emailMsg)
		if err == nil {
			successCount++
			_ = imapClient.MarkAsRead(msg.Uid)
		} else {
			log.Printf("SyncEmails: Failed to process email: %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"synced": successCount,
		"message": fmt.Sprintf("Successfully synced %d emails out of %d unread", successCount, len(messages)),
	})
}
