package reports

import (
	"context"
	"fmt"
	"time"

	"ai-desk/internal/models"
	"github.com/google/uuid"
	"gorm.io/clause"
	"gorm.io/gorm"
)

type ReportRepository struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (rr *ReportRepository) SaveReport(ctx context.Context, customerID uint, month, year int,
	csvData, pdfData []byte, sentToEmails []string) (*models.MonthlyReport, error) {

	report := &models.MonthlyReport{
		ID:           uuid.New().String(),
		CustomerID:   customerID,
		Month:        month,
		Year:         year,
		CSVData:      csvData,
		PDFData:      pdfData,
		GeneratedAt:  time.Now(),
		SentToEmails: sentToEmails,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Use UPSERT to handle duplicate (customer_id, month, year)
	result := rr.db.WithContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(report)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to save report: %w", result.Error)
	}

	return report, nil
}

func (rr *ReportRepository) GetReport(ctx context.Context, reportID string) (*models.MonthlyReport, error) {
	var report models.MonthlyReport
	if err := rr.db.WithContext(ctx).First(&report, "id = ?", reportID).Error; err != nil {
		return nil, fmt.Errorf("report not found: %w", err)
	}
	return &report, nil
}

func (rr *ReportRepository) GetReportByCustomerMonthYear(ctx context.Context, customerID uint, month, year int) (*models.MonthlyReport, error) {
	var report models.MonthlyReport
	if err := rr.db.WithContext(ctx).First(&report, "customer_id = ? AND month = ? AND year = ?", customerID, month, year).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch report: %w", err)
	}
	return &report, nil
}

func (rr *ReportRepository) ListReports(ctx context.Context, customerID *uint, limit, offset int) ([]models.MonthlyReport, int64, error) {
	var reports []models.MonthlyReport
	var count int64

	query := rr.db.WithContext(ctx)

	if customerID != nil {
		query = query.Where("customer_id = ?", *customerID)
	}

	// Get total count
	if err := query.Model(&models.MonthlyReport{}).Count(&count).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count reports: %w", err)
	}

	// Fetch paginated results
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&reports).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch reports: %w", err)
	}

	return reports, count, nil
}

func (rr *ReportRepository) MarkAsSent(ctx context.Context, reportID string, sentToEmails []string) error {
	now := time.Now()
	return rr.db.WithContext(ctx).Model(&models.MonthlyReport{}).
		Where("id = ?", reportID).
		Updates(map[string]interface{}{
			"sent_at":        now,
			"sent_to_emails": sentToEmails,
		}).Error
}

func (rr *ReportRepository) DeleteReport(ctx context.Context, reportID string) error {
	return rr.db.WithContext(ctx).Where("id = ?", reportID).Delete(&models.MonthlyReport{}).Error
}
