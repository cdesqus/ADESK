package email

import (
	"fmt"
	"log"
	"strings"
	"time"

	"ai-desk/internal/models"
	"gorm.io/gorm"
)

// StartDailyOpenTicketSummary starts a background goroutine to send daily summaries of OPEN tickets
func StartDailyOpenTicketSummary(db *gorm.DB, smtpClient *SMTPClient, hour, minute int, companyName, frontendURL string) {
	go func() {
		for {
			now := time.Now()
			// Calculate next run time
			nextRun := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
			
			// If the time has already passed today, schedule for tomorrow
			if now.After(nextRun) || now.Equal(nextRun) {
				nextRun = nextRun.Add(24 * time.Hour)
			}
			
			durationToWait := nextRun.Sub(now)
			log.Printf("[Email Scheduler] Next daily OPEN ticket summary scheduled for: %v", nextRun.Format(time.RFC1123))
			
			// Wait until the scheduled time
			time.Sleep(durationToWait)
			
			// Execute the task
			runDailyOpenTicketSummary(db, smtpClient, companyName, frontendURL)
		}
	}()
}

func runDailyOpenTicketSummary(db *gorm.DB, smtpClient *SMTPClient, companyName, frontendURL string) {
	log.Println("[Email Scheduler] Running daily OPEN ticket summary task...")
	
	var openTickets []models.Ticket
	// Preload Customer and Engineer to show details
	if err := db.Preload("Customer").Preload("Engineer").Where("status = ?", "OPEN").Order("created_at asc").Find(&openTickets).Error; err != nil {
		log.Printf("[Email Scheduler] Error finding open tickets: %v", err)
		return
	}

	if len(openTickets) == 0 {
		log.Println("[Email Scheduler] No open tickets found. Skipping summary email.")
		return
	}

	// Build HTML Table Rows
	var rows strings.Builder
	for _, t := range openTickets {
		ticketURL := ""
		if frontendURL != "" {
			ticketURL = fmt.Sprintf("%s/tickets/%d", strings.TrimRight(frontendURL, "/"), t.ID)
		}

		customerName := "-"
		if t.Customer.Name != "" {
			customerName = t.Customer.Name
		}

		engineerName := "Unassigned"
		if t.Engineer != nil && t.Engineer.Name != "" {
			engineerName = t.Engineer.Name
		}

		linkHTML := fmt.Sprintf(`<a href="%s" style="color: #2563eb; text-decoration: none; font-weight: 600;">%s</a>`, ticketURL, t.TicketNumber)
		if ticketURL == "" {
			linkHTML = t.TicketNumber
		}

		priorityColor := "#f59e0b" // MEDIUM
		switch t.Priority {
		case "LOW":
			priorityColor = "#16a34a"
		case "HIGH":
			priorityColor = "#ea580c"
		case "URGENT":
			priorityColor = "#dc2626"
		}

		priorityHTML := fmt.Sprintf(`<span style="color: %s; font-weight: 600;">%s</span>`, priorityColor, t.Priority)

		rows.WriteString(fmt.Sprintf(`
			<tr>
				<td style="padding: 12px; border-bottom: 1px solid #e5e7eb;">%s</td>
				<td style="padding: 12px; border-bottom: 1px solid #e5e7eb; color: #111827; font-weight: 500;">%s</td>
				<td style="padding: 12px; border-bottom: 1px solid #e5e7eb;">%s</td>
				<td style="padding: 12px; border-bottom: 1px solid #e5e7eb;">%s</td>
				<td style="padding: 12px; border-bottom: 1px solid #e5e7eb;">%s</td>
				<td style="padding: 12px; border-bottom: 1px solid #e5e7eb;">%s</td>
			</tr>`,
			linkHTML, t.Title, customerName, engineerName, priorityHTML, t.CreatedAt.Format("02 Jan 2006 15:04"),
		))
	}

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin: 0; padding: 0; background-color: #f3f4f6; font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color: #f3f4f6; padding: 40px 0;">
<tr><td align="center">
<table width="800" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 8px; border: 1px solid #e5e7eb; overflow: hidden; box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);">
	<!-- Header -->
	<tr>
		<td style="background-color: #1e3a8a; padding: 25px 30px; border-bottom: 3px solid #3b82f6;">
			<h1 style="color: #ffffff; margin: 0; font-size: 22px; font-weight: 600;">Laporan Harian Tiket OPEN</h1>
			<p style="color: #bfdbfe; margin: 8px 0 0 0; font-size: 14px;">Terdapat <strong>%d</strong> tiket yang masih berstatus OPEN pada hari ini.</p>
		</td>
	</tr>
	<!-- Content -->
	<tr>
		<td style="padding: 30px;">
			<table width="100%%" cellpadding="0" cellspacing="0" style="border-collapse: collapse; text-align: left; font-size: 13px;">
				<thead>
					<tr style="background-color: #f9fafb; color: #4b5563; text-transform: uppercase; letter-spacing: 0.05em;">
						<th style="padding: 12px; border-bottom: 2px solid #e5e7eb; font-weight: 600;">ID Tiket</th>
						<th style="padding: 12px; border-bottom: 2px solid #e5e7eb; font-weight: 600;">Judul Tiket</th>
						<th style="padding: 12px; border-bottom: 2px solid #e5e7eb; font-weight: 600;">Pelanggan</th>
						<th style="padding: 12px; border-bottom: 2px solid #e5e7eb; font-weight: 600;">Engineer</th>
						<th style="padding: 12px; border-bottom: 2px solid #e5e7eb; font-weight: 600;">Prioritas</th>
						<th style="padding: 12px; border-bottom: 2px solid #e5e7eb; font-weight: 600;">Tgl Dibuat</th>
					</tr>
				</thead>
				<tbody>
					%s
				</tbody>
			</table>
		</td>
	</tr>
	<!-- Footer -->
	<tr>
		<td style="background-color: #f9fafb; padding: 20px 30px; border-top: 1px solid #e5e7eb; text-align: center;">
			<p style="color: #6b7280; font-size: 13px; margin: 0; line-height: 1.5;">
				Pesan ini dibuat secara otomatis oleh sistem.<br>
				<strong>%s Helpdesk System</strong>
			</p>
		</td>
	</tr>
</table>
</td></tr>
</table>
</body>
</html>`, len(openTickets), rows.String(), companyName)

	subject := fmt.Sprintf("[AI-DESK] Ringkasan Harian: %d Tiket Masih OPEN", len(openTickets))
	
	// Send to support
	if err := smtpClient.SendHTMLEmail("support@idesolusi.co.id", "", subject, htmlBody); err != nil {
		log.Printf("[Email Scheduler] Failed to send open tickets summary: %v", err)
	} else {
		log.Printf("[Email Scheduler] Successfully sent open tickets summary to support@idesolusi.co.id")
	}
}
