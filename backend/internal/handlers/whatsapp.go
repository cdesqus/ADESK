package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"strconv"
	"time"

	"ai-desk/internal/models"
	"ai-desk/internal/whatsapp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WhatsAppHandler struct {
	db             *gorm.DB
	wahaClient     *whatsapp.WahaClient
	messageSender  *whatsapp.MessageSender
	actionHandler  *whatsapp.ActionHandler
}

func NewWhatsAppHandler(
	db *gorm.DB,
	wahaClient *whatsapp.WahaClient,
	messageSender *whatsapp.MessageSender,
	actionHandler *whatsapp.ActionHandler,
) *WhatsAppHandler {
	return &WhatsAppHandler{
		db:            db,
		wahaClient:    wahaClient,
		messageSender: messageSender,
		actionHandler: actionHandler,
	}
}

type CreateSessionRequest struct {
	SessionName string `json:"session_name" binding:"required"`
}

type VerifySessionRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// POST /api/whatsapp/sessions
func (h *WhatsAppHandler) CreateSession(c *gin.Context) {
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Check if session already exists
	var existing models.WhatsAppSession
	if err := h.db.Where("session_name = ?", req.SessionName).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "session already exists"})
		return
	}

	// Create session in Waha Plus
	if err := h.wahaClient.CreateSession(req.SessionName); err != nil {
		log.Printf("Error creating Waha session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session", "details": err.Error()})
		return
	}

	// Save session to database
	session := models.WhatsAppSession{
		ID:          uuid.New().String(),
		SessionName: req.SessionName,
		Status:      "PENDING",
		CreatedAt:   time.Now(),
	}

	if err := h.db.Create(&session).Error; err != nil {
		log.Printf("Error saving session to database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save session"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":            session.ID,
		"session_name":  session.SessionName,
		"status":        session.Status,
		"created_at":    session.CreatedAt,
	})
}

// GET /api/whatsapp/sessions/:id/qr
func (h *WhatsAppHandler) GetSessionQR(c *gin.Context) {
	sessionID := c.Param("id")

	// Get session from database
	var session models.WhatsAppSession
	if err := h.db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	// Get QR code from Waha Plus
	qrCode, err := h.wahaClient.GetSessionQR(session.SessionName)
	if err != nil {
		if strings.Contains(err.Error(), "status 404") {
			log.Printf("Session %s not found in Waha, recreating...", session.SessionName)
			if createErr := h.wahaClient.CreateSession(session.SessionName); createErr != nil {
				log.Printf("Failed to auto-recreate session: %v", createErr)
			} else {
				// Wait for session to be ready
				time.Sleep(2 * time.Second)
				qrCode, err = h.wahaClient.GetSessionQR(session.SessionName)
			}
		}
	}
	if err != nil {
		log.Printf("Error getting QR code: %v", err)
		
		// If Waha says the session is already WORKING, update the database
		wahaSession, checkErr := h.wahaClient.CheckSessionStatus(session.SessionName)
		if checkErr == nil && wahaSession.Status == "WORKING" {
			session.Status = "WORKING"
			session.PhoneNumber = wahaSession.PhoneNumber
			h.db.Save(&session)
			c.JSON(http.StatusOK, gin.H{
				"qr_code": "",
				"status": "WORKING",
				"phone_number": session.PhoneNumber,
				"message": "Session is already connected",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get QR code", "details": err.Error()})
		return
	}

	// Store QR code in database
	session.QRCode = qrCode
	h.db.Save(&session)

	c.JSON(http.StatusOK, gin.H{
		"qr_code": qrCode,
	})
}

type RequestPairingCodeRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
}

// POST /api/whatsapp/sessions/:id/pairing-code
func (h *WhatsAppHandler) RequestPairingCode(c *gin.Context) {
	sessionID := c.Param("id")

	var req RequestPairingCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Get session from database
	var session models.WhatsAppSession
	if err := h.db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	// Request pairing code from Waha Plus
	code, err := h.wahaClient.RequestPairingCode(session.SessionName, req.PhoneNumber)
	if err != nil {
		if strings.Contains(err.Error(), "status 404") {
			log.Printf("Session %s not found in Waha, recreating...", session.SessionName)
			if createErr := h.wahaClient.CreateSession(session.SessionName); createErr != nil {
				log.Printf("Failed to auto-recreate session: %v", createErr)
			} else {
				// Wait for session to be ready
				time.Sleep(2 * time.Second)
				code, err = h.wahaClient.RequestPairingCode(session.SessionName, req.PhoneNumber)
			}
		}
	}
	if err != nil {
		log.Printf("Error requesting pairing code: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get pairing code", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pairing_code": code,
	})
}

// POST /api/whatsapp/sessions/:id/verify
func (h *WhatsAppHandler) VerifySession(c *gin.Context) {
	sessionID := c.Param("id")

	// Get session from database
	var session models.WhatsAppSession
	if err := h.db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	// Check status with Waha Plus
	wahaSession, err := h.wahaClient.CheckSessionStatus(session.SessionName)
	if err != nil {
		log.Printf("Error checking session status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check session", "details": err.Error()})
		return
	}

	// Update database
	if wahaSession.Status == "WORKING" || wahaSession.Connected {
		session.Status = "WORKING" // use WORKING to be consistent
		if wahaSession.PhoneNumber != "" {
			session.PhoneNumber = wahaSession.PhoneNumber
		}
	} else {
		session.Status = wahaSession.Status
	}
	h.db.Save(&session)

	c.JSON(http.StatusOK, gin.H{
		"status":        session.Status,
		"phone_number":  session.PhoneNumber,
		"connected":     session.Status == "WORKING",
	})
}

// DELETE /api/whatsapp/sessions/:id
func (h *WhatsAppHandler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("id")

	// Get session from database
	var session models.WhatsAppSession
	if err := h.db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	// Delete session from Waha Plus
	if err := h.wahaClient.DeleteSession(session.SessionName); err != nil {
		log.Printf("Error deleting Waha session: %v", err)
	}

	// Soft delete from database
	now := time.Now()
	session.DeletedAt = &now
	h.db.Save(&session)

	c.JSON(http.StatusOK, gin.H{"message": "session deleted"})
}

// GET /api/whatsapp/sessions
func (h *WhatsAppHandler) GetSessions(c *gin.Context) {
	var sessions []models.WhatsAppSession
	if err := h.db.Where("deleted_at IS NULL").Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sessions"})
		return
	}

	if sessions == nil {
		sessions = make([]models.WhatsAppSession, 0)
	}

	c.JSON(http.StatusOK, sessions)
}

