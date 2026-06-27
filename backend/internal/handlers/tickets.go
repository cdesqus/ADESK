package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"ai-desk/internal/email"
	"ai-desk/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type TicketHandler struct {
	db         *gorm.DB
	smtpClient *email.SMTPClient
}

func NewTicketHandler(db *gorm.DB, smtpClient *email.SMTPClient) *TicketHandler {
	return &TicketHandler{db: db, smtpClient: smtpClient}
}

// CreateTicket creates a new ticket
// POST /api/tickets
func (h *TicketHandler) CreateTicket(c *gin.Context) {
	var ticket models.Ticket

	if err := c.ShouldBindJSON(&ticket); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Validate required fields
	if ticket.Title == "" || ticket.CustomerID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title and customer_id are required"})
		return
	}

	// Verify customer exists
	var customer models.Customer
	if err := h.db.First(&customer, ticket.CustomerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "customer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// If engineer_id provided, verify engineer exists and belongs to same customer
	if ticket.EngineerID != nil && *ticket.EngineerID > 0 {
		var engineer models.Engineer
		if err := h.db.First(&engineer, *ticket.EngineerID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "engineer not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		if engineer.CustomerID != nil && *engineer.CustomerID != ticket.CustomerID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "engineer does not belong to this customer"})
			return
		}
	}

	// Set defaults
	if ticket.Status == "" {
		ticket.Status = "OPEN"
	}
	if ticket.Priority == "" {
		ticket.Priority = "MEDIUM"
	}

	if err := h.db.Create(&ticket).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ticket", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ticket)
}

// GetTickets retrieves tickets with filters
// GET /api/tickets?customer_id=1&status=OPEN&priority=HIGH&engineer_id=1&page=1&limit=10
func (h *TicketHandler) GetTickets(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", c.DefaultQuery("pageSize", "10"))
	customerID := c.DefaultQuery("customer_id", c.Query("customerId"))
	status := c.Query("status")
	priority := c.Query("priority")
	engineerID := c.DefaultQuery("engineer_id", c.Query("engineerId"))
	search := c.Query("search")

	pageNum, err := strconv.Atoi(page)
	if err != nil || pageNum < 1 {
		pageNum = 1
	}

	limitNum, err := strconv.Atoi(limit)
	if err != nil || limitNum < 1 || limitNum > 100 {
		limitNum = 10
	}

	var tickets []models.Ticket
	query := h.db.Offset((pageNum-1)*limitNum).Limit(limitNum).Order("id DESC").Preload("Engineer").Preload("Updates").Preload("Customer")

	if customerID != "" {
		query = query.Where("customer_id = ?", customerID)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if priority != "" {
		query = query.Where("priority = ?", priority)
	}

	if engineerID != "" {
		query = query.Where("engineer_id = ?", engineerID)
	}

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("title ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}

	if err := query.Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tickets"})
		return
	}

	var total int64
	countQuery := h.db.Model(&models.Ticket{})
	if customerID != "" {
		countQuery = countQuery.Where("customer_id = ?", customerID)
	}
	if status != "" {
		countQuery = countQuery.Where("status = ?", status)
	}
	if priority != "" {
		countQuery = countQuery.Where("priority = ?", priority)
	}
	if engineerID != "" {
		countQuery = countQuery.Where("engineer_id = ?", engineerID)
	}
	if search != "" {
		searchPattern := "%" + search + "%"
		countQuery = countQuery.Where("title ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}
	countQuery.Count(&total)

	totalPages := int(total) / limitNum
	if int(total)%limitNum > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       tickets,
		"total":      total,
		"page":       pageNum,
		"pageSize":   limitNum,
		"totalPages": totalPages,
	})
}

// GetTicketByID retrieves a ticket by ID
// GET /api/tickets/:id
func (h *TicketHandler) GetTicketByID(c *gin.Context) {
	id := c.Param("id")

	ticketID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket ID"})
		return
	}

	var ticket models.Ticket
	if err := h.db.Preload("Engineer").Preload("Updates").Preload("Customer").First(&ticket, ticketID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch ticket"})
		return
	}

	c.JSON(http.StatusOK, ticket)
}

