package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// StringArray represents a PostgreSQL text[] array
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *StringArray) Scan(value interface{}) error {
	bytes, _ := value.([]byte)
	return json.Unmarshal(bytes, &a)
}

// User represents a system user
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Email     string         `gorm:"uniqueIndex;not null" json:"email"`
	Password  string         `gorm:"not null" json:"-"`
	Role      string         `gorm:"not null;default:'VIEWER'" json:"role"` // ADMIN, SUPPORT, ENGINEER, VIEWER
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Customer represents a customer organization
type Customer struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Name         string         `gorm:"not null;index" json:"name"`
	Domain       string         `gorm:"uniqueIndex" json:"domain"`
	EmailSupport string         `json:"email_support"`
	Address      string         `gorm:"type:text" json:"address"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Engineers []Engineer        `gorm:"foreignKey:CustomerID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"engineers,omitempty"`
	Tickets   []Ticket          `gorm:"foreignKey:CustomerID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"tickets,omitempty"`
	WAGroups  []CustomerWAGroup `gorm:"foreignKey:CustomerID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"wa_groups,omitempty"`
}

// SystemSetting represents a dynamic system configuration key-value pair
type SystemSetting struct {
	Key       string    `gorm:"primaryKey" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Engineer represents a support engineer
type Engineer struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	CustomerID    *uint          `gorm:"index" json:"customer_id,omitempty"`
	Name          string         `gorm:"not null;index" json:"name"`
	Email         string         `gorm:"not null" json:"email"`
	WhatsappNumber string         `json:"whatsapp_number"`
	Skills        datatypes.JSONSlice[string] `gorm:"type:jsonb;default:'[]'" json:"skills"`
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Customer Customer `gorm:"foreignKey:CustomerID" json:"-"`
	Tickets  []Ticket `gorm:"foreignKey:EngineerID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"tickets,omitempty"`
	Updates  []Update `gorm:"foreignKey:EngineerID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"updates,omitempty"`
}

// Ticket represents a support ticket
type Ticket struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	TicketNumber   string         `gorm:"uniqueIndex" json:"ticket_number"`
	CustomerID     uint           `gorm:"index;not null" json:"customer_id"`
	EngineerID     *uint          `gorm:"index" json:"engineer_id"`
	Title          string         `gorm:"not null;index" json:"title"`
	Description    string         `gorm:"type:text" json:"description"`
	Status         string         `gorm:"index;default:'OPEN'" json:"status"` // OPEN, IN_PROGRESS, RESOLVED, CLOSED
	Priority       string         `gorm:"default:'MEDIUM'" json:"priority"`   // LOW, MEDIUM, HIGH, URGENT
	Category       string         `json:"category"`
	Source         string         `json:"source"` // EMAIL, CHAT, PHONE, WEB, WHATSAPP
	EmailFrom      string         `json:"email_from"`
	EmailMessageID string         `json:"email_message_id"`
	WhatsappFrom   string         `json:"whatsapp_from"`
	WhatsappSessionID string       `json:"whatsapp_session_id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	ResolvedAt     *time.Time     `json:"resolved_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Customer Customer `gorm:"foreignKey:CustomerID" json:"-"`
	Engineer *User    `gorm:"foreignKey:EngineerID" json:"engineer,omitempty"`
	Updates  []Update `gorm:"foreignKey:TicketID" json:"updates,omitempty"`
}

// BeforeCreate generates the auto-increment ticket number
func (t *Ticket) BeforeCreate(tx *gorm.DB) (err error) {
	if t.TicketNumber == "" {
		yearMonth := time.Now().Format("2006-01") // e.g. "2026-06"

		// Use advisory locking for the ticket sequence generator so concurrent
		// ticket creation does not select the same next value.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", yearMonth).Error; err != nil {
			return err
		}

		var maxSeq int
		if err := tx.Raw(
			"SELECT COALESCE(MAX(CAST(SPLIT_PART(ticket_number, '-', 3) AS INTEGER)), 0) FROM tickets WHERE ticket_number LIKE ?",
			yearMonth+"-%",
		).Scan(&maxSeq).Error; err != nil {
			return err
		}

		sequence := maxSeq + 1
		t.TicketNumber = fmt.Sprintf("%s-%03d", yearMonth, sequence)
	}
	return
}

