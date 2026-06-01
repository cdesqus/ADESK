package reports

import (
	"bytes"
	"fmt"
	"sort"

	"ai-desk/internal/models"
	fpdf "github.com/phpdave11/gofpdf"
)

const (
	colorHeaderRGB = 46
	colorHeaderG   = 117
	colorHeaderB   = 182
	pageMargin     = 10
	headerHeight   = 8
	cellHeight     = 6
)

func cell(pdf *fpdf.Fpdf, w, h float64, txt, border string, ln int, align string, fill bool) {
	pdf.CellFormat(w, h, txt, border, ln, align, fill, 0, "")
}

func ExportToPDF(report *models.ReportData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(pageMargin, pageMargin, pageMargin)
	pdf.SetAutoPageBreak(true, 15)

	// Page 1: Cover page
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 24)
	pdf.SetTextColor(colorHeaderRGB, colorHeaderG, colorHeaderB)
	cell(pdf, 0, 20, "Monthly Support Report", "0", 1, "C", false)

	pdf.SetFont("Helvetica", "", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(10)
	cell(pdf, 0, 10, fmt.Sprintf("Customer: %s", report.CustomerName), "0", 1, "C", false)
	cell(pdf, 0, 10, fmt.Sprintf("Month: %s", report.Month), "0", 1, "C", false)
	cell(pdf, 0, 10, fmt.Sprintf("Year: %d", report.Year), "0", 1, "C", false)
	cell(pdf, 0, 10, fmt.Sprintf("Generated: %s", report.GeneratedAt.Format("2006-01-02 15:04")), "0", 1, "C", false)

	// Page 2: Executive Summary
	pdf.AddPage()
	addSectionHeader(pdf, "Executive Summary")

	// Key metrics boxes
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(colorHeaderRGB, colorHeaderG, colorHeaderB)
	pdf.SetTextColor(255, 255, 255)

	metrics := []struct {
		label string
		value string
	}{
		{"Total Tickets", fmt.Sprintf("%d", report.Metrics.TotalTickets)},
		{"Resolved", fmt.Sprintf("%d (%.1f%%)", report.Metrics.ResolvedTickets, float64(report.Metrics.ResolvedTickets)*100/float64(report.Metrics.TotalTickets))},
		{"Avg Resolution", fmt.Sprintf("%.2f hrs", report.Metrics.AverageResolutionTime)},
		{"SLA Compliance", fmt.Sprintf("%.1f%%", report.Metrics.SLACompliance)},
	}

	boxWidth := 40.0
	for _, m := range metrics {
		cell(pdf, boxWidth, 10, m.label, "1", 0, "C", true)
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFillColor(240, 240, 240)
		cell(pdf, boxWidth, 10, m.value, "1", 1, "C", true)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetFillColor(colorHeaderRGB, colorHeaderG, colorHeaderB)
		pdf.SetTextColor(255, 255, 255)
	}

	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(5)

	// Status breakdown table
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(colorHeaderRGB, colorHeaderG, colorHeaderB)
	pdf.SetTextColor(255, 255, 255)
	cell(pdf, 30, headerHeight, "Status", "1", 0, "C", true)
	cell(pdf, 30, headerHeight, "Count", "1", 1, "C", true)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(0, 0, 0)
	alternateRow := false
	for status, count := range report.Metrics.ByStatus {
		if alternateRow {
			pdf.SetFillColor(240, 240, 240)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		cell(pdf, 30, cellHeight, status, "1", 0, "L", true)
		cell(pdf, 30, cellHeight, fmt.Sprintf("%d", count), "1", 1, "C", true)
		alternateRow = !alternateRow
	}

	pdf.Ln(5)

	// Priority breakdown
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(colorHeaderRGB, colorHeaderG, colorHeaderB)
	pdf.SetTextColor(255, 255, 255)
	cell(pdf, 30, headerHeight, "Priority", "1", 0, "C", true)
	cell(pdf, 30, headerHeight, "Count", "1", 1, "C", true)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(0, 0, 0)
	alternateRow = false
	priorityOrder := []string{"URGENT", "HIGH", "MEDIUM", "LOW"}
	for _, priority := range priorityOrder {
		if count, ok := report.Metrics.ByPriority[priority]; ok {
			if alternateRow {
				pdf.SetFillColor(240, 240, 240)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
			cell(pdf, 30, cellHeight, priority, "1", 0, "L", true)
			cell(pdf, 30, cellHeight, fmt.Sprintf("%d", count), "1", 1, "C", true)
			alternateRow = !alternateRow
		}
	}

	// Page 3: Engineer Performance & Sources
	pdf.AddPage()
	addSectionHeader(pdf, "Performance & Breakdown")

	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(colorHeaderRGB, colorHeaderG, colorHeaderB)
	pdf.SetTextColor(255, 255, 255)
	cell(pdf, 40, headerHeight, "Engineer", "1", 0, "L", true)
	cell(pdf, 25, headerHeight, "Tickets", "1", 0, "C", true)
	cell(pdf, 25, headerHeight, "Avg Time", "1", 0, "C", true)
	cell(pdf, 25, headerHeight, "Resolution %", "1", 1, "C", true)

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(0, 0, 0)
	alternateRow = false
	for _, stat := range report.Metrics.EngineerStats {
		if alternateRow {
			pdf.SetFillColor(240, 240, 240)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		cell(pdf, 40, cellHeight, stat.Name, "1", 0, "L", true)
		cell(pdf, 25, cellHeight, fmt.Sprintf("%d", stat.TicketsHandled), "1", 0, "C", true)
		cell(pdf, 25, cellHeight, fmt.Sprintf("%.2f h", stat.AvgTime), "1", 0, "C", true)
		cell(pdf, 25, cellHeight, fmt.Sprintf("%.1f%%", stat.ResolutionRate), "1", 1, "C", true)
		alternateRow = !alternateRow
	}

	pdf.Ln(5)

	// Source breakdown
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(colorHeaderRGB, colorHeaderG, colorHeaderB)
	pdf.SetTextColor(255, 255, 255)
	cell(pdf, 30, headerHeight, "Source", "1", 0, "C", true)
	cell(pdf, 30, headerHeight, "Count", "1", 1, "C", true)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(0, 0, 0)
	alternateRow = false
	sourceOrder := []string{"EMAIL", "WHATSAPP", "WEB", "CHAT", "PHONE"}
	for _, source := range sourceOrder {
		if count, ok := report.Metrics.BySource[source]; ok {
			if alternateRow {
				pdf.SetFillColor(240, 240, 240)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
			cell(pdf, 30, cellHeight, source, "1", 0, "L", true)
			cell(pdf, 30, cellHeight, fmt.Sprintf("%d", count), "1", 1, "C", true)
			alternateRow = !alternateRow
		}
	}

	// Page 4+: Tickets Detail
	pdf.AddPage()
	addSectionHeader(pdf, "Ticket Details")

	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetFillColor(colorHeaderRGB, colorHeaderG, colorHeaderB)
	pdf.SetTextColor(255, 255, 255)

	cell(pdf, 12, headerHeight, "ID", "1", 0, "C", true)
	cell(pdf, 35, headerHeight, "Title", "1", 0, "L", true)
	cell(pdf, 20, headerHeight, "Created", "1", 0, "C", true)
	cell(pdf, 20, headerHeight, "Status", "1", 0, "C", true)
	cell(pdf, 15, headerHeight, "Hrs", "1", 0, "C", true)
	cell(pdf, 30, headerHeight, "Engineer", "1", 1, "L", true)

	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(0, 0, 0)
	alternateRow = false

	// Sort tickets by created date descending
	tickets := make([]models.TicketSummary, len(report.TicketsList))
	copy(tickets, report.TicketsList)
	sort.Slice(tickets, func(i, j int) bool {
		return tickets[i].CreatedAt.After(tickets[j].CreatedAt)
	})

	for _, ticket := range tickets {
		if pdf.GetY() > 250 {
			pdf.AddPage()
			addSectionHeader(pdf, "Ticket Details (continued)")
			pdf.SetFont("Helvetica", "B", 8)
			pdf.SetFillColor(colorHeaderRGB, colorHeaderG, colorHeaderB)
			pdf.SetTextColor(255, 255, 255)
			cell(pdf, 12, headerHeight, "ID", "1", 0, "C", true)
			cell(pdf, 35, headerHeight, "Title", "1", 0, "L", true)
			cell(pdf, 20, headerHeight, "Created", "1", 0, "C", true)
			cell(pdf, 20, headerHeight, "Status", "1", 0, "C", true)
			cell(pdf, 15, headerHeight, "Hrs", "1", 0, "C", true)
			cell(pdf, 30, headerHeight, "Engineer", "1", 1, "L", true)
			pdf.SetFont("Helvetica", "", 7)
			pdf.SetTextColor(0, 0, 0)
			alternateRow = false
		}

		if alternateRow {
			pdf.SetFillColor(240, 240, 240)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		cell(pdf, 12, cellHeight, fmt.Sprintf("%d", ticket.ID), "1", 0, "C", true)
		titleLen := len(ticket.Title)
		if titleLen > 25 {
			titleLen = 25
		}
		cell(pdf, 35, cellHeight, ticket.Title[:titleLen], "1", 0, "L", true)
		cell(pdf, 20, cellHeight, ticket.CreatedAt.Format("01-02"), "1", 0, "C", true)
		cell(pdf, 20, cellHeight, ticket.Status, "1", 0, "C", true)
		cell(pdf, 15, cellHeight, fmt.Sprintf("%.1f", ticket.TimeToResolve), "1", 0, "C", true)
		cell(pdf, 30, cellHeight, ticket.Engineer, "1", 1, "L", true)
		alternateRow = !alternateRow
	}

	// Add footer with page numbers
	totalPages := pdf.PageCount()
	for i := 1; i <= totalPages; i++ {
		pdf.Seek(0, 2)
		if i <= pdf.PageCount() {
			pdf.SetPage(i)
			pdf.SetY(-15)
			pdf.SetFont("Helvetica", "I", 8)
			pdf.SetTextColor(128, 128, 128)
			cell(pdf, 0, 10, fmt.Sprintf("Page %d of %d", i, totalPages), "0", 0, "C", false)
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func addSectionHeader(pdf *fpdf.Fpdf, title string) {
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(colorHeaderRGB, colorHeaderG, colorHeaderB)
	cell(pdf, 0, 10, title, "0", 1, "L", false)
	pdf.SetDrawColor(colorHeaderRGB, colorHeaderG, colorHeaderB)
	pdf.Line(pdf.GetX(), pdf.GetY(), pdf.GetX()+190, pdf.GetY())
	pdf.Ln(5)
	pdf.SetTextColor(0, 0, 0)
}
