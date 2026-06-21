package whatsapp

import (
	"fmt"
	"log"
	"time"

	"ai-desk/internal/models"
	"gorm.io/gorm"
)

// StartDailyScheduler starts a background goroutine to send daily summaries to support groups
func StartDailyScheduler(db *gorm.DB, messageSender *MessageSender, hour, minute int) {
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
			log.Printf("[Scheduler] Next daily summary scheduled for: %v", nextRun.Format(time.RFC1123))
			
			// Wait until the scheduled time
			time.Sleep(durationToWait)
			
			// Execute the task
			runDailySummary(db, messageSender)
		}
	}()
}

func runDailySummary(db *gorm.DB, messageSender *MessageSender) {
	log.Println("[Scheduler] Running daily summary task...")
	
	// Get stats for today
	var openCount, inProgressCount, resolvedCount int64
	today := time.Now().Truncate(24 * time.Hour) // Midnight today
	
	db.Model(&models.Ticket{}).Where("status = ?", "OPEN").Count(&openCount)
	db.Model(&models.Ticket{}).Where("status = ?", "IN_PROGRESS").Count(&inProgressCount)
	db.Model(&models.Ticket{}).Where("status = ? AND resolved_at >= ?", "RESOLVED", today).Count(&resolvedCount)

	summaryMsg := fmt.Sprintf(`📊 *Daily Ticket Summary* 📊
Tanggal: %s

🔹 Tiket OPEN: %d
⏳ Tiket IN PROGRESS: %d
✅ Tiket RESOLVED (Hari Ini): %d

Ayo semangat tim! 🚀`, time.Now().Format("02 Jan 2006"), openCount, inProgressCount, resolvedCount)

	// Find support groups
	var supportGroups []models.CustomerWAGroup
	if err := db.Where("is_support = ?", true).Find(&supportGroups).Error; err != nil {
		log.Printf("[Scheduler] Error finding support groups: %v", err)
		return
	}
	
	if len(supportGroups) == 0 {
		log.Println("[Scheduler] No support groups configured.")
		return
	}
	
	// We need an active session to send messages
	var session models.WhatsAppSession
	if err := db.Where("status = ? AND deleted_at IS NULL", "WORKING").First(&session).Error; err != nil {
		log.Printf("[Scheduler] No active WhatsApp session available to send summaries.")
		return
	}
	
	for _, group := range supportGroups {
		if err := messageSender.SendMessage(session.SessionName, group.GroupID, summaryMsg); err != nil {
			log.Printf("[Scheduler] Failed to send summary to group %s: %v", group.GroupID, err)
		} else {
			log.Printf("[Scheduler] Sent daily summary to group %s", group.GroupID)
		}
	}
}
