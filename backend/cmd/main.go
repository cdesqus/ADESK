package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"ai-desk/config"
	"ai-desk/internal/db"
	"ai-desk/internal/email"
	"ai-desk/internal/handlers"
	"ai-desk/internal/jobs"
	"ai-desk/internal/middleware"
	"ai-desk/internal/models"
	"ai-desk/internal/reports"
	"ai-desk/internal/whatsapp"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	database, err := db.InitDB(cfg.GetDSN())
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Run migrations
	if err := db.Migrate(database); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create seed data if database is empty
	seedDatabase(database)

	// Override config with DB settings
	var settings []models.SystemSetting
	if err := database.Find(&settings).Error; err == nil {
		for _, s := range settings {
			switch s.Key {
			case "EMAIL_USER":
				cfg.EmailUser = s.Value
			case "EMAIL_PASSWORD":
				cfg.EmailPassword = s.Value
			case "EMAIL_IMAP_HOST":
				cfg.EmailIMAPHost = s.Value
			case "EMAIL_IMAP_PORT":
				if p, err := strconv.Atoi(s.Value); err == nil {
					cfg.EmailIMAPPort = p
				}
			case "EMAIL_POLLING_INTERVAL":
				cfg.EmailPollingInterval = s.Value
			}
		}
	}

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.Default()

	// Apply global middleware
	router.Use(corsMiddleware())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(database, cfg.JWTSecret)
	customerHandler := handlers.NewCustomerHandler(database)
	engineerHandler := handlers.NewEngineerHandler(database)
	ticketHandler := handlers.NewTicketHandler(database)
	updateHandler := handlers.NewUpdateHandler(database)

	// Initialize WhatsApp components
	wahaClient := whatsapp.NewWahaClient(cfg.WahaAPIURL, cfg.WahaAPIKey)
	messageSender := whatsapp.NewMessageSender(wahaClient, database)
	actionHandler := whatsapp.NewActionHandler(database, messageSender, wahaClient)
	whatsappHandler := handlers.NewWhatsAppHandler(database, wahaClient, messageSender, actionHandler)

	// Read email settings from SystemSetting to override env values BEFORE creating SMTP client
	var setting models.SystemSetting
	if err := database.Where("key = ?", "EMAIL_IMAP_HOST").First(&setting).Error; err == nil {
		cfg.EmailIMAPHost = setting.Value
	}
	if err := database.Where("key = ?", "EMAIL_IMAP_PORT").First(&setting).Error; err == nil {
		if portNum, err := strconv.Atoi(setting.Value); err == nil {
			cfg.EmailIMAPPort = portNum
		}
	}
	if err := database.Where("key = ?", "EMAIL_USER").First(&setting).Error; err == nil {
		cfg.EmailUser = setting.Value
	}
	if err := database.Where("key = ?", "EMAIL_PASSWORD").First(&setting).Error; err == nil {
		cfg.EmailPassword = setting.Value
	}
	if err := database.Where("key = ?", "EMAIL_POLLING_INTERVAL").First(&setting).Error; err == nil {
		cfg.EmailPollingInterval = setting.Value
	}

	// Setup SMTP fallback
	smtpUser := cfg.SMTPUser
	if smtpUser == "" {
		smtpUser = cfg.EmailUser
	}
	smtpPass := cfg.SMTPPassword
	if smtpPass == "" {
		smtpPass = cfg.EmailPassword
	}
	smtpHost := cfg.SMTPHost
	if smtpHost == "smtp.gmail.com" && cfg.EmailIMAPHost != "" && cfg.EmailIMAPHost != "imap.gmail.com" {
		// Attempt a guess if it's using default gmail but IMAP is not
		smtpHost = strings.Replace(cfg.EmailIMAPHost, "imap.", "smtp.", 1)
	}

	// Initialize email components
	domainMatcher := email.NewDomainMatcher(database)
	smtpClient := email.NewSMTPClient(
		smtpHost,
		cfg.SMTPPort,
		smtpUser,
		smtpPass,
		smtpUser, // Use email user as from address
		cfg.EmailFromName,
	)
	emailHandler := handlers.NewEmailHandler(database, domainMatcher, smtpClient, cfg)


	// Initialize and start email poller
	var emailPoller *jobs.EmailPollerJob
	if cfg.EmailUser != "" && cfg.EmailPassword != "" {
		imapClient := email.NewIMAPClient(
			cfg.EmailIMAPHost,
			cfg.EmailIMAPPort,
			cfg.EmailUser,
			cfg.EmailPassword,
		)
		pollingInterval, _ := time.ParseDuration(cfg.EmailPollingInterval)
		if pollingInterval == 0 {
			pollingInterval = 5 * time.Minute
		}
		emailPoller = jobs.NewEmailPollerJob(
			database,
			imapClient,
			emailHandler,
			domainMatcher,
			pollingInterval,
		)
		if err := emailPoller.Start(); err != nil {
			log.Printf("Warning: Failed to start email poller: %v", err)
		}
	} else {
		log.Printf("Email integration disabled: EMAIL_USER and EMAIL_PASSWORD not configured")
	}

	// Initialize report components
	reportGenerator := reports.NewReportGenerator(database, cfg.SLAHours)
	reportMailer := reports.NewReportMailer(smtpClient)
	reportRepository := reports.NewReportRepository(database)
	reportScheduler := reports.NewReportScheduler(database, reportGenerator, reportMailer, reportRepository, cfg.SLAHours)
	reportHandler := handlers.NewReportHandler(database, reportGenerator, reportRepository, reportScheduler, reportMailer)

	// Initialize dashboard component
	dashboardHandler := handlers.NewDashboardHandler(database)

	// Start report scheduler
	if err := reportScheduler.Start(); err != nil {
		log.Printf("Warning: Failed to start report scheduler: %v", err)
	}

	// Routes
	setupRoutes(router, cfg, authHandler, customerHandler, engineerHandler, ticketHandler, updateHandler, emailHandler, whatsappHandler, reportHandler, dashboardHandler)

	// Setup graceful shutdown
	go setupGracefulShutdown(emailPoller, reportScheduler)

	// Start server
	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Starting server on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRoutes(
	router *gin.Engine,
	cfg *config.Config,
	authHandler *handlers.AuthHandler,
	customerHandler *handlers.CustomerHandler,
	engineerHandler *handlers.EngineerHandler,
	ticketHandler *handlers.TicketHandler,
	updateHandler *handlers.UpdateHandler,
	emailHandler *handlers.EmailHandler,
	waHandler *handlers.WhatsAppHandler,
	reportHandler *handlers.ReportHandler,
	dashboardHandler *handlers.DashboardHandler,
) {
	// Public routes
	auth := router.Group("/api/auth")
	{
		auth.POST("/login", authHandler.Login)
	}

	// Protected routes
	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))

	protected.GET("/auth/me", authHandler.Me)

	// Dashboard routes
	dashboard := protected.Group("/dashboard")
	{
		dashboard.GET("/summary", dashboardHandler.GetSummary)
	}

	// Customer routes
	customers := protected.Group("/customers")
	{
		customers.POST("", customerHandler.CreateCustomer)
		customers.GET("", customerHandler.GetCustomers)
		customers.GET("/:id", customerHandler.GetCustomerByID)
		customers.PUT("/:id", customerHandler.UpdateCustomer)
		customers.DELETE("/:id", customerHandler.DeleteCustomer)
	}

	// Engineer routes
	engineers := protected.Group("/engineers")
	{
		engineers.POST("", engineerHandler.CreateEngineer)
		engineers.GET("", engineerHandler.GetEngineers)
		engineers.GET("/:id", engineerHandler.GetEngineerByID)
		engineers.PUT("/:id", engineerHandler.UpdateEngineer)
		engineers.DELETE("/:id", engineerHandler.DeleteEngineer)
	}

	// Ticket routes
	tickets := protected.Group("/tickets")
	{
		tickets.POST("", ticketHandler.CreateTicket)
		tickets.GET("", ticketHandler.GetTickets)
		tickets.GET("/:id", ticketHandler.GetTicketByID)
		tickets.PUT("/:id", ticketHandler.UpdateTicket)
		tickets.DELETE("/:id", ticketHandler.DeleteTicket)

		// Update/comment routes under tickets
		tickets.POST("/:id/updates", updateHandler.CreateUpdate)
		tickets.GET("/:id/updates", updateHandler.GetTicketUpdates)
	}

	// Update routes
	updates := protected.Group("/updates")
	{
		updates.DELETE("/:id", updateHandler.DeleteUpdate)
	}

	// Report routes
	reportRoutes := protected.Group("/reports")
	{
		reportRoutes.POST("/generate", reportHandler.GenerateReport)
		reportRoutes.GET("", reportHandler.ListReports)
		reportRoutes.GET("/:id", reportHandler.GetReport)
		reportRoutes.GET("/:id/download", reportHandler.DownloadReport)
		reportRoutes.POST("/:id/resend", reportHandler.ResendReport)
		reportRoutes.DELETE("/:id", reportHandler.DeleteReport)
	}

	// Email webhook (internal only - no auth required for now, but should be secured in production)
	email := router.Group("/api/email")
	{
		email.POST("/webhook", emailHandler.ProcessEmailWebhook)
	}

	// Email protected routes
	emailProtected := protected.Group("/email")
	{
		emailProtected.GET("/settings", emailHandler.GetEmailSettings)
		emailProtected.PATCH("/settings", emailHandler.UpdateEmailSettings)
		emailProtected.GET("/domain-mappings", emailHandler.GetDomainMappings)
		emailProtected.GET("/auto-reply-template", emailHandler.GetAutoReplyTemplate)
		emailProtected.PATCH("/auto-reply-template", emailHandler.UpdateAutoReplyTemplate)
		emailProtected.GET("/history", emailHandler.GetEmailHistory)
		emailProtected.POST("/test-connection", emailHandler.TestEmailConnection)
		emailProtected.POST("/sync", emailHandler.SyncEmails)
	}

	// WhatsApp routes (webhook without auth, management with auth)
	whatsappWebhook := router.Group("/api/whatsapp")
	{
		whatsappWebhook.POST("/webhook", waHandler.ProcessWebhook)
	}

	// WhatsApp routes
	wa := protected.Group("/whatsapp")
	{
		wa.POST("/sessions", waHandler.CreateSession)
		wa.GET("/sessions", waHandler.GetSessions)
		wa.GET("/sessions/:id/qr", waHandler.GetSessionQR)
		wa.POST("/sessions/:id/pairing-code", waHandler.RequestPairingCode)
		wa.POST("/sessions/:id/verify", waHandler.VerifySession)
		wa.DELETE("/sessions/:id", waHandler.DeleteSession)
		wa.POST("/engineers/:id/phone", waHandler.AddEngineerPhone)
		wa.GET("/engineer-phones", waHandler.GetEngineerPhones)
		wa.POST("/test-message", waHandler.TestMessage)
		wa.GET("/webhook/status", waHandler.GetHookStatus)
		wa.GET("/logs", waHandler.GetLogs)
	}
}

