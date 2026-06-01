package middleware

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error      string `json:"error"`
	StatusCode int    `json:"status_code"`
	Message    string `json:"message,omitempty"`
}

// HandleError provides centralized error handling
func HandleError(c *gin.Context, statusCode int, err error) {
	log.Printf("Error: %v", err)

	message := "An error occurred"
	if err != nil {
		message = err.Error()
	}

	c.JSON(statusCode, ErrorResponse{
		Error:      http.StatusText(statusCode),
		StatusCode: statusCode,
		Message:    message,
	})
}

// ValidateID validates and parses an ID from URL parameter
func ValidateID(idStr string) (uint, error) {
	var id uint
	n, err := fmt.Sscanf(idStr, "%d", &id)
	if err != nil || n != 1 {
		return 0, fmt.Errorf("invalid ID format")
	}
	if id == 0 {
		return 0, fmt.Errorf("ID cannot be zero")
	}
	return id, nil
}
