package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ai-desk/internal/models"
	"ai-desk/internal/reports"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReportHandler struct {
	db        *gorm.DB
	generator *reports.ReportGenerator
	repo      *reports.ReportRepository
	scheduler *reports.ReportScheduler
	mailer    *reports.ReportMailer
}

func NewReportHandler(db *gorm.DB, generator *reports.ReportGenerator,
	repo *reports.ReportRepository, scheduler *reports.ReportScheduler, mailer *reports.ReportMailer) *ReportHandler {
	return &ReportHandler{
		db:        db,
		generator: generator,
		repo:      repo,
		scheduler: scheduler,
		mailer:    mailer,
	}
}

type GenerateReportRequest struct {
	CustomerID uint `json:"customer_id" binding:"required"`
	Month      int  `json:"month" binding:"required,min=1,max=12"`
	Year       int  `json:"year" binding:"required,min=2020"`
}

type ReportListResponse struct {
	ID           string     `json:"id"`
	CustomerID   uint       `json:"customer_id"`
	CustomerName string     `json:"customer_name"`
	Month        int        `json:"month"`
	Year         int        `json:"year"`
	GeneratedAt  time.Time  `json:"generated_at"`
	SentAt       *time.Time `json:"sent_at"`
	SentToEmails []string   `json:"sent_to_emails"`
}

// POST /api/reports/generate - Generate report on-demand
func (rh *ReportHandler) GenerateReport(c *gin.Context) {
	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	reportData, err := rh.scheduler.GenerateReportOnDemand(ctx, req.CustomerID, req.Month, req.Year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to generate report: %v", err)})
		return
	}

	c.JSON(http.StatusOK, reportData)
}

// GET /api/reports - List all reports with pagination
func (rh *ReportHandler) ListReports(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 1
	}
	if offset < 0 {
		offset = 0
	}

	var customerID *uint
	if cid := c.Query("customer_id"); cid != "" {
		if id, err := strconv.ParseUint(cid, 10, 32); err == nil {
			id32 := uint(id)
			customerID = &id32
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	reportsList, total, err := rh.repo.ListReports(ctx, customerID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to list reports: %v", err)})
		return
	}

	response := make([]ReportListResponse, len(reportsList))
	for i, r := range reportsList {
		response[i] = ReportListResponse{
			ID:           r.ID,
			CustomerID:   r.CustomerID,
			CustomerName: r.Customer.Name,
			Month:        r.Month,
			Year:         r.Year,
			GeneratedAt:  r.GeneratedAt,
			SentAt:       r.SentAt,
			SentToEmails: r.SentToEmails,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   response,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GET /api/reports/:id - Get report details
func (rh *ReportHandler) GetReport(c *gin.Context) {
	reportID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	report, err := rh.repo.GetReport(ctx, reportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
		return
	}

	reportData, err := rh.generator.GenerateMonthlyReport(ctx, report.CustomerID, report.Month, report.Year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           report.ID,
		"customer_name": reportData.CustomerName,
		"month":        reportData.Month,
		"month_num":    reportData.MonthNum,
		"year":         reportData.Year,
		"generated_at": reportData.GeneratedAt,
		"metrics":      reportData.Metrics,
		"tickets_list": reportData.TicketsList,
		"sent_at":      report.SentAt,
	})
}

// GET /api/reports/:id/download - Download CSV or PDF
func (rh *ReportHandler) DownloadReport(c *gin.Context) {
	reportID := c.Param("id")
	format := c.DefaultQuery("format", "pdf")

	if format != "csv" && format != "pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format. Use 'csv' or 'pdf'"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	report, err := rh.repo.GetReport(ctx, reportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
		return
	}

	var data []byte
	var contentType string
	var filename string

	if format == "csv" {
		data = report.CSVData
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		filename = fmt.Sprintf("report_%d_%d.xlsx", report.Month, report.Year)
	} else {
		data = report.PDFData
		contentType = "application/pdf"
		filename = fmt.Sprintf("report_%d_%d.pdf", report.Month, report.Year)
	}

	if len(data) == 0 {
		// Dynamically generate if missing
		reportData, err := rh.generator.GenerateMonthlyReport(ctx, report.CustomerID, report.Month, report.Year)
		if err == nil {
			if format == "csv" {
				data, _ = reports.ExportToCSV(reportData)
			} else {
				data, _ = reports.ExportToPDF(reportData)
			}
			// Best effort save
			_, _ = rh.repo.SaveReport(ctx, report.CustomerID, report.Month, report.Year, report.CSVData, report.PDFData, report.SentToEmails)
		}

		if len(data) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Report file not found or failed to generate"})
			return
		}
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, contentType, data)
}

// POST /api/reports/:id/resend - Resend report email
func (rh *ReportHandler) ResendReport(c *gin.Context) {
	reportID := c.Param("id")

	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Minute)
	defer cancel()

	report, err := rh.repo.GetReport(ctx, reportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
		return
	}

	// Fetch customer for name and SLA hours
	var customer models.Customer
	if err := rh.db.WithContext(ctx).First(&customer, report.CustomerID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Customer not found"})
		return
	}

	// Regenerate metrics for email body
	reportData, err := rh.generator.GenerateMonthlyReport(ctx, customer.ID, report.Month, report.Year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to regenerate report"})
		return
	}

	// Send email using handler's mailer
	if err := rh.mailer.SendReportEmail(ctx, req.Email, customer.Name,
		report.CSVData, report.PDFData, &reportData.Metrics, reportData.Month); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to send email: %v", err)})
		return
	}

	// Update sent tracking
	_ = rh.repo.MarkAsSent(ctx, reportID, []string{req.Email})

	c.JSON(http.StatusOK, gin.H{"message": "Report email sent successfully"})
}

// DELETE /api/reports/:id - Archive report (soft delete)
func (rh *ReportHandler) DeleteReport(c *gin.Context) {
	reportID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := rh.repo.DeleteReport(ctx, reportID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to delete report: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Report deleted successfully"})
}
