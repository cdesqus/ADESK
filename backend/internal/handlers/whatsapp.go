package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"strconv"
	"time"

	"ai-desk/internal/ai"
	"ai-desk/internal/models"
	"ai-desk/internal/whatsapp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WhatsAppHandler struct {
	db             *gorm.DB
	wahaClient     *whatsapp.WahaClient
	messageSender  *whatsapp.MessageSender
	actionHandler  *whatsapp.ActionHandler
	aiClient       *ai.OpenAIClient
}

func NewWhatsAppHandler(
	db *gorm.DB,
	wahaClient *whatsapp.WahaClient,
	messageSender *whatsapp.MessageSender,
	actionHandler *whatsapp.ActionHandler,
	aiClient *ai.OpenAIClient,
) *WhatsAppHandler {
	return &WhatsAppHandler{
		db:            db,
		wahaClient:    wahaClient,
		messageSender: messageSender,
		actionHandler: actionHandler,
		aiClient:      aiClient,
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

	// Determine the actual payload data (WAHA uses either 'data' or 'payload' depending on version)
	var rawPayloadData []byte
	if len(payload.Payload) > 0 {
		rawPayloadData = payload.Payload
	} else if len(payload.Data) > 0 {
		rawPayloadData = payload.Data
	} else {
		rawPayloadData = []byte("{}")
	}

	// Handle session.status events
	if payload.Event == "session.status" {
		// WAHA may send status in different formats depending on version:
		// Format 1: {"status": "WORKING"}
		// Format 2: {"event": "session.status", "session": "name", "payload": {"status": "WORKING"}}
		// Format 3: root level status field
		var rawData map[string]interface{}
		var newStatus string

		if err := json.Unmarshal(rawPayloadData, &rawData); err == nil {
			// Try direct "status" field first
			if s, ok := rawData["status"].(string); ok && s != "" {
				newStatus = s
			}
			// Try nested "payload.status"
			if newStatus == "" {
				if payloadObj, ok := rawData["payload"].(map[string]interface{}); ok {
					if s, ok := payloadObj["status"].(string); ok && s != "" {
						newStatus = s
					}
				}
			}
		}

		if newStatus != "" {
			log.Printf("[WhatsApp Webhook] Session status event for '%s': new_status=%s raw_data=%s",
				payload.Session, newStatus, string(rawPayloadData))

			var session models.WhatsAppSession
			if err := h.db.Where("session_name = ? AND deleted_at IS NULL", payload.Session).First(&session).Error; err == nil {
				session.Status = newStatus
				if newStatus == "WORKING" && session.PhoneNumber == "" {
					if wahaSession, err := h.wahaClient.CheckSessionStatus(payload.Session); err == nil {
						if wahaSession.PhoneNumber != "" {
							session.PhoneNumber = wahaSession.PhoneNumber
						}
						if wahaSession.Me.Name != "" {
							log.Printf("[WhatsApp Webhook] Session '%s' connected as: %s (%s)",
								payload.Session, wahaSession.Me.Name, wahaSession.PhoneNumber)
						}
					}
				}
				h.db.Save(&session)
				log.Printf("[WhatsApp Webhook] Session '%s' DB status updated to: %s", payload.Session, newStatus)
			} else {
				log.Printf("[WhatsApp Webhook] Session '%s' not found in DB for status update", payload.Session)
			}
		} else {
			log.Printf("[WhatsApp Webhook] Could not parse status from session.status event: %s", string(rawPayloadData))
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Only handle message events (WAHA may send "message" or "message.any" depending on config)
	if payload.Event != "message" && payload.Event != "message.any" {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Parse message event
	var msgEvent whatsapp.MessageEvent
	if err := json.Unmarshal(rawPayloadData, &msgEvent); err != nil {
		log.Printf("Failed to parse message event: %v", err)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// CRITICAL: Skip messages sent by the bot itself to prevent infinite loops.
	// When the bot sends a reply, Waha fires a 'message' event back with fromMe=true.
	// Without this check, the bot would process its own replies and create duplicate tickets.
	var rawMsg map[string]interface{}
	_ = json.Unmarshal(rawPayloadData, &rawMsg)
	if fromMe, ok := rawMsg["fromMe"].(bool); ok && fromMe {
		log.Printf("[WhatsApp] Skipping own message (fromMe=true) in session=%s", payload.Session)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Detect if it's a group
	isGroup := strings.HasSuffix(msgEvent.From, "@g.us")
	msgEvent.IsGroup = isGroup

	// In groups, the actual sender is in the Participant field
	senderPhone := msgEvent.From
	if isGroup && msgEvent.Participant != "" {
		senderPhone = msgEvent.Participant
	}

	// Determine if the message is directed to the bot
	isDirectedToBot := !isGroup // Private messages are always directed to the bot

	// For group messages, check if the bot is mentioned
	if isGroup {
		var session models.WhatsAppSession
		if err := h.db.First(&session, "session_name = ?", payload.Session).Error; err == nil {
			botNum := normalizeWhatsAppID(session.PhoneNumber)
			botLid := ""
			if wahaSession, err := h.wahaClient.CheckSessionStatus(payload.Session); err == nil {
				if botNum == "" {
					botNum = normalizeWhatsAppID(wahaSession.PhoneNumber)
				}
				botLid = normalizeWhatsAppID(wahaSession.Me.ID)
			}

			log.Printf("[WhatsApp] bot mention candidates session=%s botNum=%s botLid=%s", payload.Session, botNum, botLid)

			if botNum != "" && containsMention(msgEvent.Body, botNum) {
				isDirectedToBot = true
			}
			if !isDirectedToBot && botLid != "" && containsMention(msgEvent.Body, botLid) {
				isDirectedToBot = true
			}
		}

		// Fallback for textual mentions like @helpdesk or any leading numeric mention in the group
		if !isDirectedToBot {
			reBotMention := regexp.MustCompile(`(?i)@helpdesk`)
			if reBotMention.MatchString(msgEvent.Body) {
				isDirectedToBot = true
			}
		}

		if !isDirectedToBot && looksLikeBotMention(msgEvent.Body) {
			log.Printf("[WhatsApp] numeric group mention detected, directing to bot: %s", msgEvent.Body)
			isDirectedToBot = true
		}
	}

	if msgEvent.ID != "" {
		inserted, err := h.insertInboundMessageIfNew(payload.Session, msgEvent)
		if err != nil {
			log.Printf("[WhatsApp] failed to insert inbound message log: %v", err)
		} else if !inserted {
			log.Printf("[WhatsApp] duplicate inbound message skipped session=%s messageID=%s body=%q", payload.Session, msgEvent.ID, msgEvent.Body)
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
			return
		}
	} else {
		// If there is no message ID, fall back to logging normally.
		h.logIncomingMessage(payload.Session, msgEvent)
	}

	log.Printf("[WhatsApp] message metadata session=%s from=%s type=%s isGroup=%t directedToBot=%t body=%q", payload.Session, msgEvent.From, msgEvent.Type, isGroup, isDirectedToBot, msgEvent.Body)

	// Parse message for actions when the payload contains a body
	if strings.TrimSpace(msgEvent.Body) != "" {
		var action *whatsapp.ParsedAction
		var aiReply string

		log.Printf("[WhatsApp] parsing action from body session=%s from=%s isDirectedToBot=%t", payload.Session, senderPhone, isDirectedToBot)

		// First try OpenAI if it's directed to bot
		if isDirectedToBot && h.aiClient != nil {
			waResp, err := h.aiClient.ParseWhatsAppMessage(msgEvent.Body)
			if err == nil && waResp != nil && isValidWhatsAppAction(waResp.ActionType) {
				action = &whatsapp.ParsedAction{
					ActionType: waResp.ActionType,
					TicketID:   waResp.TicketID,
					Content:    waResp.Content,
				}
				aiReply = waResp.NaturalReply
				log.Printf("[WhatsApp] OpenAI parsed action=%s ticketID=%s", waResp.ActionType, waResp.TicketID)
			} else {
				log.Printf("[WhatsApp] OpenAI parsing invalid or failed action=%q err=%v, falling back to regex", waResp.ActionType, err)
			}
		} else if isDirectedToBot && h.aiClient == nil {
			log.Printf("[WhatsApp] AI client is nil (OPENAI_API_KEY not set?), using regex parser only")
		}

		// Fallback to Regex parser
		if action == nil {
			action = whatsapp.ParseMessage(msgEvent.Body, isDirectedToBot)
		}

		// Check for quoted message (swipe reply)
		var rawData map[string]interface{}
		_ = json.Unmarshal(rawPayloadData, &rawData)
		
		quotedBody := ""
		if hasQuoted, ok := rawData["hasQuotedMsg"].(bool); ok && hasQuoted {
			if _data, ok := rawData["_data"].(map[string]interface{}); ok {
				if quotedMsg, ok := _data["quotedMsg"].(map[string]interface{}); ok {
					if qBody, ok := quotedMsg["body"].(string); ok {
						quotedBody = qBody
					}
				}
			}
		}

		// If it's directed to bot (either private or mentioned) and has a quoted message with a ticket ID
		if isDirectedToBot && quotedBody != "" {
			re := regexp.MustCompile(`(?i)(\d{4}-\d{2}-\d{3,})`)
			matches := re.FindStringSubmatch(quotedBody)
			if len(matches) > 1 {
				ticketID := matches[1]
				// Clean the @mention from the body for the update content
				reMention := regexp.MustCompile(`@\d+\s*`)
				cleanContent := strings.TrimSpace(reMention.ReplaceAllString(msgEvent.Body, ""))
				
				// Overwrite action to be an update
				action = &whatsapp.ParsedAction{
					ActionType: "update",
					TicketID:   ticketID,
					Content:    cleanContent,
				}
				// Clear AI reply since this is a hard-override
				aiReply = ""
			}
		}

		// CRITICAL FALLBACK: If the message is directed to the bot but both AI and regex
		// failed to produce an action, generate a fallback response so the customer
		// is NEVER left without a reply.
		if action == nil && isDirectedToBot && strings.TrimSpace(msgEvent.Body) != "" {
			log.Printf("[WhatsApp] both AI and regex failed for directed message, using fallback")
			fallback := ai.GenerateFallbackResponse(msgEvent.Body)
			action = &whatsapp.ParsedAction{
				ActionType: fallback.ActionType,
				TicketID:   fallback.TicketID,
				Content:    fallback.Content,
			}
			aiReply = fallback.NaturalReply
		}

		if action != nil {
			log.Printf("[WhatsApp] action resolved session=%s action=%s ticketID=%s content=%q", payload.Session, action.ActionType, action.TicketID, action.Content)
			// Reply to the chat where the message originated (group or private)
			replyTo := msgEvent.From
			go h.handleAction(payload.Session, senderPhone, replyTo, isGroup, action, aiReply)
		} else {
			log.Printf("[WhatsApp] no action produced for directed message session=%s from=%s body=%q", payload.Session, senderPhone, msgEvent.Body)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *WhatsAppHandler) insertInboundMessageIfNew(sessionName string, msg whatsapp.MessageEvent) (bool, error) {
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

	err := h.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&logEntry).Error
	if err != nil {
		return false, err
	}

	// If the row was not inserted, the returned ID will still be populated in Go, so we need
	// to detect duplicates based on the inserted_at timestamp behavior.
	var existing models.WhatsAppLog
	if err := h.db.Where("session_name = ? AND message_id = ? AND direction = ?", sessionName, msg.ID, "inbound").First(&existing).Error; err != nil {
		return false, err
	}

	return existing.ID == logEntry.ID, nil
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

	if err := h.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&logEntry).Error; err != nil {
		log.Printf("Failed to log incoming message: %v", err)
	}
}

func normalizeWhatsAppID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if strings.Contains(id, "@") {
		id = strings.Split(id, "@")[0]
	}
	if strings.Contains(id, ":") {
		id = strings.Split(id, ":")[0]
	}
	return id
}

func containsMention(body, id string) bool {
	if body == "" || id == "" {
		return false
	}

	bodyLower := strings.ToLower(body)
	idLower := strings.ToLower(id)
	if strings.Contains(bodyLower, "@"+idLower) {
		return true
	}

	reMention := regexp.MustCompile(`@([0-9]{5,})`)
	for _, match := range reMention.FindAllStringSubmatch(bodyLower, -1) {
		if len(match) > 1 && match[1] == idLower {
			return true
		}
	}

	return false
}

func looksLikeBotMention(body string) bool {
	reMention := regexp.MustCompile(`@([0-9]{5,})`)
	matches := reMention.FindAllStringSubmatch(body, -1)
	return len(matches) > 0
}

func isValidWhatsAppAction(actionType string) bool {
	switch actionType {
	case "create_ticket", "update", "close", "reopen", "status_check":
		return true
	default:
		return false
	}
}

func (h *WhatsAppHandler) handleAction(sessionName, senderPhone, replyTo string, isGroup bool, action *whatsapp.ParsedAction, aiReply string) {
	switch action.ActionType {
	case "create_ticket":
		if err := h.actionHandler.HandleCreateTicket(sessionName, senderPhone, replyTo, isGroup, action.Content, aiReply); err != nil {
			log.Printf("Error handling create_ticket: %v", err)
		}
	case "update":
		if err := h.actionHandler.HandleTicketUpdate(sessionName, senderPhone, replyTo, isGroup, action.TicketID, action.Content); err != nil {
			log.Printf("Error handling update: %v", err)
		}
	case "close":
		if err := h.actionHandler.HandleTicketClose(sessionName, senderPhone, replyTo, isGroup, action.TicketID, action.Content); err != nil {
			log.Printf("Error handling close: %v", err)
		}
	case "reopen":
		if err := h.actionHandler.HandleTicketReopen(sessionName, senderPhone, replyTo, isGroup, action.TicketID); err != nil {
			log.Printf("Error handling reopen: %v", err)
		}
	case "status_check":
		if err := h.actionHandler.HandleStatusCheck(sessionName, senderPhone, replyTo, isGroup, action.TicketID); err != nil {
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

// POST /api/whatsapp/test-message
func (h *WhatsAppHandler) TestMessage(c *gin.Context) {
	var req struct {
		SessionID   string `json:"sessionId"`
		PhoneNumber string `json:"phoneNumber"`
		Message     string `json:"message"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request"})
		return
	}

	// Verify session exists
	var session models.WhatsAppSession
	if err := h.db.First(&session, "id = ?", req.SessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "session not found"})
		return
	}

	// Format phone number / chat ID
	chatId := req.PhoneNumber

	// If it doesn't end with @g.us (not a group ID), format it as a standard phone number
	if !strings.HasSuffix(chatId, "@g.us") {
		var number strings.Builder
		for _, r := range req.PhoneNumber {
			if r >= '0' && r <= '9' {
				number.WriteRune(r)
			}
		}
		
		if number.Len() == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid phone number"})
			return
		}
		chatId = number.String() + "@c.us"
	}

	// Send message
	_, err := h.wahaClient.SendMessage(session.SessionName, chatId, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": fmt.Sprintf("failed to send message: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "message sent successfully"})
}

// PUT /api/whatsapp/engineer-phones/:id
func (h *WhatsAppHandler) UpdateEngineerPhone(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		PhoneNumber string `json:"phoneNumber"`
		GroupID     string `json:"groupId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var phone models.EngineerWAPhone
	if err := h.db.Preload("Engineer").Where("id = ?", id).First(&phone).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "phone not found"})
		return
	}

	if req.PhoneNumber != "" {
		phone.PhoneNumber = req.PhoneNumber
	}
	phone.GroupID = req.GroupID

	if err := h.db.Save(&phone).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update phone"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           phone.ID,
		"engineerId":   strconv.FormatUint(uint64(phone.EngineerID), 10),
		"phoneNumber":  phone.PhoneNumber,
		"groupId":      phone.GroupID,
		"isActive":     phone.IsActive,
		"engineerName": phone.Engineer.Name,
		"createdAt":    phone.CreatedAt,
	})
}

// DELETE /api/whatsapp/engineer-phones/:id
func (h *WhatsAppHandler) DeleteEngineerPhone(c *gin.Context) {
	id := c.Param("id")
	
	if err := h.db.Where("id = ?", id).Delete(&models.EngineerWAPhone{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete phone"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}
