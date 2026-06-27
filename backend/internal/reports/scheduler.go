package reports

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"ai-desk/internal/models"
	"gorm.io/gorm"
)

type ReportScheduler struct {
	db        *gorm.DB
	generator *ReportGenerator
	mailer    *ReportMailer
	repo      *ReportRepository
	running   bool
	mu        sync.Mutex
	stopChan  chan struct{}
	slaHours  int
}

func NewReportScheduler(db *gorm.DB, generator *ReportGenerator,
	mailer *ReportMailer, repo *ReportRepository, slaHours int) *ReportScheduler {
	return &ReportScheduler{
		db:        db,
		generator: generator,
		mailer:    mailer,
		repo:      repo,
		stopChan:  make(chan struct{}),
		slaHours:  slaHours,
	}
}

func (rs *ReportScheduler) Start() error {
	rs.mu.Lock()
	if rs.running {
		rs.mu.Unlock()
		return fmt.Errorf("report scheduler already running")
	}
	rs.running = true
	rs.mu.Unlock()

	go rs.run()
	return nil
}

func (rs *ReportScheduler) Stop() {
	rs.mu.Lock()
	if rs.running {
		rs.running = false
		close(rs.stopChan)
	}
	rs.mu.Unlock()
}

func (rs *ReportScheduler) IsRunning() bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.running
}

func (rs *ReportScheduler) run() {
	// Calculate time until next 1st of month at 8:00 AM
	nextRun := rs.calculateNextRun()
	ticker := time.NewTicker(time.Until(nextRun))
	defer ticker.Stop()

	log.Printf("Report scheduler started. Next run: %s", nextRun.Format("2006-01-02 15:04:05 MST"))

	for {
		select {
		case <-rs.stopChan:
			log.Printf("Report scheduler stopped")
			return
		case <-ticker.C:
			log.Printf("Starting monthly report generation...")
			rs.generateAllReports()

			// Reset ticker for next month
			nextRun = rs.calculateNextRun()
			ticker.Reset(time.Until(nextRun))
			log.Printf("Next report generation scheduled for: %s", nextRun.Format("2006-01-02 15:04:05 MST"))
		}
	}
}

func (rs *ReportScheduler) calculateNextRun() time.Time {
	now := time.Now()
	// Next run: 1st of next month at 8:00 AM
	nextMonth := now.AddDate(0, 1, 0)
	nextRun := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 8, 0, 0, 0, now.Location())

	// If we're still on the 1st and before 8:00 AM, run at 8:00 AM today
	if now.Day() == 1 && now.Hour() < 8 {
		nextRun = time.Date(now.Year(), now.Month(), 1, 8, 0, 0, 0, now.Location())
	}

	return nextRun
}

func (rs *ReportScheduler) generateAllReports() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Get all active customers
	var customers []models.Customer
	if err := rs.db.WithContext(ctx).Where("is_active = ?", true).Find(&customers).Error; err != nil {
		log.Printf("Error fetching active customers: %v", err)
		return
	}

	// Generate reports for previous month
	now := time.Now()
	previousMonth := now.AddDate(0, -1, 0)
	month := int(previousMonth.Month())
	year := previousMonth.Year()

	successCount := 0
	failureCount := 0

	for _, customer := range customers {
		if err := rs.generateAndSendReport(ctx, customer, month, year); err != nil {
			log.Printf("Failed to generate report for customer %s (ID: %d): %v", customer.Name, customer.ID, err)
			failureCount++
		} else {
			log.Printf("Successfully generated report for customer %s (ID: %d)", customer.Name, customer.ID)
			successCount++
		}
	}

	log.Printf("Report generation completed. Success: %d, Failed: %d", successCount, failureCount)
}

func (rs *ReportScheduler) generateAndSendReport(ctx context.Context, customer models.Customer, month, year int) error {
	// Generate report data
	reportData, err := rs.generator.GenerateMonthlyReport(ctx, customer.ID, month, year)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Export to CSV
	csvBytes, err := ExportToCSV(reportData)
	if err != nil {
		return fmt.Errorf("failed to export CSV: %w", err)
	}

	// Export to PDF
	pdfBytes, err := ExportToPDF(reportData)
	if err != nil {
		return fmt.Errorf("failed to export PDF: %w", err)
	}

	// Save to database
	report, err := rs.repo.SaveReport(ctx, customer.ID, month, year, csvBytes, pdfBytes, []string{customer.EmailSupport})
	if err != nil {
		return fmt.Errorf("failed to save report: %w", err)
	}

	// Send email if customer has email configured
	if customer.EmailSupport != "" {
		if err := rs.mailer.SendReportEmail(ctx, customer.EmailSupport, customer.Name,
			csvBytes, pdfBytes, &reportData.Metrics, reportData.Month); err != nil {
			log.Printf("Warning: Failed to send report email to %s: %v", customer.EmailSupport, err)
			// Continue even if email fails - report is still saved
		} else {
			// Mark as sent
			_ = rs.repo.MarkAsSent(ctx, report.ID, []string{customer.EmailSupport})
		}
	}

	return nil
}

// GenerateReportOnDemand generates a report immediately without waiting for schedule
func (rs *ReportScheduler) GenerateReportOnDemand(ctx context.Context, customerID uint, month, year int) (*models.ReportData, error) {
	// Generate report data
	reportData, err := rs.generator.GenerateMonthlyReport(ctx, customerID, month, year)
	if err != nil {
		return nil, fmt.Errorf("failed to generate report: %w", err)
	}

	// Export to CSV
	csvBytes, err := ExportToCSV(reportData)
	if err != nil {
		return nil, fmt.Errorf("failed to export CSV: %w", err)
	}

	// Export to PDF
	pdfBytes, err := ExportToPDF(reportData)
	if err != nil {
		return nil, fmt.Errorf("failed to export PDF: %w", err)
	}

	// Save to database
	_, err = rs.repo.SaveReport(ctx, customerID, month, year, csvBytes, pdfBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	return reportData, nil
}
