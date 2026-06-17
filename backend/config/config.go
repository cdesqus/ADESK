package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	ServerPort string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Redis
	RedisHost string
	RedisPort string

	// JWT
	JWTSecret string

	// Email - IMAP
	EmailIMAPHost string
	EmailIMAPPort int
	EmailUser     string
	EmailPassword string

	// Email - SMTP
	SMTPHost       string
	SMTPPort       int
	SMTPUser       string
	SMTPPassword   string
	EmailFromName  string

	// Email Polling
	EmailPollingInterval string

	// WhatsApp
	WahaAPIURL     string
	WahaAPIPort    string
	WahaWebhookURL string
	WahaAPIKey     string

	// Reports
	ReportGenerationTime string
	ReportMonthDay       int
	SLAHours             int
	ReportRetentionDays  int

	// Environment
	Environment string

	// AI Configuration
	OpenAIKey string
}

func Load() *Config {
	// Load .env file if it exists
	_ = godotenv.Load()

	imapPort := 993
	if port := getEnv("EMAIL_IMAP_PORT", ""); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			imapPort = p
		}
	}

	smtpPort := 587
	if port := getEnv("SMTP_PORT", ""); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			smtpPort = p
		}
	}

	slaHours := 24
	if sla := getEnv("SLA_HOURS", ""); sla != "" {
		if h, err := strconv.Atoi(sla); err == nil {
			slaHours = h
		}
	}

	reportMonthDay := 1
	if day := getEnv("REPORT_MONTH_DAY", ""); day != "" {
		if d, err := strconv.Atoi(day); err == nil && d >= 1 && d <= 31 {
			reportMonthDay = d
		}
	}

	reportRetentionDays := 365
	if days := getEnv("REPORT_RETENTION_DAYS", ""); days != "" {
		if d, err := strconv.Atoi(days); err == nil {
			reportRetentionDays = d
		}
	}

	cfg := &Config{
		// Server
		ServerPort: getEnv("SERVER_PORT", "8080"),

		// Database
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "ai_desk"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

		// Redis
		RedisHost: getEnv("REDIS_HOST", "localhost"),
		RedisPort: getEnv("REDIS_PORT", "6379"),

		// JWT
		JWTSecret: getEnv("JWT_SECRET", "your-secret-key-change-in-production"),

		// Email - IMAP
		EmailIMAPHost: getEnv("EMAIL_IMAP_HOST", "imap.gmail.com"),
		EmailIMAPPort: imapPort,
		EmailUser:     getEnv("EMAIL_USER", ""),
		EmailPassword: getEnv("EMAIL_PASSWORD", ""),

		// Email - SMTP
		SMTPHost:      getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:      smtpPort,
		SMTPUser:      getEnv("SMTP_USER", ""),
		SMTPPassword:  getEnv("SMTP_PASSWORD", ""),
		EmailFromName: getEnv("EMAIL_FROM_NAME", "IDE SOLUSI INTEGRASI Support"),

		// Email Polling
		EmailPollingInterval: getEnv("EMAIL_POLLING_INTERVAL", "5m"),

		// WhatsApp - Waha Plus
		WahaAPIURL:     getEnv("WAHA_API_URL", "http://waha:3000"),
		WahaAPIPort:    getEnv("WAHA_API_PORT", "3000"),
		WahaWebhookURL: getEnv("WAHA_WEBHOOK_URL", "http://go-app:8000/api/whatsapp/webhook"),
		WahaAPIKey:     getEnv("WAHA_API_KEY", ""),

		// Reports
		ReportGenerationTime: getEnv("REPORT_GENERATION_TIME", "08:00"),
		ReportMonthDay:       reportMonthDay,
		SLAHours:             slaHours,
		ReportRetentionDays:  reportRetentionDays,

		// Environment
		Environment: getEnv("ENVIRONMENT", "development"),
		OpenAIKey:   getEnv("OPENAI_API_KEY", ""),
	}

	// Validate required env vars in production
	if cfg.Environment == "production" {
		if cfg.JWTSecret == "your-secret-key-change-in-production" {
			log.Printf("CRITICAL WARNING: Using default JWT_SECRET in production mode! This is a massive security risk.")
		}
		if cfg.EmailUser == "" || cfg.EmailPassword == "" {
			log.Printf("WARNING: EMAIL_USER and EMAIL_PASSWORD are not set in production. Email integration will be disabled.")
		}
		if cfg.SMTPUser == "" || cfg.SMTPPassword == "" {
			log.Printf("WARNING: SMTP_USER and SMTP_PASSWORD are not set in production. Outbound emails will fail.")
		}
	}

	return cfg
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost,
		c.DBPort,
		c.DBUser,
		c.DBPassword,
		c.DBName,
		c.DBSSLMode,
	)
}

func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
