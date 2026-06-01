-- Create email_logs table
CREATE TABLE IF NOT EXISTS email_logs (
    id VARCHAR(36) PRIMARY KEY,
    email_message_id VARCHAR(255),
    sender_email VARCHAR(255) NOT NULL,
    domain_matched VARCHAR(255),
    customer_id INTEGER,
    ticket_id INTEGER,
    status VARCHAR(50) NOT NULL, -- SUCCESS, FAILED, UNKNOWN_DOMAIN
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_email_logs_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE SET NULL,
    CONSTRAINT fk_email_logs_ticket FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE SET NULL
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_email_logs_status ON email_logs(status);
CREATE INDEX IF NOT EXISTS idx_email_logs_customer_id ON email_logs(customer_id);
CREATE INDEX IF NOT EXISTS idx_email_logs_ticket_id ON email_logs(ticket_id);
CREATE INDEX IF NOT EXISTS idx_email_logs_created_at ON email_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_email_logs_sender_email ON email_logs(sender_email);
CREATE INDEX IF NOT EXISTS idx_email_logs_message_id ON email_logs(email_message_id);

-- Add email_message_id field to tickets table if it doesn't exist
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS email_message_id VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_tickets_email_message_id ON tickets(email_message_id);
