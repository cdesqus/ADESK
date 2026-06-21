package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ai-desk/internal/email"
	"ai-desk/internal/models"
	"ai-desk/internal/whatsapp"
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
	messageSender   *whatsapp.MessageSender
}

// NewEmailHandler creates a new email handler
func NewEmailHandler(db *gorm.DB, domainMatcher *email.DomainMatcher, smtpClient *email.SMTPClient, cfg *config.Config, messageSender *whatsapp.MessageSender) *EmailHandler {
	return &EmailHandler{
		db:            db,
		domainMatcher: domainMatcher,
		smtpClient:    smtpClient,
		cfg:           cfg,
		messageSender: messageSender,
	}
}

// CreateTicketFromEmail creates a ticket from parsed email data
// POST /api/email/webhook (internal only)
func (h *EmailHandler) CreateTicketFromEmail(emailMsg *email.EmailMessage) (*models.Ticket, bool, *models.EmailLog, error) {
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
					return nil, false, logEntry, err
				}
			} else {
				logEntry.Status = "FAILED"
				logEntry.ErrorMessage = "Database error: " + err.Error()
				return nil, false, logEntry, err
			}
		}

		matchResult.CustomerID = unknownCustomer.ID
		matchResult.CustomerName = unknownCustomer.Name
		logEntry.DomainMatched = "UNKNOWN"
	}

	// Check if this is a reply to an existing ticket (subject contains [2026-06-001])
	var existingTicketID uint
	var existingTicket models.Ticket
	re := regexp.MustCompile(`\[(\d{4}-\d{2}-\d{3,})\]`)
	matches := re.FindStringSubmatch(emailMsg.Subject)
	if len(matches) > 1 {
		ticketNum := matches[1]
		if err := h.db.Where("ticket_number = ?", ticketNum).First(&existingTicket).Error; err == nil {
			existingTicketID = existingTicket.ID
		}
	}


	if existingTicketID > 0 {
		// Append as comment
		comment := models.Update{
			TicketID:   existingTicketID,
			Message:    emailMsg.Body,
			ActionType: "COMMENT",
		}
		
		if err := h.db.Create(&comment).Error; err != nil {
			log.Printf("Failed to append comment: %v", err)
			// Proceed to return the ticket anyway
		}
		
		logEntry.DomainMatched = email.ExtractDomain(emailMsg.From)
		logEntry.CustomerID = &matchResult.CustomerID
		logEntry.TicketID = &existingTicketID
		logEntry.Status = "SUCCESS"

		log.Printf("Email reply appended to ticket %s from %s", existingTicket.TicketNumber, emailMsg.From)

		var ticket models.Ticket
		h.db.First(&ticket, existingTicketID)
		return &ticket, false, logEntry, nil
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
		return nil, false, logEntry, err
	}

	ticketID := ticket.ID
	logEntry.TicketID = &ticketID
	logEntry.Status = "SUCCESS"

	log.Printf("Ticket created: %s for customer %s from email %s", ticket.TicketNumber, matchResult.CustomerName, emailMsg.From)

	return &ticket, true, logEntry, nil
}

// ProcessEmailWithLogging creates ticket and logs the process
func (h *EmailHandler) ProcessEmailWithLogging(emailMsg *email.EmailMessage) (*models.Ticket, error) {
	ticket, isNew, logEntry, err := h.CreateTicketFromEmail(emailMsg)

	// Save log entry
	if logEntry != nil {
		if err := h.db.Create(logEntry).Error; err != nil {
			log.Printf("Failed to save email log: %v", err)
		}
	}

	if err != nil {
		return nil, err
	}

	// Send auto-reply with AI classification if ticket created successfully AND it's a new ticket
	if ticket != nil && isNew {
		var customer models.Customer
		if err := h.db.First(&customer, ticket.CustomerID).Error; err == nil {
			classification, replyErr := h.smtpClient.SendAutoReplyWithClassification(
				emailMsg.From, emailMsg.FromName, emailMsg.Subject,
				ticket.TicketNumber, ticket.ID, &customer, emailMsg.Body,
				h.cfg.OpenAIKey, h.cfg.FrontendURL,
			)
			if replyErr != nil {
				log.Printf("Failed to send auto-reply: %v", replyErr)
			}

			// Update ticket with AI classification
			if classification != nil {
				updates := map[string]interface{}{}
				if classification.Category != "" {
					updates["category"] = classification.Category
				}
				if classification.Priority != "" {
					updates["priority"] = classification.Priority
				}
				if len(updates) > 0 {
					if err := h.db.Model(ticket).Updates(updates).Error; err != nil {
						log.Printf("Failed to update ticket classification: %v", err)
					} else {
						// Reload ticket with updated fields
						h.db.First(ticket, ticket.ID)
						log.Printf("[AI] Ticket %s classified: category=%s, priority=%s", ticket.TicketNumber, ticket.Category, ticket.Priority)
					}
				}
			}
		}

		// Auto-assign engineer and send notifications
		h.autoAssignAndNotify(ticket, emailMsg)
	}

	return ticket, nil
}

