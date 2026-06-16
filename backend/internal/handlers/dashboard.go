package handlers

import (
	"net/http"

	"ai-desk/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DashboardHandler struct {
	db *gorm.DB
}

func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

func (h *DashboardHandler) GetSummary(c *gin.Context) {
	var totalTickets int64
	var openTickets int64
	var resolvedTickets int64
	var inProgressTickets int64
	var closedTickets int64
	var waitingCustomerTickets int64

	h.db.Model(&models.Ticket{}).Count(&totalTickets)
	h.db.Model(&models.Ticket{}).Where("status = ?", "open").Count(&openTickets)
	h.db.Model(&models.Ticket{}).Where("status = ?", "resolved").Count(&resolvedTickets)
	h.db.Model(&models.Ticket{}).Where("status = ?", "in_progress").Count(&inProgressTickets)
	h.db.Model(&models.Ticket{}).Where("status = ?", "closed").Count(&closedTickets)
	h.db.Model(&models.Ticket{}).Where("status = ?", "waiting_customer").Count(&waitingCustomerTickets)

	var recentTickets []models.Ticket
	h.db.Order("created_at DESC").Limit(5).Preload("Customer").Find(&recentTickets)

	c.JSON(http.StatusOK, gin.H{
		"stats": gin.H{
			"total":            totalTickets,
			"open":             openTickets,
			"resolved":         resolvedTickets,
			"in_progress":      inProgressTickets,
			"closed":           closedTickets,
			"waiting_customer": waitingCustomerTickets,
		},
		"recent_tickets": recentTickets,
	})
}
