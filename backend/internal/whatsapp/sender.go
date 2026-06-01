package whatsapp

import (
	"fmt"
	"log"
	"time"

	"ai-desk/internal/models"
	"gorm.io/gorm"
)

type MessageSender struct {
	client *WahaClient
	db     *gorm.DB
}

func NewMessageSender(client *WahaClient, db *gorm.DB) *MessageSender {
	return &MessageSender{
		client: client,
		db:     db,
	}
}

func (s *MessageSender) SendMessage(sessionName, phone, message string) error {
	return s.sendWithRetry(sessionName, phone, message, 3)
}

func (s *MessageSender) sendWithRetry(sessionName, phone, message string, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		msgID, err := s.client.SendMessage(sessionName, phone, message)
		if err == nil {
			log.Printf("Message sent successfully via WhatsApp: msgID=%s to=%s", msgID, phone)
			s.logMessage(sessionName, "", msgID, phone, message, "text", "outbound", "delivered", nil, nil)
			return nil
		}
		lastErr = err
		log.Printf("Attempt %d failed to send WhatsApp message: %v", attempt+1, err)
		if attempt < maxRetries-1 {
			time.Sleep(time.Duration((attempt+1)*2) * time.Second)
		}
	}

	s.logMessage(sessionName, "", "", phone, message, "text", "outbound", "failed", nil, nil)
	return fmt.Errorf("failed to send message after %d retries: %v", maxRetries, lastErr)
}

func (s *MessageSender) SendGroupMessage(sessionName, groupID, message string) error {
	if groupID == "" {
		return fmt.Errorf("group id is required for group messages")
	}
	return s.sendWithRetry(sessionName, groupID, message, 3)
}

func (s *MessageSender) logMessage(sessionName, messageID, sentID, phone, body, msgType, direction, status string, customerID, ticketID *uint) {
	logEntry := models.WhatsAppLog{
		ID:          fmt.Sprintf("wa_%d", time.Now().UnixNano()),
		SessionName: sessionName,
		MessageID:   messageID,
		FromPhone:   "system",
		ToPhone:     phone,
		Body:        body,
		MessageType: msgType,
		Direction:   direction,
		Status:      status,
		CustomerID:  customerID,
		TicketID:    ticketID,
		CreatedAt:   time.Now(),
	}

	if err := s.db.Create(&logEntry).Error; err != nil {
		log.Printf("Failed to log WhatsApp message: %v", err)
	}
}
