package reports

import (
	"context"
	"fmt"
	"time"

	"ai-desk/internal/models"
	"gorm.io/gorm"
)

type ReportGenerator struct {
	db       *gorm.DB
	slaHours int
}

func NewReportGenerator(db *gorm.DB, slaHours int) *ReportGenerator {
	if slaHours <= 0 {
		slaHours = 24
	}
	return &ReportGenerator{
		db:       db,
		slaHours: slaHours,
	}
}

func (rg *ReportGenerator) GenerateMonthlyReport(ctx context.Context, customerID uint, month, year int) (*models.ReportData, error) {
	if month < 1 || month > 12 {
		return nil, fmt.Errorf("invalid month: %d", month)
	}

	// Fetch customer
	var customer models.Customer
	if err := rg.db.WithContext(ctx).First(&customer, customerID).Error; err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	// Get all tickets for the month
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	var tickets []models.Ticket
	if err := rg.db.WithContext(ctx).
		Where("customer_id = ? AND created_at >= ? AND created_at < ?", customerID, startDate, endDate).
		Preload("Engineer").
		Order("created_at DESC").
		Find(&tickets).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch tickets: %w", err)
	}

	// Initialize metrics
	metrics := models.ReportMetrics{
		ByStatus:      make(map[string]int),
		ByPriority:    make(map[string]int),
		BySource:      make(map[string]int),
		EngineerStats: []models.EngineerStat{},
	}

	// Track engineer stats: map[engineerID] = {name, handled, totalTime, resolved}
	engineerMap := make(map[uint]*engineerStats)

	var ticketsList []models.TicketSummary
	var totalResolutionTime float64
	var resolvedCount int
	var slaCompliantCount int

	for _, ticket := range tickets {
		metrics.TotalTickets++

		// Count by status
		metrics.ByStatus[ticket.Status]++
		switch ticket.Status {
		case "OPEN":
			metrics.OpenTickets++
		case "IN_PROGRESS":
			metrics.InProgressTickets++
		case "RESOLVED", "CLOSED":
			metrics.ResolvedTickets++
			resolvedCount++
		}

		// Count by priority
		metrics.ByPriority[ticket.Priority]++

		// Count by source
		metrics.BySource[ticket.Source]++

		// Calculate resolution time
		var timeToResolve float64
		if ticket.ResolvedAt != nil {
			timeToResolve = ticket.ResolvedAt.Sub(ticket.CreatedAt).Hours()
			totalResolutionTime += timeToResolve

			// Check SLA compliance
			if timeToResolve <= float64(rg.slaHours) {
				slaCompliantCount++
			}
		}

		// Track engineer stats
		if ticket.EngineerID != nil {
			if _, exists := engineerMap[*ticket.EngineerID]; !exists {
				name := "Unknown Engineer"
				if ticket.Engineer != nil && ticket.Engineer.Name != "" {
					name = ticket.Engineer.Name
				} else {
					var engineer models.Engineer
					if err := rg.db.WithContext(ctx).First(&engineer, *ticket.EngineerID).Error; err == nil {
						name = engineer.Name
					}
				}
				engineerMap[*ticket.EngineerID] = &engineerStats{
					ID:   *ticket.EngineerID,
					Name: name,
				}
			}
			engineerMap[*ticket.EngineerID].Handled++
			if ticket.ResolvedAt != nil {
				engineerMap[*ticket.EngineerID].TotalTime += timeToResolve
				engineerMap[*ticket.EngineerID].ResolvedCount++
			}
		}

		// Build ticket summary
		summary := models.TicketSummary{
			TicketNumber:  ticket.TicketNumber,
			ID:            ticket.ID,
			Title:         ticket.Title,
			Description:   ticket.Description,
			CreatedAt:     ticket.CreatedAt,
			ResolvedAt:    ticket.ResolvedAt,
			TimeToResolve: timeToResolve,
			Status:        ticket.Status,
			Source:        ticket.Source,
		}
		if ticket.Engineer != nil {
			summary.Engineer = ticket.Engineer.Name
		}
		ticketsList = append(ticketsList, summary)
	}

	// Calculate averages
	if resolvedCount > 0 {
		metrics.AverageResolutionTime = totalResolutionTime / float64(resolvedCount)
		metrics.SLACompliance = (float64(slaCompliantCount) / float64(resolvedCount)) * 100
	}

	// Build engineer stats slice
	for _, stats := range engineerMap {
		engStat := models.EngineerStat{
			EngineerID:     stats.ID,
			Name:           stats.Name,
			TicketsHandled: stats.Handled,
			ResolutionRate: 0,
		}
		if stats.Handled > 0 {
			engStat.AvgTime = stats.TotalTime / float64(stats.Handled)
			engStat.ResolutionRate = (float64(stats.ResolvedCount) / float64(stats.Handled)) * 100
		}
		metrics.EngineerStats = append(metrics.EngineerStats, engStat)
	}

	// Get month name
	monthName := startDate.Format("January 2006")

	return &models.ReportData{
		CustomerName: customer.Name,
		Month:        monthName,
		MonthNum:     month,
		Year:         year,
		GeneratedAt:  time.Now(),
		Metrics:      metrics,
		TicketsList:  ticketsList,
	}, nil
}

type engineerStats struct {
	ID            uint
	Name          string
	Handled       int
	TotalTime     float64
	ResolvedCount int
}
