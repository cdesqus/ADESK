package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"ai-desk/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EngineerHandler struct {
	db *gorm.DB
}

func NewEngineerHandler(db *gorm.DB) *EngineerHandler {
	return &EngineerHandler{db: db}
}

// CreateEngineer creates a new engineer
// POST /api/engineers
func (h *EngineerHandler) CreateEngineer(c *gin.Context) {
	var engineer models.Engineer

	if err := c.ShouldBindJSON(&engineer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Validate required fields
	if engineer.Name == "" || engineer.Email == "" || engineer.CustomerID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, email, and customer_id are required"})
		return
	}

	// Verify customer exists
	var customer models.Customer
	if err := h.db.First(&customer, engineer.CustomerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "customer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if err := h.db.Create(&engineer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create engineer", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, engineer)
}

// GetEngineers retrieves all engineers with optional filters
// GET /api/engineers?customer_id=1&is_active=true&page=1&limit=10
func (h *EngineerHandler) GetEngineers(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")
	customerID := c.Query("customer_id")
	isActive := c.Query("is_active")

	pageNum, err := strconv.Atoi(page)
	if err != nil || pageNum < 1 {
		pageNum = 1
	}

	limitNum, err := strconv.Atoi(limit)
	if err != nil || limitNum < 1 || limitNum > 100 {
		limitNum = 10
	}

	var engineers []models.Engineer
	query := h.db.Offset((pageNum - 1) * limitNum).Limit(limitNum)

	if customerID != "" {
		query = query.Where("customer_id = ?", customerID)
	}

	if isActive != "" {
		query = query.Where("is_active = ?", isActive == "true")
	}

	if err := query.Find(&engineers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch engineers"})
		return
	}

	var total int64
	countQuery := h.db.Model(&models.Engineer{})
	if customerID != "" {
		countQuery = countQuery.Where("customer_id = ?", customerID)
	}
	if isActive != "" {
		countQuery = countQuery.Where("is_active = ?", isActive == "true")
	}
	countQuery.Count(&total)

	c.JSON(http.StatusOK, gin.H{
		"data":  engineers,
		"total": total,
		"page":  pageNum,
		"limit": limitNum,
	})
}

// GetEngineerByID retrieves an engineer by ID
// GET /api/engineers/:id
func (h *EngineerHandler) GetEngineerByID(c *gin.Context) {
	id := c.Param("id")

	engineerID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid engineer ID"})
		return
	}

	var engineer models.Engineer
	if err := h.db.Preload("Tickets").Preload("Updates").First(&engineer, engineerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "engineer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch engineer"})
		return
	}

	c.JSON(http.StatusOK, engineer)
}

// UpdateEngineer updates an existing engineer
// PUT /api/engineers/:id
func (h *EngineerHandler) UpdateEngineer(c *gin.Context) {
	id := c.Param("id")

	engineerID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid engineer ID"})
		return
	}

	var updateReq models.Engineer
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Check if engineer exists
	var engineer models.Engineer
	if err := h.db.First(&engineer, engineerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "engineer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch engineer"})
		return
	}

	// Update only provided fields
	if err := h.db.Model(&engineer).Updates(updateReq).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update engineer", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, engineer)
}

// DeleteEngineer soft deletes an engineer
// DELETE /api/engineers/:id
func (h *EngineerHandler) DeleteEngineer(c *gin.Context) {
	id := c.Param("id")

	engineerID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid engineer ID"})
		return
	}

	if err := h.db.Delete(&models.Engineer{}, engineerID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete engineer"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