// UpdateTicket updates a ticket
// PUT /api/tickets/:id
func (h *TicketHandler) UpdateTicket(c *gin.Context) {
	id := c.Param("id")

	ticketID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket ID"})
		return
	}

	var updateReq struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Priority    string `json:"priority"`
		Category    string `json:"category"`
		EngineerID  *uint  `json:"engineer_id"`
	}

	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Check if ticket exists
	var ticket models.Ticket
	if err := h.db.First(&ticket, ticketID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch ticket"})
		return
	}

	// If engineer_id provided, verify engineer exists and belongs to same customer
	if updateReq.EngineerID != nil && *updateReq.EngineerID > 0 {
		var engineer models.Engineer
		if err := h.db.First(&engineer, *updateReq.EngineerID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "engineer not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		if engineer.CustomerID != nil && *engineer.CustomerID != ticket.CustomerID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "engineer does not belong to this customer"})
			return
		}
	}

	// Handle status change to RESOLVED
	updates := map[string]interface{}{}
	if updateReq.Title != "" {
		updates["title"] = updateReq.Title
	}
	if updateReq.Description != "" {
		updates["description"] = updateReq.Description
	}
	if updateReq.Status != "" {
		updates["status"] = updateReq.Status
		if updateReq.Status == "RESOLVED" {
			now := time.Now()
			updates["resolved_at"] = now
		}
	}
	if updateReq.Priority != "" {
		updates["priority"] = updateReq.Priority
	}
	if updateReq.Category != "" {
		updates["category"] = updateReq.Category
	}
	if updateReq.EngineerID != nil {
		updates["engineer_id"] = updateReq.EngineerID
	}

	if err := h.db.Model(&ticket).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update ticket", "details": err.Error()})
		return
	}

	// Send closing email if status changed to CLOSED and email address exists
	if updateReq.Status == "CLOSED" && ticket.Status != "CLOSED" && ticket.EmailFrom != "" {
		go func(emailFrom, ticketNum string) {
			subject := fmt.Sprintf("[AI-DESK] Tiket %s Telah Ditutup", ticketNum)
			htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; color: #333333; line-height: 1.6; font-size: 14px; padding: 10px;">
	<div>
		<p>Yth. Pelanggan,</p>
		<p>Tiket dukungan Anda dengan ID <strong>%s</strong> telah dinyatakan selesai dan ditutup.</p>
		<p>Terima kasih telah menghubungi layanan kami. Jika Anda masih memerlukan bantuan lebih lanjut, silakan buat tiket baru.</p>
		<br>
		<p>Salam,<br>Tim Support</p>
	</div>
</body>
</html>`, ticketNum)
			if err := h.smtpClient.SendHTMLEmail(emailFrom, "", subject, htmlBody); err != nil {
				log.Printf("Failed to send ticket closing email to %s: %v", emailFrom, err)
			}
		}(ticket.EmailFrom, ticket.TicketNumber)
	}

	// Reload to get updated data
	h.db.Preload("Engineer").Preload("Updates").First(&ticket, ticketID)

	c.JSON(http.StatusOK, ticket)
}

// DeleteTicket soft deletes a ticket
// DELETE /api/tickets/:id
func (h *TicketHandler) DeleteTicket(c *gin.Context) {
	id := c.Param("id")

	ticketID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket ID"})
		return
	}

	if err := h.db.Delete(&models.Ticket{}, ticketID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete ticket"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// BulkTicketAction performs bulk actions on multiple tickets
// POST /api/tickets/bulk
func (h *TicketHandler) BulkTicketAction(c *gin.Context) {
	var req struct {
		TicketIDs []uint `json:"ticket_ids" binding:"required"`
		Action    string `json:"action" binding:"required"` // "delete", "update_status", "update_priority"
		Status    string `json:"status"`
		Priority  string `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	if len(req.TicketIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no tickets selected"})
		return
	}

	switch req.Action {
	case "delete":
		if err := h.db.Where("id IN ?", req.TicketIDs).Delete(&models.Ticket{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete tickets", "details": err.Error()})
			return
		}
	case "update_status":
		if req.Status == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status is required for update_status action"})
			return
		}
		updates := map[string]interface{}{"status": req.Status}
		if req.Status == "RESOLVED" {
			updates["resolved_at"] = time.Now()
		}
		if err := h.db.Model(&models.Ticket{}).Where("id IN ?", req.TicketIDs).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status", "details": err.Error()})
			return
		}
	case "update_priority":
		if req.Priority == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "priority is required for update_priority action"})
			return
		}
		if err := h.db.Model(&models.Ticket{}).Where("id IN ?", req.TicketIDs).Update("priority", req.Priority).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update priority", "details": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "bulk action completed successfully"})
}

// BulkExportExcel exports selected tickets to an Excel file
// POST /api/tickets/bulk-export
func (h *TicketHandler) BulkExportExcel(c *gin.Context) {
	var req struct {
		TicketIDs []uint `json:"ticket_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	if len(req.TicketIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no tickets selected"})
		return
	}

	var tickets []models.Ticket
	if err := h.db.Preload("Customer").Where("id IN ?", req.TicketIDs).Order("created_at DESC").Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tickets"})
		return
	}

	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	sheet := "Sheet1"
	headers := []string{"Ticket Number", "Title", "Customer", "Status", "Priority", "Created At"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	for i, ticket := range tickets {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), ticket.TicketNumber)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), ticket.Title)
		customerName := ""
		if ticket.Customer.ID != 0 {
			customerName = ticket.Customer.Name
		}
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), customerName)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), ticket.Status)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), ticket.Priority)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), ticket.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate excel file"})
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", `attachment; filename="tickets_export.xlsx"`)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}
