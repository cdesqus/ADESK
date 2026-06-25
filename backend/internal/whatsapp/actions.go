package whatsapp

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"ai-desk/internal/models"
	"gorm.io/gorm"
)

type ActionHandler struct {
	db            *gorm.DB
	messageSender *MessageSender
	wahaClient    *WahaClient
}

func NewActionHandler(db *gorm.DB, messageSender *MessageSender, wahaClient *WahaClient) *ActionHandler {
	return &ActionHandler{
		db:            db,
		messageSender: messageSender,
		wahaClient:    wahaClient,
	}
}

func summarizeWhatsAppTicket(content string) (string, string) {
	clean := strings.TrimSpace(content)
	if clean == "" {
		return "WhatsApp issue", ""
	}

	// Remove any leading mention syntax like @Helpdesk or @12345
	reMention := regexp.MustCompile(`(?i)^@(?:helpdesk(?:\s+ide)?|[0-9]{5,})\s*`)
	clean = reMention.ReplaceAllString(clean, "")
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return "WhatsApp issue", ""
	}

	// If the message is short, use a fixed WhatsApp prefix for clarity
	words := strings.Fields(clean)
	if len(words) <= 5 {
		title := fmt.Sprintf("WhatsApp: %s", clean)
		if len(title) > 80 {
			title = strings.TrimSpace(title[:80])
		}
		return title, clean
	}

	// Prefer first sentence for title when available
	if idx := strings.IndexAny(clean, ".!?\n"); idx > 0 {
		title := strings.TrimSpace(clean[:idx])
		if len(title) > 80 {
			title = strings.TrimSpace(title[:80])
		}
		if title == "" {
			title = strings.Join(words[:8], " ")
		}
		return title, clean
	}

	// Otherwise, use first 8 words as title
	if len(words) > 8 {
		title := strings.Join(words[:8], " ")
		if len(title) > 80 {
			title = strings.TrimSpace(title[:80])
		}
		return title, clean
	}

	return clean, clean
}

