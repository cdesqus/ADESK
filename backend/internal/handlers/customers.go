package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"ai-desk/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CustomerHandler struct {
	db *gorm.DB
}

func NewCustomerHandler(db *gorm.DB) *CustomerHandler {
	return &CustomerHandler{db: db}
}

// CreateCustomer creates a new customer
// POST /api/customers
func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var customer models.Customer

	if err := c.ShouldBindJSON(&customer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Validate required fields
	if customer.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	if err := h.db.Create(&customer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create customer", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, customer)
}

// GetCustomers retrieves all customers with pagination
// GET /api/customers?page=1&limit=10
func (h *CustomerHandler) GetCustomers(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", c.DefaultQuery("pageSize", "10"))
	isActive := c.Query("is_active")

	pageNum, err := strconv.Atoi(page)
	if err != nil || pageNum < 1 {
		pageNum = 1
	}

	limitNum, err := strconv.Atoi(limit)
	if err != nil || limitNum < 1 || limitNum > 100 {
		limitNum = 10
	}

	var customers []models.Customer
	query := h.db.Offset((pageNum - 1) * limitNum).Limit(limitNum).Preload("WAGroups")

	if isActive != "" {
		query = query.Where("is_active = ?", isActive == "true")
	}

	if err := query.Find(&customers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch customers"})
		return
	}

	var total int64
	h.db.Model(&models.Customer{}).Count(&total)

	totalPages := int(total) / limitNum
	if int(total)%limitNum > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       customers,
		"total":      total,
		"page":       pageNum,
		"pageSize":   limitNum,
		"totalPages": totalPages,
	})
}

// GetCustomerByID retrieves a customer by ID
// GET /api/customers/:id
func (h *CustomerHandler) GetCustomerByID(c *gin.Context) {
	id := c.Param("id")

	customerID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer ID"})
		return
	}

	var customer models.Customer
	if err := h.db.Preload("Engineers").Preload("Tickets").Preload("WAGroups").First(&customer, customerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch customer"})
		return
	}

	c.JSON(http.StatusOK, customer)
}

// UpdateCustomer updates an existing customer
// PUT /api/customers/:id
func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	id := c.Param("id")

	customerID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer ID"})
		return
	}

	var updateReq models.Customer
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Check if customer exists
	var customer models.Customer
	if err := h.db.First(&customer, customerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch customer"})
		return
	}

	// Update only provided fields
	if err := h.db.Model(&customer).Updates(updateReq).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update customer", "details": err.Error()})
		return
	}

	// Update WAGroups association if provided
	// We check if it's sent in the request (even if empty, they might want to clear it)
	// But in JSON, omitted means nil, while empty array means []
	if updateReq.WAGroups != nil {
		// Gorm Replace will clear old ones and insert new ones
		if err := h.db.Model(&customer).Association("WAGroups").Replace(updateReq.WAGroups); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update WA groups", "details": err.Error()})
			return
		}
	}

	// Reload with associations
	h.db.Preload("WAGroups").First(&customer, customerID)

	c.JSON(http.StatusOK, customer)
}

// DeleteCustomer soft deletes a customer
// DELETE /api/customers/:id
func (h *CustomerHandler) DeleteCustomer(c *gin.Context) {
	id := c.Param("id")

	customerID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer ID"})
		return
	}

	if err := h.db.Delete(&models.Customer{}, customerID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete customer"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
