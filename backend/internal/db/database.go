package db

import (
	"fmt"
	"log"

	"ai-desk/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB initializes the database connection with connection pooling
func InitDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	return db, nil
}

// Migrate runs all migrations
func removeDuplicateWhatsAppLogs(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.WhatsAppLog{}) {
		return nil
	}

	return db.Exec(`
		DELETE FROM whatsapp_logs
		WHERE id IN (
			SELECT id FROM (
				SELECT id,
					ROW_NUMBER() OVER (PARTITION BY session_name, message_id ORDER BY created_at, id) AS rn
				FROM whatsapp_logs
			) duplicates
			WHERE duplicates.rn > 1
		);
	`).Error
}

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		&models.User{},
		&models.Customer{},
		&models.Engineer{},
		&models.Ticket{},
		&models.Update{},
		&models.EmailLog{},
		&models.WhatsAppSession{},
		&models.EngineerWAPhone{},
		&models.WhatsAppLog{},
		&models.MonthlyReport{},
		&models.SystemSetting{},
		&models.CustomerWAGroup{},
	)
	if err != nil {
		// If auto-migrate failed, it might be due to ticket_number unique constraint on existing rows.
		// Let's try to fix existing rows if the column exists.
		if db.Migrator().HasColumn(&models.Ticket{}, "TicketNumber") {
			db.Exec("UPDATE tickets SET ticket_number = 'TK-MIG-' || id WHERE ticket_number IS NULL OR ticket_number = ''")
		}

		if derr := removeDuplicateWhatsAppLogs(db); derr != nil {
			log.Printf("Failed to remove duplicate WhatsApp logs: %v", derr)
		}
		
		// Retry migration
		err = db.AutoMigrate(
			&models.User{},
			&models.Customer{},
			&models.Engineer{},
			&models.Ticket{},
			&models.Update{},
			&models.EmailLog{},
			&models.WhatsAppSession{},
			&models.EngineerWAPhone{},
			&models.WhatsAppLog{},
			&models.MonthlyReport{},
			&models.SystemSetting{},
			&models.CustomerWAGroup{},
		)
		if err != nil {
			return err
		}
	}
	
	// Ensure customer_id can be null for Engineers handling all customers
	db.Exec("ALTER TABLE engineers ALTER COLUMN customer_id DROP NOT NULL;")
	
	return nil
}

// Close closes the database connection
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	return sqlDB.Close()
}