func (h *ActionHandler) HandleCreateTicket(sessionName, senderPhone, replyTo string, isGroup bool, content string, aiReply string) error {
	log.Printf("[WhatsApp] HandleCreateTicket session=%s sender=%s replyTo=%s isGroup=%t content=%q", sessionName, senderPhone, replyTo, isGroup, content)
	var customer *models.Customer
	var customerID uint
	foundCustomer := false

	// If it's a group, try to find mapping in CustomerWAGroup
	if isGroup {
		var waGroup models.CustomerWAGroup
		if err := h.db.Where("group_id = ?", replyTo).First(&waGroup).Error; err == nil {
			if waGroup.IsSupport {
				// For support groups, try to extract customer name from the content
				var customers []models.Customer
				if err := h.db.Where("is_active = ?", true).Find(&customers).Error; err == nil {
					contentLower := strings.ToLower(content)
					for _, c := range customers {
						if c.Name != "" && strings.Contains(contentLower, strings.ToLower(c.Name)) {
							customerID = c.ID
							foundCustomer = true
							break
						}
					}
				}
				
				if !foundCustomer {
					h.messageSender.SendMessage(sessionName, replyTo,
						"Untuk membuat tiket dari grup support internal, mohon sebutkan nama customer di dalam pesan Anda (contoh: 'Buatkan tiket untuk PT Kaumtech: ...').")
					return fmt.Errorf("customer name not mentioned in support group")
				}
			} else if waGroup.CustomerID != nil {
				// Assign to specific customer
				if err := h.db.First(&customer, *waGroup.CustomerID).Error; err == nil {
					customerID = customer.ID
					foundCustomer = true
				}
			}
		}
	}

	// Try to find customer by phone number if not mapped via group
	if !foundCustomer {
		var engineerPhone models.EngineerWAPhone
		if err := h.db.Where("phone_number = ?", senderPhone).First(&engineerPhone).Error; err == nil {
			// Phone belongs to an engineer
			var engineer models.Engineer
			if err := h.db.First(&engineer, engineerPhone.EngineerID).Error; err == nil && engineer.CustomerID != nil {
				customerID = *engineer.CustomerID
				if err := h.db.First(&customer, customerID).Error; err == nil {
					foundCustomer = true
				}
			}
		}
	}
	
	if !foundCustomer {
		// Try to find customer by matching phone in customer contacts
		// For now, assign to first active customer
		if err := h.db.Where("is_active = ?", true).First(&customer).Error; err != nil {
			h.messageSender.SendMessage(sessionName, replyTo,
				"Maaf, kami tidak dapat memproses pesanan Anda. Silakan hubungi admin.")
			return fmt.Errorf("no active customer found")
		}
		customerID = customer.ID
	}

	const maxRetries = 5
	yearMonth := time.Now().Format("2006-01")

	var ticket models.Ticket
	for attempt := 1; attempt <= maxRetries; attempt++ {
		tx := h.db.Begin()
		if tx.Error != nil {
			log.Printf("Failed to start transaction for ticket creation: %v", tx.Error)
			h.messageSender.SendMessage(sessionName, replyTo,
				"Maaf, terjadi kesalahan saat membuat tiket. Silakan coba lagi.")
			return tx.Error
		}

		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", yearMonth).Error; err != nil {
			tx.Rollback()
			log.Printf("Failed to acquire advisory lock for ticket numbering: %v", err)
			h.messageSender.SendMessage(sessionName, replyTo,
				"Maaf, terjadi kesalahan saat membuat tiket. Silakan coba lagi.")
			return err
		}

		var maxSeq int
		if err := tx.Raw(
			"SELECT COALESCE(MAX(CAST(SPLIT_PART(ticket_number, '-', 3) AS INTEGER)), 0) FROM tickets WHERE ticket_number LIKE ?",
			yearMonth+"-%",
		).Scan(&maxSeq).Error; err != nil {
			tx.Rollback()
			log.Printf("Failed to compute next ticket sequence: %v", err)
			h.messageSender.SendMessage(sessionName, replyTo,
				"Maaf, terjadi kesalahan saat membuat tiket. Silakan coba lagi.")
			return err
		}

		sequence := maxSeq + 1

		title, description := summarizeWhatsAppTicket(content)
		ticket = models.Ticket{
			CustomerID:        customerID,
			Title:             title,
			Description:       description,
			Status:            "OPEN",
			Priority:          "MEDIUM",
			Source:            "WHATSAPP",
			WhatsappFrom:      senderPhone,
			WhatsappSessionID: sessionName,
			CreatedAt:         time.Now(),
		}
		ticket.TicketNumber = fmt.Sprintf("%s-%03d", yearMonth, sequence)
		log.Printf("[WhatsApp] Creating ticket with number %s (maxSeq=%d) attempt=%d", ticket.TicketNumber, maxSeq, attempt)

		err := tx.Create(&ticket).Error
		if err != nil {
			tx.Rollback()
			if strings.Contains(err.Error(), "tickets_ticket_number_key") || strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				log.Printf("Duplicate ticket number detected on attempt %d (value=%s), retrying...", attempt, ticket.TicketNumber)
				continue
			}

			log.Printf("Error creating ticket: %v", err)
			h.messageSender.SendMessage(sessionName, replyTo,
				"Maaf, terjadi kesalahan saat membuat tiket. Silakan coba lagi.")
			return err
		}

		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			if strings.Contains(err.Error(), "tickets_ticket_number_key") || strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				log.Printf("Duplicate ticket number detected during commit on attempt %d (value=%s), retrying...", attempt, ticket.TicketNumber)
				continue
			}
			log.Printf("Failed to commit ticket transaction: %v", err)
			h.messageSender.SendMessage(sessionName, replyTo,
				"Maaf, terjadi kesalahan saat membuat tiket. Silakan coba lagi.")
			return err
		}

		break
	}

	if ticket.ID == 0 {
		log.Printf("Failed to create ticket after %d attempts due to duplicate ticket numbers", maxRetries)
		h.messageSender.SendMessage(sessionName, replyTo,
			"Maaf, terjadi kesalahan saat membuat tiket. Silakan coba lagi.")
		return fmt.Errorf("failed to create ticket after %d attempts", maxRetries)
	}

	// Assign to engineer (round-robin)
	engineer := h.assignEngineer(customerID)
	if engineer != nil {
		ticket.EngineerID = &engineer.ID
		h.db.Save(&ticket)

		// Notify engineer
		message := fmt.Sprintf("Tiket baru %s dari %s: %s", ticket.TicketNumber, senderPhone, content)
		if engineer.WhatsappNumber != "" {
			h.messageSender.SendMessage(sessionName, formatPhone(engineer.WhatsappNumber), message)
		}
	}

	// Send confirmation to customer (or group)
	confirmMsg := ""
	if aiReply != "" {
		confirmMsg = strings.ReplaceAll(aiReply, "{ticket_number}", ticket.TicketNumber)
		if !strings.Contains(confirmMsg, ticket.TicketNumber) {
			confirmMsg = fmt.Sprintf("%s\n(Ref: %s)", confirmMsg, ticket.TicketNumber)
		}
	} else {
		confirmMsg = fmt.Sprintf("Siap, tiket sudah dibuatkan dengan nomor tiket *%s* ya. Tim kami akan segera mengeceknya! 🛠️", ticket.TicketNumber)
	}
	h.messageSender.SendMessage(sessionName, replyTo, confirmMsg)

	log.Printf("Ticket created: %s from WhatsApp %s (ReplyTo: %s)", ticket.TicketNumber, senderPhone, replyTo)
	return nil
}