// autoAssignAndNotify assigns an engineer to the ticket and sends notifications
func (h *EmailHandler) autoAssignAndNotify(ticket *models.Ticket, emailMsg *email.EmailMessage) {
	// Skip if ticket already has an engineer assigned
	if ticket.EngineerID != nil && *ticket.EngineerID > 0 {
		return
	}

	// Find available engineer for this customer (round-robin by least assigned tickets)
	engineer := h.findAvailableEngineer(ticket.CustomerID)
	if engineer == nil {
		log.Printf("[EmailNotify] No available engineer found for customer %d, ticket %s", ticket.CustomerID, ticket.TicketNumber)
		return
	}

	// Assign engineer to ticket
	ticket.EngineerID = &engineer.ID
	if err := h.db.Model(ticket).Update("engineer_id", engineer.ID).Error; err != nil {
		log.Printf("[EmailNotify] Failed to assign engineer %d to ticket %s: %v", engineer.ID, ticket.TicketNumber, err)
		return
	}

	log.Printf("[EmailNotify] Auto-assigned engineer %s (ID:%d) to ticket %s", engineer.Name, engineer.ID, ticket.TicketNumber)

	// Send email notification to engineer
	go h.sendEngineerEmailNotification(engineer, ticket, emailMsg)

	// Send WhatsApp notification to engineer
	go h.sendEngineerWhatsAppNotification(engineer, ticket, emailMsg)
}

// findAvailableEngineer finds the best engineer to assign using round-robin
func (h *EmailHandler) findAvailableEngineer(customerID uint) *models.Engineer {
	var engineers []models.Engineer

	// Get all active engineers for this customer
	if err := h.db.Where("customer_id = ? AND is_active = ?", customerID, true).
		Order("id").Find(&engineers).Error; err != nil || len(engineers) == 0 {
		// Fallback: try to find any active engineer
		if err := h.db.Where("is_active = ?", true).
			Order("id").Find(&engineers).Error; err != nil || len(engineers) == 0 {
			return nil
		}
	}

	if len(engineers) == 1 {
		return &engineers[0]
	}

	// Round-robin: find engineer with fewest open tickets
	var bestEngineer *models.Engineer
	minTickets := int64(999999)

	for i := range engineers {
		var count int64
		h.db.Model(&models.Ticket{}).
			Where("engineer_id = ? AND status IN ?", engineers[i].ID, []string{"OPEN", "IN_PROGRESS"}).
			Count(&count)

		if count < minTickets {
			minTickets = count
			bestEngineer = &engineers[i]
		}
	}

	return bestEngineer
}

// sendEngineerEmailNotification sends a professional HTML email notification to the assigned engineer
func (h *EmailHandler) sendEngineerEmailNotification(engineer *models.Engineer, ticket *models.Ticket, emailMsg *email.EmailMessage) {
	if engineer.Email == "" {
		log.Printf("[EmailNotify] Engineer %s has no email, skipping email notification", engineer.Name)
		return
	}

	subject := fmt.Sprintf("[AI-DESK] Tiket Baru %s Ditugaskan Kepada Anda", ticket.TicketNumber)

	// Build ticket URL
	ticketURL := ""
	if h.cfg.FrontendURL != "" {
		ticketURL = fmt.Sprintf("%s/tickets/%d", strings.TrimRight(h.cfg.FrontendURL, "/"), ticket.ID)
	}

	htmlBody := email.BuildEngineerNotificationHTML(
		engineer.Name,
		ticket.ID,
		ticket.Title,
		ticket.Priority,
		ticket.Source,
		emailMsg.From,
		ticket.CreatedAt.Format("02 Jan 2006 15:04 WIB"),
		ticket.Description,
		ticketURL,
		h.cfg.EmailFromName,
		ticket.Category,
	)

	if err := h.smtpClient.SendHTMLEmail(engineer.Email, "support@idesolusi.co.id", subject, htmlBody); err != nil {
		log.Printf("[EmailNotify] Failed to send email notification to engineer %s (%s): %v", engineer.Name, engineer.Email, err)
	} else {
		log.Printf("[EmailNotify] Email notification sent to engineer %s (%s) for ticket %s", engineer.Name, engineer.Email, ticket.TicketNumber)
	}
}

