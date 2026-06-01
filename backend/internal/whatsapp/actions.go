package whatsapp

import (
	"fmt"
	"log"
	"strconv"
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

func (h *ActionHandler) HandleCreateTicket(sessionName, fromPhone, content string) error {
	var customer *models.Customer
	var customerID uint

	// Try to find customer by phone number
	var engineerPhone models.EngineerWAPhone
	if err := h.db.Where("phone_number = ?", fromPhone).First(&engineerPhone).Error; err == nil {
		// Phone belongs to an engineer
		var engineer models.Engineer
		if err := h.db.First(&engineer, engineerPhone.EngineerID).Error; err == nil {
			customerID = engineer.CustomerID
			if err := h.db.First(&customer, engineer.CustomerID).Error; err != nil {
				return fmt.Errorf("customer not found")
			}
		}
	} else {
		// Try to find customer by matching phone in customer contacts
		// For now, assign to first active customer
		if err := h.db.Where("is_active = ?", true).First(&customer).Error; err != nil {
			h.messageSender.SendMessage(sessionName, fromPhone,
				"Maaf, kami tidak dapat memproses pesanan Anda. Silakan hubungi admin.")
			return fmt.Errorf("no active customer found")
		}
		customerID = customer.ID
	}

	// Create ticket
	ticket := models.Ticket{
		CustomerID:        customerID,
		Title:             fmt.Sprintf("WhatsApp Ticket from %s", fromPhone),
		Description:       content,
		Status:            "OPEN",
		Priority:          "MEDIUM",
		Source:            "WHATSAPP",
		WhatsappFrom:      fromPhone,
		WhatsappSessionID: sessionName,
		CreatedAt:         time.Now(),
	}

	if err := h.db.Create(&ticket).Error; err != nil {
		log.Printf("Error creating ticket: %v", err)
		h.messageSender.SendMessage(sessionName, fromPhone,
			"Maaf, terjadi kesalahan saat membuat tiket. Silakan coba lagi.")
		return err
	}

	// Assign to engineer (round-robin)
	engineer := h.assignEngineer(customerID)
	if engineer != nil {
		ticket.EngineerID = &engineer.ID
		h.db.Save(&ticket)

		// Notify engineer
		message := fmt.Sprintf("Tiket baru TK-%d dari %s: %s", ticket.ID, fromPhone, content)
		if engineer.WhatsappNumber != "" {
			h.messageSender.SendMessage(sessionName, engineer.WhatsappNumber, message)
		}
	}

	// Send confirmation to customer
	confirmMsg := fmt.Sprintf("Terima kasih! Kami buat tiket TK-%d untuk Anda. Tim kami akan segera membantu.", ticket.ID)
	h.messageSender.SendMessage(sessionName, fromPhone, confirmMsg)

	log.Printf("Ticket created: TK-%d from WhatsApp %s", ticket.ID, fromPhone)
	return nil
}

func (h *ActionHandler) HandleTicketUpdate(sessionName, fromPhone, ticketID, content string) error {
	// Verify sender is engineer
	var engineerPhone models.EngineerWAPhone
	if err := h.db.Where("phone_number = ?", fromPhone).First(&engineerPhone).Error; err != nil {
		h.messageSender.SendMessage(sessionName, fromPhone,
			"Anda tidak memiliki akses untuk mengubah tiket ini.")
		return fmt.Errorf("engineer phone not found")
	}

	// Get ticket
	var ticket models.Ticket
	id, _ := strconv.ParseUint(ticketID, 10, 32)
	if err := h.db.First(&ticket, id).Error; err != nil {
		h.messageSender.SendMessage(sessionName, fromPhone,
			fmt.Sprintf("Tiket TK-%s tidak ditemukan.", ticketID))
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

	// Send update to customer
	customerMsg := fmt.Sprintf("Update TK-%d: %s", ticket.ID, content)
	h.messageSender.SendMessage(sessionName, ticket.WhatsappFrom, customerMsg)

	log.Printf("Ticket TK-%d updated with progress", ticket.ID)
	return nil
}

func (h *ActionHandler) HandleTicketClose(sessionName, fromPhone, ticketID, content string) error {
	// Verify sender is engineer
	var engineerPhone models.EngineerWAPhone
	if err := h.db.Where("phone_number = ?", fromPhone).First(&engineerPhone).Error; err != nil {
		h.messageSender.SendMessage(sessionName, fromPhone,
			"Anda tidak memiliki akses untuk menutup tiket ini.")
		return fmt.Errorf("engineer phone not found")
	}

	// Get ticket
	var ticket models.Ticket
	id, _ := strconv.ParseUint(ticketID, 10, 32)
	if err := h.db.First(&ticket, id).Error; err != nil {
		h.messageSender.SendMessage(sessionName, fromPhone,
			fmt.Sprintf("Tiket TK-%s tidak ditemukan.", ticketID))
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

	// Send resolution to customer
	customerMsg := fmt.Sprintf("Tiket TK-%d ditutup. Resolusi: %s\nBalas dengan 'setuju' jika Anda puas dengan solusinya.", ticket.ID, content)
	h.messageSender.SendMessage(sessionName, ticket.WhatsappFrom, customerMsg)

	log.Printf("Ticket TK-%d closed", ticket.ID)
	return nil
}

func (h *ActionHandler) HandleTicketReopen(sessionName, fromPhone, ticketID string) error {
	// Get ticket
	var ticket models.Ticket
	id, _ := strconv.ParseUint(ticketID, 10, 32)
	if err := h.db.First(&ticket, id).Error; err != nil {
		h.messageSender.SendMessage(sessionName, fromPhone,
			fmt.Sprintf("Tiket TK-%s tidak ditemukan.", ticketID))
		return err
	}

	// Verify sender is original customer
	if ticket.WhatsappFrom != fromPhone {
		h.messageSender.SendMessage(sessionName, fromPhone,
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

	// Notify engineer
	if ticket.EngineerID != nil {
		var engineer models.Engineer
		if err := h.db.First(&engineer, *ticket.EngineerID).Error; err == nil && engineer.WhatsappNumber != "" {
			engineerMsg := fmt.Sprintf("Tiket TK-%d dibuka kembali oleh customer. Alasan: %s", ticket.ID, ticket.WhatsappFrom)
			h.messageSender.SendMessage(sessionName, engineer.WhatsappNumber, engineerMsg)
		}
	}

	log.Printf("Ticket TK-%d reopened", ticket.ID)
	return nil
}

func (h *ActionHandler) HandleStatusCheck(sessionName, fromPhone, ticketID string) error {
	// Get ticket
	var ticket models.Ticket
	id, _ := strconv.ParseUint(ticketID, 10, 32)
	if err := h.db.First(&ticket, id).Error; err != nil {
		h.messageSender.SendMessage(sessionName, fromPhone,
			fmt.Sprintf("Tiket TK-%s tidak ditemukan.", ticketID))
		return err
	}

	// Build status message
	statusMsg := fmt.Sprintf("Status TK-%d: %s\nJudul: %s\nDeskripsi: %s\nPrioritas: %s",
		ticket.ID, ticket.Status, ticket.Title, ticket.Description, ticket.Priority)

	h.messageSender.SendMessage(sessionName, fromPhone, statusMsg)
	log.Printf("Status check for TK-%d", ticket.ID)
	return nil
}

func (h *ActionHandler) assignEngineer(customerID uint) *models.Engineer {
	var engineer models.Engineer

	// Get engineers for this customer ordered by ID to do round-robin
	if err := h.db.Where("customer_id = ? AND is_active = ?", customerID, true).
		Order("id").First(&engineer).Error; err != nil {
		return nil
	}

	return &engineer
}