func (h *ActionHandler) HandleTicketUpdate(sessionName, senderPhone, replyTo string, isGroup bool, ticketID, content string) error {
	// Verify sender is engineer. Accept multiple formats: raw webhook value,
	// normalized numeric id (no suffix), or formatted chat id with @c.us.
	norm := normalizeWhatsAppID(senderPhone)
	var engineerPhone models.EngineerWAPhone
	if err := h.db.Where("phone_number = ? OR phone_number = ? OR phone_number = ?", senderPhone, norm, formatPhone(norm)).First(&engineerPhone).Error; err != nil {
		h.messageSender.SendMessage(sessionName, replyTo,
			"Anda tidak memiliki akses untuk mengubah tiket ini.")
		return fmt.Errorf("engineer phone not found")
	}

	// Get ticket
	var ticket models.Ticket
	if err := h.db.Where("ticket_number = ?", ticketID).First(&ticket).Error; err != nil {
		h.messageSender.SendMessage(sessionName, replyTo,
			fmt.Sprintf("Tiket %s tidak ditemukan.", ticketID))
		return err
	}

	// Update status to IN_PROGRESS
	ticket.Status = "IN_PROGRESS"
	h.db.Save(&ticket)

	// Add update/comment
	update := models.Update{
		TicketID:   ticket.ID,
		EngineerID: &engineerPhone.EngineerID,
		Message:    content,
		ActionType: "COMMENT",
		CreatedAt:  time.Now(),
	}
	h.db.Create(&update)

	// Send update back to the chat (group or private)
	customerMsg := fmt.Sprintf("Update %s: %s", ticket.TicketNumber, content)
	h.messageSender.SendMessage(sessionName, replyTo, customerMsg)
	
	// Also notify original ticket creator if they are not in the same chat
	if !isGroup && ticket.WhatsappFrom != "" && ticket.WhatsappFrom != replyTo {
		h.messageSender.SendMessage(sessionName, formatPhone(ticket.WhatsappFrom), customerMsg)
	}

	log.Printf("Ticket %s updated with progress by %s", ticket.TicketNumber, senderPhone)
	return nil
}

func (h *ActionHandler) HandleTicketClose(sessionName, senderPhone, replyTo string, isGroup bool, ticketID, content string) error {
	// Verify sender is engineer. Accept multiple formats like in updates.
	norm := normalizeWhatsAppID(senderPhone)
	var engineerPhone models.EngineerWAPhone
	if err := h.db.Where("phone_number = ? OR phone_number = ? OR phone_number = ?", senderPhone, norm, formatPhone(norm)).First(&engineerPhone).Error; err != nil {
		h.messageSender.SendMessage(sessionName, replyTo,
			"Anda tidak memiliki akses untuk menutup tiket ini.")
		return fmt.Errorf("engineer phone not found")
	}

	// Get ticket
	var ticket models.Ticket
	if err := h.db.Where("ticket_number = ?", ticketID).First(&ticket).Error; err != nil {
		h.messageSender.SendMessage(sessionName, replyTo,
			fmt.Sprintf("Tiket %s tidak ditemukan.", ticketID))
		return err
	}

	// Update status to RESOLVED
	now := time.Now()
	ticket.Status = "RESOLVED"
	ticket.ResolvedAt = &now
	h.db.Save(&ticket)

	// Add update/comment
	update := models.Update{
		TicketID:   ticket.ID,
		EngineerID: &engineerPhone.EngineerID,
		Message:    content,
		ActionType: "STATUS_CHANGE",
		CreatedAt:  time.Now(),
	}
	h.db.Create(&update)

	// Send resolution back to the chat
	customerMsg := fmt.Sprintf("Tiket %s ditutup. Resolusi: %s\nBalas dengan '%s reopen' jika masih ada masalah.", ticket.TicketNumber, content, ticket.TicketNumber)
	h.messageSender.SendMessage(sessionName, replyTo, customerMsg)

	// Notify original ticket creator if not the same chat
	if !isGroup && ticket.WhatsappFrom != "" && ticket.WhatsappFrom != replyTo {
		h.messageSender.SendMessage(sessionName, formatPhone(ticket.WhatsappFrom), customerMsg)
	}

	log.Printf("Ticket %s closed by %s", ticket.TicketNumber, senderPhone)
	return nil
}

