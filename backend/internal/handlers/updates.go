package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"ai-desk/internal/email"
	"ai-desk/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"fmt"
	"log"
)

type UpdateHandler struct {
	db         *gorm.DB
	smtpClient *email.SMTPClient
}

func NewUpdateHandler(db *gorm.DB, smtpClient *email.SMTPClient) *UpdateHandler {
	return &UpdateHandler{db: db, smtpClient: smtpClient}
}

// CreateUpdate adds an update/comment to a ticket
// POST /api/tickets/:ticket_id/updates
func (h *UpdateHandler) CreateUpdate(c *gin.Context) {
	ticketID := c.Param("id")

	tid, err := strconv.ParseUint(ticketID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket ID"})
		return
	}

	var req struct {
		models.Update
		SendEmail bool `json:"send_email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	update := req.Update

	// Validate required fields
	if update.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}

	// Verify ticket exists
	var ticket models.Ticket
	if err := h.db.First(&ticket, tid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// If engineer_id provided, verify engineer exists
	if update.EngineerID != nil && *update.EngineerID > 0 {
		var engineer models.Engineer
		if err := h.db.First(&engineer, *update.EngineerID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "engineer not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}

	// Set ticket ID and defaults
	update.TicketID = uint(tid)
	if update.ActionType == "" {
		update.ActionType = "COMMENT"
	}

	if err := h.db.Create(&update).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create update", "details": err.Error()})
		return
	}

	// Send email to customer if requested
	if req.SendEmail && ticket.EmailFrom != "" {
		go func(emailFrom, ticketNum, msg string) {
			subject := fmt.Sprintf("[AI-DESK] Re: [%s] Update Tiket", ticketNum)
			htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; color: #333333; line-height: 1.6; font-size: 14px; padding: 10px;">
	<div>
		<p>Halo,</p>
		<p>%s</p>
		<br>
		<p>Salam,<br>Tim Support</p>
	</div>
</body>
</html>`, msg)
			if err := h.smtpClient.SendHTMLEmail(emailFrom, "", subject, htmlBody); err != nil {
				log.Printf("Failed to send ticket reply email to %s: %v", emailFrom, err)
			}
		}(ticket.EmailFrom, ticket.TicketNumber, update.Message)
	}

	c.JSON(http.StatusCreated, update)
}

// GetTicketUpdates retrieves all updates for a ticket
// GET /api/tickets/:ticket_id/updates?page=1&limit=20
func (h *UpdateHandler) GetTicketUpdates(c *gin.Context) {
	ticketID := c.Param("id")
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "20")

	tid, err := strconv.ParseUint(ticketID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket ID"})
		return
	}

	pageNum, err := strconv.Atoi(page)
	if err != nil || pageNum < 1 {
		pageNum = 1
	}

	limitNum, err := strconv.Atoi(limit)
	if err != nil || limitNum < 1 || limitNum > 100 {
		limitNum = 20
	}

	// Verify ticket exists
	var ticket models.Ticket
	if err := h.db.First(&ticket, tid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	var updates []models.Update
	if err := h.db.Where("ticket_id = ?", tid).
		Preload("Engineer").
		Order("created_at DESC").
		Offset((pageNum - 1) * limitNum).
		Limit(limitNum).
		Find(&updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch updates"})
		return
	}

	var total int64
	h.db.Model(&models.Update{}).Where("ticket_id = ?", tid).Count(&total)

	c.JSON(http.StatusOK, gin.H{
		"data":  updates,
		"total": total,
		"page":  pageNum,
		"limit": limitNum,
	})
}

// DeleteUpdate soft deletes an update
// DELETE /api/updates/:id
func (h *UpdateHandler) DeleteUpdate(c *gin.Context) {
	id := c.Param("id")

	updateID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid update ID"})
		return
	}

	// Check if update exists
	var update models.Update
	if err := h.db.First(&update, updateID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "update not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if err := h.db.Delete(&update).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete update"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