// setupGracefulShutdown sets up graceful shutdown handling
func setupGracefulShutdown(emailPoller *jobs.EmailPollerJob, reportScheduler *reports.ReportScheduler) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Printf("Shutdown signal received")

	if emailPoller != nil && emailPoller.IsRunning() {
		log.Printf("Stopping email poller...")
		emailPoller.Stop()
	}

	if reportScheduler != nil && reportScheduler.IsRunning() {
		log.Printf("Stopping report scheduler...")
		reportScheduler.Stop()
	}

	log.Printf("Shutdown complete")
	os.Exit(0)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func seedDatabase(database *gorm.DB) {
	// Check if data already exists
	var count int64
	database.Model(&models.User{}).Count(&count)
	if count > 0 {
		return // Data already seeded
	}

	// Create admin user
	hashedPassword, _ := handlers.HashPassword("admin123")
	adminUser := models.User{
		Email:    "admin@aidesK.local",
		Password: hashedPassword,
		Role:     "ADMIN",
	}
	if err := database.Create(&adminUser).Error; err != nil {
		log.Printf("Failed to seed admin user: %v", err)
	} else {
		log.Printf("Created admin user with ID: %d", adminUser.ID)
	}

	// Create test customer
	customer := models.Customer{
		Name:         "Test Company",
		Domain:       "test.example.com",
		EmailSupport: "support@test.example.com",
		IsActive:     true,
	}
	if err := database.Create(&customer).Error; err != nil {
		log.Printf("Failed to seed customer: %v", err)
	} else {
		log.Printf("Created test customer with ID: %d", customer.ID)
	}
}