func (h *ActionHandler) HandleTicketReopen(sessionName, senderPhone, replyTo string, isGroup bool, ticketID string) error {
	// Get ticket
	var ticket models.Ticket
	if err := h.db.Where("ticket_number = ?", ticketID).First(&ticket).Error; err != nil {
		h.messageSender.SendMessage(sessionName, replyTo,
			fmt.Sprintf("Tiket %s tidak ditemukan.", ticketID))
		return err
	}

	// Verify sender is original customer (or allow anyone in the group chat to reopen)
	if normalizeWhatsAppID(ticket.WhatsappFrom) != normalizeWhatsAppID(senderPhone) && !isGroup {
		h.messageSender.SendMessage(sessionName, replyTo,
			"Anda tidak memiliki akses untuk membuka kembali tiket ini.")
		return fmt.Errorf("phone mismatch")
	}

	// Update status to REOPENED
	ticket.Status = "REOPENED"
	h.db.Save(&ticket)

	// Add update
	update := models.Update{
		TicketID:   ticket.ID,
		Message:    "Tiket dibuka kembali oleh customer",
		ActionType: "REOPENED",
		CreatedAt:  time.Now(),
	}
	h.db.Create(&update)

	// Notify in chat
	h.messageSender.SendMessage(sessionName, replyTo, fmt.Sprintf("Tiket %s telah dibuka kembali.", ticket.TicketNumber))

	// Notify engineer
	if ticket.EngineerID != nil {
		var engineer models.Engineer
		if err := h.db.First(&engineer, *ticket.EngineerID).Error; err == nil && engineer.WhatsappNumber != "" {
			engineerMsg := fmt.Sprintf("Tiket %s dibuka kembali oleh customer. Alasan: %s", ticket.TicketNumber, ticket.WhatsappFrom)
			h.messageSender.SendMessage(sessionName, formatPhone(engineer.WhatsappNumber), engineerMsg)
		}
	}

	log.Printf("Ticket %s reopened", ticket.TicketNumber)
	return nil
}

func (h *ActionHandler) HandleStatusCheck(sessionName, senderPhone, replyTo string, isGroup bool, ticketID string) error {
	// Get ticket
	var ticket models.Ticket
	if err := h.db.Where("ticket_number = ?", ticketID).First(&ticket).Error; err != nil {
		h.messageSender.SendMessage(sessionName, replyTo,
			fmt.Sprintf("Tiket %s tidak ditemukan.", ticketID))
		return err
	}

	// Build status message
	statusMsg := fmt.Sprintf("Status %s: %s\nJudul: %s\nDeskripsi: %s\nPrioritas: %s",
		ticket.TicketNumber, ticket.Status, ticket.Title, ticket.Description, ticket.Priority)

	h.messageSender.SendMessage(sessionName, replyTo, statusMsg)
	log.Printf("Status check for %s by %s", ticket.TicketNumber, senderPhone)
	return nil
}

func (h *ActionHandler) assignEngineer(customerID uint) *models.Engineer {
	var engineer models.Engineer

	// First try to assign an active engineer for this customer
	if err := h.db.Where("customer_id = ? AND is_active = ?", customerID, true).
		Order("id").First(&engineer).Error; err == nil {
		return &engineer
	}

	// Fallback to any active engineer if no customer-specific engineer exists
	if err := h.db.Where("is_active = ?", true).
		Order("id").First(&engineer).Error; err == nil {
		return &engineer
	}

	return nil
}

// formatPhone formats a phone number for WhatsApp chatId
func formatPhone(phone string) string {
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