// Update represents a status change or comment on a ticket
type Update struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	TicketID   uint           `gorm:"index;not null" json:"ticket_id"`
	EngineerID *uint          `gorm:"index" json:"engineer_id"`
	Message    string         `gorm:"type:text;not null" json:"message"`
	ActionType string         `gorm:"default:'COMMENT'" json:"action_type"` // COMMENT, STATUS_CHANGE, ASSIGNED, REOPENED
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Ticket   Ticket    `gorm:"foreignKey:TicketID" json:"-"`
	Engineer *Engineer `gorm:"foreignKey:EngineerID" json:"engineer,omitempty"`
}

// EmailLog represents an email processing log entry
type EmailLog struct {
	ID             string         `gorm:"primaryKey" json:"id"`
	EmailMessageID string         `json:"email_message_id"`
	SenderEmail    string         `json:"sender_email"`
	DomainMatched  string         `json:"domain_matched"`
	CustomerID     *uint          `gorm:"index" json:"customer_id"`
	TicketID       *uint          `gorm:"index" json:"ticket_id"`
	Status         string         `gorm:"index" json:"status"` // SUCCESS, FAILED, UNKNOWN_DOMAIN
	ErrorMessage   string         `gorm:"type:text" json:"error_message"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`

	// Relations
	Customer *Customer `gorm:"foreignKey:CustomerID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-"`
	Ticket   *Ticket   `gorm:"foreignKey:TicketID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-"`
}

// WhatsAppSession represents a WhatsApp business session
type WhatsAppSession struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	SessionName  string    `gorm:"uniqueIndex;not null" json:"session_name"`
	PhoneNumber  string    `json:"phone_number"`
	Status       string    `gorm:"default:'PENDING'" json:"status"` // PENDING, CONNECTED, DISCONNECTED
	QRCode       string    `gorm:"type:text" json:"qr_code"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// EngineerWAPhone represents a WhatsApp phone linked to an engineer
type EngineerWAPhone struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	EngineerID   uint      `gorm:"index;not null" json:"engineer_id"`
	PhoneNumber  string    `gorm:"not null" json:"phone_number"`
	GroupID      string    `json:"group_id"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`

	// Relations
	Engineer Engineer `gorm:"foreignKey:EngineerID;constraint:OnDelete:CASCADE" json:"-"`
}

// WhatsAppLog represents WhatsApp message activity log
type WhatsAppLog struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	SessionName    string    `gorm:"index;not null" json:"session_name"`
	MessageID      string    `gorm:"index" json:"message_id"`
	FromPhone      string    `gorm:"index;not null" json:"from_phone"`
	ToPhone        string    `gorm:"index" json:"to_phone"`
	Body           string    `gorm:"type:text" json:"body"`
	MessageType    string    `json:"message_type"` // text, image, audio, video
	Direction      string    `json:"direction"` // inbound, outbound
	Status         string    `json:"status"` // received, delivered, read, failed
	CustomerID     *uint     `gorm:"index" json:"customer_id"`
	TicketID       *uint     `gorm:"index" json:"ticket_id"`
	ErrorMessage   string    `gorm:"type:text" json:"error_message"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Relations
	Customer *Customer `gorm:"foreignKey:CustomerID;constraint:OnDelete:SET NULL" json:"-"`
	Ticket   *Ticket   `gorm:"foreignKey:TicketID;constraint:OnDelete:SET NULL" json:"-"`
}

