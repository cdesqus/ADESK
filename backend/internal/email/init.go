package email

// Package initialization helper
// This file provides utility functions for initializing email components

import (
	"log"
	"time"

	"ai-desk/config"
	"ai-desk/internal/handlers"
	"ai-desk/internal/jobs"
	"gorm.io/gorm"
)

// InitializeEmailComponents initializes all email-related services
// Returns emailHandler and emailPoller, or nil if email is disabled
func InitializeEmailComponents(
	db *gorm.DB,
	cfg *config.Config,
) (*handlers.EmailHandler, *jobs.EmailPollerJob) {
	// Check if email is configured
	if cfg.EmailUser == "" || cfg.EmailPassword == "" {
		log.Printf("Email integration disabled: EMAIL_USER and EMAIL_PASSWORD not configured")
		return nil, nil
	}

	// Initialize domain matcher
	domainMatcher := NewDomainMatcher(db)

	// Initialize SMTP client
	smtpClient := NewSMTPClient(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUser,
		cfg.SMTPPassword,
		cfg.SMTPUser,
		cfg.EmailFromName,
	)

	// Initialize email handler
	emailHandler := handlers.NewEmailHandler(db, domainMatcher, smtpClient)

	// Initialize IMAP client
	imapClient := NewIMAPClient(
		cfg.EmailIMAPHost,
		cfg.EmailIMAPPort,
		cfg.EmailUser,
		cfg.EmailPassword,
	)

	// Parse polling interval
	pollingInterval, _ := time.ParseDuration(cfg.EmailPollingInterval)
	if pollingInterval == 0 {
		pollingInterval = 5 * time.Minute
	}

	// Initialize email poller
	emailPoller := jobs.NewEmailPollerJob(
		db,
		imapClient,
		emailHandler,
		domainMatcher,
		pollingInterval,
	)

	// Start email poller
	if err := emailPoller.Start(); err != nil {
		log.Printf("Warning: Failed to start email poller: %v", err)
	}

	return emailHandler, emailPoller
}