// POST /api/whatsapp/webhook (from Waha Plus)
func (h *WhatsAppHandler) ProcessWebhook(c *gin.Context) {
	var payload whatsapp.WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Printf("Invalid webhook payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	log.Printf("Received WhatsApp webhook: event=%s session=%s", payload.Event, payload.Session)

	// Handle session.status events
	if payload.Event == "session.status" {
		var statusEvent struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(payload.Data, &statusEvent); err == nil {
			log.Printf("Session status updated for %s: %s", payload.Session, statusEvent.Status)
			
			var session models.WhatsAppSession
			if err := h.db.First(&session, "session_name = ?", payload.Session).Error; err == nil {
				session.Status = statusEvent.Status
				if session.Status == "WORKING" && session.PhoneNumber == "" {
					if wahaSession, err := h.wahaClient.CheckSessionStatus(payload.Session); err == nil {
						if wahaSession.PhoneNumber != "" {
							session.PhoneNumber = wahaSession.PhoneNumber
						}
					}
				}
				h.db.Save(&session)
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Only handle message events
	if payload.Event != "message" {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Parse message event
	var msgEvent whatsapp.MessageEvent
	if err := json.Unmarshal(payload.Data, &msgEvent); err != nil {
		log.Printf("Failed to parse message event: %v", err)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Log the incoming message
	h.logIncomingMessage(payload.Session, msgEvent)

	// Parse message for actions
	if msgEvent.Type == "text" {
		action := whatsapp.ParseMessage(msgEvent.Body)
		if action != nil {
			go h.handleAction(payload.Session, msgEvent.From, action)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *WhatsAppHandler) logIncomingMessage(sessionName string, msg whatsapp.MessageEvent) {
	logEntry := models.WhatsAppLog{
		ID:          uuid.New().String(),
		SessionName: sessionName,
		MessageID:   msg.ID,
		FromPhone:   msg.From,
		ToPhone:     msg.To,
		Body:        msg.Body,
		MessageType: msg.Type,
		Direction:   "inbound",
		Status:      "received",
		CreatedAt:   time.Now(),
	}

	if err := h.db.Create(&logEntry).Error; err != nil {
		log.Printf("Failed to log incoming message: %v", err)
	}
}

func (h *WhatsAppHandler) handleAction(sessionName, fromPhone string, action *whatsapp.ParsedAction) {
	switch action.ActionType {
	case "create_ticket":
		if err := h.actionHandler.HandleCreateTicket(sessionName, fromPhone, action.Content); err != nil {
			log.Printf("Error handling create_ticket: %v", err)
		}
	case "update":
		if err := h.actionHandler.HandleTicketUpdate(sessionName, fromPhone, action.TicketID, action.Content); err != nil {
			log.Printf("Error handling update: %v", err)
		}
	case "close":
		if err := h.actionHandler.HandleTicketClose(sessionName, fromPhone, action.TicketID, action.Content); err != nil {
			log.Printf("Error handling close: %v", err)
		}
	case "reopen":
		if err := h.actionHandler.HandleTicketReopen(sessionName, fromPhone, action.TicketID); err != nil {
			log.Printf("Error handling reopen: %v", err)
		}
	case "status_check":
		if err := h.actionHandler.HandleStatusCheck(sessionName, fromPhone, action.TicketID); err != nil {
			log.Printf("Error handling status_check: %v", err)
		}
	}
}

// POST /api/whatsapp/engineers/:id/phone
func (h *WhatsAppHandler) AddEngineerPhone(c *gin.Context) {
	engineerID := c.Param("id")
	engineerIDUint, err := strconv.ParseUint(engineerID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid engineer id"})
		return
	}

	var req struct {
		PhoneNumber string `json:"phone_number" binding:"required"`
		GroupID     string `json:"group_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Verify engineer exists
	var engineer models.Engineer
	if err := h.db.First(&engineer, engineerIDUint).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "engineer not found"})
		return
	}

	// Create engineer phone record
	engineerPhone := models.EngineerWAPhone{
		ID:          uuid.New().String(),
		EngineerID:  uint(engineerIDUint),
		PhoneNumber: req.PhoneNumber,
		GroupID:     req.GroupID,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	if err := h.db.Create(&engineerPhone).Error; err != nil {
		log.Printf("Error creating engineer phone: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save phone"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":            engineerPhone.ID,
		"engineer_id":   engineerPhone.EngineerID,
		"phone_number":  engineerPhone.PhoneNumber,
		"is_active":     engineerPhone.IsActive,
		"created_at":    engineerPhone.CreatedAt,
	})
}

// GET /api/whatsapp/logs
func (h *WhatsAppHandler) GetLogs(c *gin.Context) {
	sessionName := c.Query("session_name")
	direction := c.Query("direction")
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "50")

	pageNum, _ := strconv.Atoi(page)
	limitNum, _ := strconv.Atoi(limit)
	if pageNum < 1 {
		pageNum = 1
	}
	if limitNum < 1 || limitNum > 100 {
		limitNum = 50
	}

	var logs []models.WhatsAppLog
	query := h.db

	if sessionName != "" {
		query = query.Where("session_name = ?", sessionName)
	}
	if direction != "" {
		query = query.Where("direction = ?", direction)
	}

	if err := query.Order("created_at DESC").
		Offset((pageNum - 1) * limitNum).
		Limit(limitNum).
		Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":   logs,
		"count":  len(logs),
		"page":   pageNum,
		"limit":  limitNum,
	})
}

// GET /api/whatsapp/engineer-phones
func (h *WhatsAppHandler) GetEngineerPhones(c *gin.Context) {
	var phones []models.EngineerWAPhone
	if err := h.db.Preload("Engineer").Where("deleted_at IS NULL").Find(&phones).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch engineer phones"})
		return
	}

	type response struct {
		ID           string    `json:"id"`
		EngineerID   string    `json:"engineerId"`
		PhoneNumber  string    `json:"phoneNumber"`
		GroupID      string    `json:"groupId"`
		IsActive     bool      `json:"isActive"`
		EngineerName string    `json:"engineerName"`
		CreatedAt    time.Time `json:"createdAt"`
	}

	res := make([]response, 0)
	for _, p := range phones {
		res = append(res, response{
			ID:           p.ID,
			EngineerID:   strconv.FormatUint(uint64(p.EngineerID), 10),
			PhoneNumber:  p.PhoneNumber,
			GroupID:      p.GroupID,
			IsActive:     p.IsActive,
			EngineerName: p.Engineer.Name,
			CreatedAt:    p.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, res)
}

// GET /api/whatsapp/webhook/status
func (h *WhatsAppHandler) GetHookStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":        "connected",
		"lastMessageAt": time.Now(),
		"errorLog":      []string{},
	})
}
