-- WhatsApp sessions table
CREATE TABLE IF NOT EXISTS whatsapp_sessions (
    id VARCHAR(36) PRIMARY KEY,
    session_name VARCHAR(255) UNIQUE NOT NULL,
    phone_number VARCHAR(20),
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    qr_code TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_sessions_name ON whatsapp_sessions(session_name);
CREATE INDEX IF NOT EXISTS idx_whatsapp_sessions_status ON whatsapp_sessions(status);
CREATE INDEX IF NOT EXISTS idx_whatsapp_sessions_phone ON whatsapp_sessions(phone_number);

-- Engineer WhatsApp phones table
CREATE TABLE IF NOT EXISTS engineer_wa_phones (
    id VARCHAR(36) PRIMARY KEY,
    engineer_id INTEGER NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    group_id VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_engineer_wa_phones FOREIGN KEY (engineer_id) REFERENCES engineers(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_engineer_wa_phones_engineer_id ON engineer_wa_phones(engineer_id);
CREATE INDEX IF NOT EXISTS idx_engineer_wa_phones_phone ON engineer_wa_phones(phone_number);
CREATE INDEX IF NOT EXISTS idx_engineer_wa_phones_active ON engineer_wa_phones(is_active);

-- WhatsApp logs table
CREATE TABLE IF NOT EXISTS whatsapp_logs (
    id VARCHAR(36) PRIMARY KEY,
    session_name VARCHAR(255) NOT NULL,
    message_id VARCHAR(255),
    from_phone VARCHAR(20) NOT NULL,
    to_phone VARCHAR(20),
    body TEXT,
    message_type VARCHAR(50),
    direction VARCHAR(20),
    status VARCHAR(50),
    customer_id INTEGER,
    ticket_id INTEGER,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_whatsapp_logs_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE SET NULL,
    CONSTRAINT fk_whatsapp_logs_ticket FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_logs_session ON whatsapp_logs(session_name);
CREATE INDEX IF NOT EXISTS idx_whatsapp_logs_message_id ON whatsapp_logs(message_id);
CREATE INDEX IF NOT EXISTS idx_whatsapp_logs_from_phone ON whatsapp_logs(from_phone);
CREATE INDEX IF NOT EXISTS idx_whatsapp_logs_direction ON whatsapp_logs(direction);
CREATE INDEX IF NOT EXISTS idx_whatsapp_logs_status ON whatsapp_logs(status);
CREATE INDEX IF NOT EXISTS idx_whatsapp_logs_customer_id ON whatsapp_logs(customer_id);
CREATE INDEX IF NOT EXISTS idx_whatsapp_logs_ticket_id ON whatsapp_logs(ticket_id);
CREATE INDEX IF NOT EXISTS idx_whatsapp_logs_created_at ON whatsapp_logs(created_at);

-- Add WhatsApp fields to tickets table if they don't exist
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS whatsapp_from VARCHAR(20);
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS whatsapp_session_id VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_tickets_whatsapp_from ON tickets(whatsapp_from);
CREATE INDEX IF NOT EXISTS idx_tickets_whatsapp_session_id ON tickets(whatsapp_session_id);
