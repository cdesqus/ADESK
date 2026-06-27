package reports

import (
	"bytes"
	"fmt"

	"ai-desk/internal/models"
	"github.com/xuri/excelize/v2"
)

func ExportToCSV(report *models.ReportData) ([]byte, error) {
	f := excelize.NewFile()

	// Create Summary sheet
	summarySheet := "Summary"
	f.NewSheet(summarySheet)

	// Set column widths
	f.SetColWidth(summarySheet, "A", "F", 20)

	// Add title
	f.SetCellValue(summarySheet, "A1", "Monthly Support Report")
	f.SetCellValue(summarySheet, "A2", fmt.Sprintf("Customer: %s", report.CustomerName))
	f.SetCellValue(summarySheet, "A3", fmt.Sprintf("Month: %s", report.Month))
	f.SetCellValue(summarySheet, "A4", fmt.Sprintf("Generated: %s", report.GeneratedAt.Format("2006-01-02 15:04:05")))

	row := 6
	// Header row
	headers := []string{"Metric", "Value"}
	f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), headers[0])
	f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), headers[1])
	row++

	// Key metrics
	f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), "Total Tickets")
	f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), report.Metrics.TotalTickets)
	row++

	f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), "Resolved Tickets")
	f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), report.Metrics.ResolvedTickets)
	row++

	f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), "Open Tickets")
	f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), report.Metrics.OpenTickets)
	row++

	f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), "In Progress")
	f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), report.Metrics.InProgressTickets)
	row++

	f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), "Avg Resolution Time (hrs)")
	f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%.2f", report.Metrics.AverageResolutionTime))
	row++

	f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), "SLA Compliance (%)")
	f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%.1f", report.Metrics.SLACompliance))
	row += 2

	// Breakdown by Status
	f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), "Tickets by Status")
	row++
	f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), "Status")
	f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), "Count")
	row++
	for status, count := range report.Metrics.ByStatus {
		f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), status)
		f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), count)
		row++
	}
	row++

	// Breakdown by Priority
	f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), "Tickets by Priority")
	row++
	f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), "Priority")
	f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), "Count")
	row++
	for priority, count := range report.Metrics.ByPriority {
		f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), priority)
		f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), count)
		row++
	}
	row++

	// Breakdown by Source
	f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), "Tickets by Source")
	row++
	f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), "Source")
	f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), "Count")
	row++
	for source, count := range report.Metrics.BySource {
		f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), source)
		f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), count)
		row++
	}

	// Create Engineer Performance sheet
	engineerSheet := "Engineers"
	f.NewSheet(engineerSheet)
	f.SetColWidth(engineerSheet, "A", "D", 20)

	f.SetCellValue(engineerSheet, "A1", "Engineer Performance")
	f.SetCellValue(engineerSheet, "A3", "Engineer Name")
	f.SetCellValue(engineerSheet, "B3", "Tickets Handled")
	f.SetCellValue(engineerSheet, "C3", "Avg Time (hrs)")
	f.SetCellValue(engineerSheet, "D3", "Resolution Rate (%)")

	row = 4
	for _, stat := range report.Metrics.EngineerStats {
		f.SetCellValue(engineerSheet, fmt.Sprintf("A%d", row), stat.Name)
		f.SetCellValue(engineerSheet, fmt.Sprintf("B%d", row), stat.TicketsHandled)
		f.SetCellValue(engineerSheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%.2f", stat.AvgTime))
		f.SetCellValue(engineerSheet, fmt.Sprintf("D%d", row), fmt.Sprintf("%.1f", stat.ResolutionRate))
		row++
	}

	// Create Tickets Detail sheet
	ticketsSheet := "Tickets"
	f.NewSheet(ticketsSheet)
	f.SetColWidth(ticketsSheet, "A", "I", 18)

	f.SetCellValue(ticketsSheet, "A1", "All Tickets")
	f.SetCellValue(ticketsSheet, "A2", "Ticket ID")
	f.SetCellValue(ticketsSheet, "B2", "Title")
	f.SetCellValue(ticketsSheet, "C2", "Description")
	f.SetCellValue(ticketsSheet, "D2", "Created")
	f.SetCellValue(ticketsSheet, "E2", "Resolved")
	f.SetCellValue(ticketsSheet, "F2", "Time (hrs)")
	f.SetCellValue(ticketsSheet, "G2", "Status")
	f.SetCellValue(ticketsSheet, "H2", "Engineer")
	f.SetCellValue(ticketsSheet, "I2", "Source")

	row = 3
	for _, ticket := range report.TicketsList {
		f.SetCellValue(ticketsSheet, fmt.Sprintf("A%d", row), ticket.TicketNumber)
		f.SetCellValue(ticketsSheet, fmt.Sprintf("B%d", row), ticket.Title)
		f.SetCellValue(ticketsSheet, fmt.Sprintf("C%d", row), ticket.Description)
		f.SetCellValue(ticketsSheet, fmt.Sprintf("D%d", row), ticket.CreatedAt.Format("2006-01-02 15:04"))
		if ticket.ResolvedAt != nil {
			f.SetCellValue(ticketsSheet, fmt.Sprintf("E%d", row), ticket.ResolvedAt.Format("2006-01-02 15:04"))
		}
		f.SetCellValue(ticketsSheet, fmt.Sprintf("F%d", row), fmt.Sprintf("%.2f", ticket.TimeToResolve))
		f.SetCellValue(ticketsSheet, fmt.Sprintf("G%d", row), ticket.Status)
		f.SetCellValue(ticketsSheet, fmt.Sprintf("H%d", row), ticket.Engineer)
		f.SetCellValue(ticketsSheet, fmt.Sprintf("I%d", row), ticket.Source)
		row++
	}

	// Remove default sheet
	f.DeleteSheet("Sheet1")

	// Write to buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write CSV file: %w", err)
	}

	return buf.Bytes(), nil
}