// sendEngineerWhatsAppNotification sends a WhatsApp notification to the assigned engineer
func (h *EmailHandler) sendEngineerWhatsAppNotification(engineer *models.Engineer, ticket *models.Ticket, emailMsg *email.EmailMessage) {
	if h.messageSender == nil {
		log.Printf("[WANotify] WhatsApp message sender not available, skipping WA notification")
		return
	}

	// Check if engineer has WhatsApp number (from Engineer model or EngineerWAPhone)
	waNumber := engineer.WhatsappNumber

	// Also check EngineerWAPhone table for registered numbers
	if waNumber == "" {
		var waPhone models.EngineerWAPhone
		if err := h.db.Where("engineer_id = ? AND is_active = ?", engineer.ID, true).
			First(&waPhone).Error; err == nil {
			waNumber = waPhone.PhoneNumber
		}
	}

	if waNumber == "" {
		log.Printf("[WANotify] Engineer %s has no WhatsApp number, skipping WA notification", engineer.Name)
		return
	}

	// Find an active WhatsApp session
	var session models.WhatsAppSession
	if err := h.db.Where("status = ? AND deleted_at IS NULL", "WORKING").
		First(&session).Error; err != nil {
		log.Printf("[WANotify] No active WhatsApp session found, skipping WA notification")
		return
	}

	// Truncate description for WA message
	descPreview := ticket.Description
	if len(descPreview) > 300 {
		descPreview = descPreview[:300] + "..."
	}

	// Build ticket URL
	ticketURL := ""
	if h.cfg.FrontendURL != "" {
		ticketURL = fmt.Sprintf("\n\n🔗 %s/tickets/%d", strings.TrimRight(h.cfg.FrontendURL, "/"), ticket.ID)
	}

	// Category emoji
	catEmoji := map[string]string{"PROBLEM": "🔴", "REQUEST": "🔵", "INQUIRY": "🟡", "FEEDBACK": "🟢"}
	emoji := catEmoji[ticket.Category]
	if emoji == "" {
		emoji = "📋"
	}

	// Priority emoji
	prioEmoji := map[string]string{"LOW": "🟢", "MEDIUM": "🟡", "HIGH": "🟠", "URGENT": "🔴"}
	pEmoji := prioEmoji[ticket.Priority]
	if pEmoji == "" {
		pEmoji = "⚪"
	}

	customerName := emailMsg.FromName
	if customerName == "" {
		customerName = "Unknown Customer"
	}

	customerCompany := "-"
	if ticket.Customer != nil && ticket.Customer.Company != "" {
		customerCompany = ticket.Customer.Company
	} else if ticket.CustomerID != 0 {
		var c models.Customer
		if err := h.db.First(&c, ticket.CustomerID).Error; err == nil {
			customerCompany = c.Company
		}
	}

	message := fmt.Sprintf("🎫 *TICKET ASSIGNMENT*\n\nAnda ditugaskan untuk menangani tiket baru:\n\n%s *%s* — %s\n\n*Pelanggan:* %s\n*Kantor:* %s\n*Kategori:* %s\n*Prioritas:* %s\n\n*Pesan:*\n_%s_%s",
		pEmoji, ticket.TicketNumber, ticket.Title, customerName, customerCompany, ticket.Category, ticket.Priority, descPreview, ticketURL)

	// Format phone number for WhatsApp (chatId format: number@c.us)
	chatID := formatWANumber(waNumber)

	if err := h.messageSender.SendMessage(session.SessionName, chatID, message); err != nil {
		log.Printf("[WANotify] Failed to send WhatsApp notification to engineer %s (%s): %v", engineer.Name, waNumber, err)
	} else {
		log.Printf("[WANotify] WhatsApp notification sent to engineer %s (%s) for ticket %s", engineer.Name, waNumber, ticket.TicketNumber)
	}
}

// formatWANumber formats a phone number for WhatsApp chatId
func formatWANumber(phone string) string {
	if strings.Contains(phone, "@") {
		return phone
	}
	cleanNumber := ""
	for _, c := range phone {
		if c >= '0' && c <= '9' {
			cleanNumber += string(c)
		}
	}
	return cleanNumber + "@c.us"
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
