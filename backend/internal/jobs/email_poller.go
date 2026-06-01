package jobs

import (
	"context"
	"log"
	"time"

	"ai-desk/internal/email"
	"ai-desk/internal/handlers"
	"gorm.io/gorm"
)

type EmailPollerJob struct {
	db              *gorm.DB
	imapClient      *email.IMAPClient
	emailHandler    *handlers.EmailHandler
	domainMatcher   *email.DomainMatcher
	ticker          *time.Ticker
	done            chan bool
	interval        time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
	isRunning       bool
}

// NewEmailPollerJob creates a new email poller job
func NewEmailPollerJob(
	db *gorm.DB,
	imapClient *email.IMAPClient,
	emailHandler *handlers.EmailHandler,
	domainMatcher *email.DomainMatcher,
	interval time.Duration,
) *EmailPollerJob {
	ctx, cancel := context.WithCancel(context.Background())
	return &EmailPollerJob{
		db:            db,
		imapClient:    imapClient,
		emailHandler:  emailHandler,
		domainMatcher: domainMatcher,
		interval:      interval,
		done:          make(chan bool),
		ctx:           ctx,
		cancel:        cancel,
		isRunning:     false,
	}
}

// Start starts the email polling job
func (epj *EmailPollerJob) Start() error {
	if epj.isRunning {
		log.Printf("Email poller already running")
		return nil
	}

	// Initial connection
	if err := epj.imapClient.Connect(); err != nil {
		log.Printf("Failed to connect IMAP on startup: %v", err)
		return err
	}

	epj.isRunning = true
	epj.ticker = time.NewTicker(epj.interval)

	go epj.run()

	log.Printf("Email poller started with interval %v", epj.interval)
	return nil
}

// Stop stops the email polling job gracefully
func (epj *EmailPollerJob) Stop() {
	if !epj.isRunning {
		return
	}

	log.Printf("Stopping email poller...")
	epj.cancel()
	epj.ticker.Stop()

	// Wait for current processing to finish
	select {
	case <-epj.done:
		log.Printf("Email poller stopped")
	case <-time.After(30 * time.Second):
		log.Printf("Email poller stop timeout")
	}

	_ = epj.imapClient.Close()
	epj.isRunning = false
}

// run is the main loop for email polling
func (epj *EmailPollerJob) run() {
	defer func() { epj.done <- true }()

	// Run immediately on start
	epj.pollEmails()

	for {
		select {
		case <-epj.ctx.Done():
			return
		case <-epj.ticker.C:
			epj.pollEmails()
		}
	}
}

// pollEmails fetches and processes unread emails
func (epj *EmailPollerJob) pollEmails() {
	log.Printf("[EmailPoller] Starting email polling cycle")

	// Ensure connection is alive
	if err := epj.imapClient.Connect(); err != nil {
		log.Printf("[EmailPoller] Connection error: %v, attempting reconnect", err)
		if err := epj.imapClient.Reconnect(); err != nil {
			log.Printf("[EmailPoller] Reconnection failed: %v", err)
			return
		}
	}

	// Fetch unread emails
	messages, err := epj.imapClient.FetchUnreadEmails()
	if err != nil {
		log.Printf("[EmailPoller] Failed to fetch emails: %v", err)
		return
	}

	if len(messages) == 0 {
		log.Printf("[EmailPoller] No unread emails")
		return
	}

	log.Printf("[EmailPoller] Found %d unread emails", len(messages))

	// Process each email
	successCount := 0
	failureCount := 0

	for _, msg := range messages {
		if msg == nil {
			continue
		}

		uid := msg.Uid
		log.Printf("[EmailPoller] Processing email UID: %d", uid)

		// Parse email
		emailMsg, err := email.ParseEmail(msg)
		if err != nil {
			log.Printf("[EmailPoller] Failed to parse email UID %d: %v", uid, err)
			failureCount++

			// Still mark as read even if parsing fails
			_ = epj.imapClient.MarkAsRead(uid)
			continue
		}

		log.Printf("[EmailPoller] Parsed email from: %s, subject: %s", emailMsg.From, emailMsg.Subject)

		// Process email and create ticket
		ticket, err := epj.emailHandler.ProcessEmailWithLogging(emailMsg)
		if err != nil {
			log.Printf("[EmailPoller] Failed to process email from %s: %v", emailMsg.From, err)
			failureCount++
		} else {
			log.Printf("[EmailPoller] Successfully created ticket %d from email", ticket.ID)
			successCount++
		}

		// Mark as read regardless of success/failure
		if err := epj.imapClient.MarkAsRead(uid); err != nil {
			log.Printf("[EmailPoller] Failed to mark UID %d as read: %v", uid, err)
		}
	}

	log.Printf("[EmailPoller] Polling cycle complete. Success: %d, Failed: %d", successCount, failureCount)
}

// IsRunning returns whether the poller is currently running
func (epj *EmailPollerJob) IsRunning() bool {
	return epj.isRunning
}