// MonthlyReport represents a generated monthly report
type MonthlyReport struct {
	ID            string     `gorm:"primaryKey" json:"id"`
	CustomerID    uint       `gorm:"index;not null" json:"customer_id"`
	Month         int        `gorm:"not null" json:"month"`          // 1-12
	Year          int        `gorm:"not null" json:"year"`
	CSVData       []byte     `gorm:"type:bytea" json:"-"`
	PDFData       []byte     `gorm:"type:bytea" json:"-"`
	GeneratedAt   time.Time  `json:"generated_at"`
	SentAt        *time.Time `json:"sent_at"`
	SentToEmails  StringArray `gorm:"type:jsonb;default:'[]'" json:"sent_to_emails"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Relations
	Customer Customer `gorm:"foreignKey:CustomerID" json:"-"`
}

// ReportMetrics contains all metrics for a monthly report
type ReportMetrics struct {
	TotalTickets           int                `json:"total_tickets"`
	ResolvedTickets        int                `json:"resolved_tickets"`
	OpenTickets            int                `json:"open_tickets"`
	InProgressTickets      int                `json:"in_progress_tickets"`
	AverageResolutionTime  float64            `json:"average_resolution_time"` // hours
	ByStatus               map[string]int     `json:"by_status"`               // OPEN: 5, RESOLVED: 20, etc
	ByPriority             map[string]int     `json:"by_priority"`             // LOW: 10, HIGH: 15, etc
	BySource               map[string]int     `json:"by_source"`               // EMAIL: 15, WHATSAPP: 10, WEB: 5
	EngineerStats          []EngineerStat     `json:"engineer_stats"`
	SLACompliance          float64            `json:"sla_compliance"` // percentage (0-100)
}

// EngineerStat contains per-engineer performance metrics
type EngineerStat struct {
	EngineerID     uint    `json:"engineer_id"`
	Name           string  `json:"name"`
	TicketsHandled int     `json:"tickets_handled"`
	AvgTime        float64 `json:"avg_time"`           // hours
	ResolutionRate float64 `json:"resolution_rate"`    // percentage
}

// TicketSummary represents a ticket in report form
type TicketSummary struct {
	ID           uint       `json:"id"`
	Title        string     `json:"title"`
	CreatedAt    time.Time  `json:"created_at"`
	ResolvedAt   *time.Time `json:"resolved_at"`
	TimeToResolve float64   `json:"time_to_resolve"` // hours
	Status       string     `json:"status"`
	Engineer     string     `json:"engineer"`
	Source       string     `json:"source"`
}

// ReportData contains all data for rendering a report
type ReportData struct {
	CustomerName  string           `json:"customer_name"`
	Month         string           `json:"month"`      // "June 2026"
	MonthNum      int              `json:"month_num"`
	Year          int              `json:"year"`
	GeneratedAt   time.Time        `json:"generated_at"`
	Metrics       ReportMetrics    `json:"metrics"`
	TicketsList   []TicketSummary  `json:"tickets_list"`
}

// CustomerWAGroup maps a WhatsApp Group ID to a Customer
type CustomerWAGroup struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CustomerID *uint     `gorm:"index" json:"customer_id"`
	GroupID    string    `gorm:"uniqueIndex;not null" json:"group_id"` // 12036302@g.us
	GroupName  string    `json:"group_name"`
	IsSupport  bool      `gorm:"default:false" json:"is_support"` // True for IDE-SUPPORT internal group
	CreatedAt  time.Time `json:"created_at"`

	Customer *Customer `gorm:"foreignKey:CustomerID;constraint:OnDelete:SET NULL" json:"-"`
}

// TableName specifies table names for gorm
func (User) TableName() string {
	return "users"
}

func (Customer) TableName() string {
	return "customers"
}

func (Engineer) TableName() string {
	return "engineers"
}

func (Ticket) TableName() string {
	return "tickets"
}

func (Update) TableName() string {
	return "updates"
}

func (EmailLog) TableName() string {
	return "email_logs"
}

func (WhatsAppSession) TableName() string {
	return "whatsapp_sessions"
}

func (EngineerWAPhone) TableName() string {
	return "engineer_wa_phones"
}

func (WhatsAppLog) TableName() string {
	return "whatsapp_logs"
}

func (MonthlyReport) TableName() string {
	return "monthly_reports"
}
