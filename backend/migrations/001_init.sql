-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'VIEWER',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- Customers table
CREATE TABLE IF NOT EXISTS customers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) UNIQUE,
    email_support VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX idx_customers_name ON customers(name);
CREATE INDEX idx_customers_domain ON customers(domain);
CREATE INDEX idx_customers_is_active ON customers(is_active);
CREATE INDEX idx_customers_deleted_at ON customers(deleted_at);

-- Engineers table
CREATE TABLE IF NOT EXISTS engineers (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    whatsapp_number VARCHAR(20),
    skills JSONB DEFAULT '[]',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (customer_id) REFERENCES customers(id) ON UPDATE CASCADE ON DELETE SET NULL
);

CREATE INDEX idx_engineers_customer_id ON engineers(customer_id);
CREATE INDEX idx_engineers_name ON engineers(name);
CREATE INDEX idx_engineers_is_active ON engineers(is_active);
CREATE INDEX idx_engineers_deleted_at ON engineers(deleted_at);

-- Tickets table
CREATE TABLE IF NOT EXISTS tickets (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    engineer_id INTEGER,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'OPEN',
    priority VARCHAR(50) DEFAULT 'MEDIUM',
    category VARCHAR(100),
    source VARCHAR(50),
    email_from VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP NULL,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (customer_id) REFERENCES customers(id) ON UPDATE CASCADE ON DELETE SET NULL,
    FOREIGN KEY (engineer_id) REFERENCES engineers(id) ON UPDATE CASCADE ON DELETE SET NULL
);

CREATE INDEX idx_tickets_customer_id ON tickets(customer_id);
CREATE INDEX idx_tickets_engineer_id ON tickets(engineer_id);
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_priority ON tickets(priority);
CREATE INDEX idx_tickets_title ON tickets(title);
CREATE INDEX idx_tickets_created_at ON tickets(created_at);
CREATE INDEX idx_tickets_deleted_at ON tickets(deleted_at);

-- Updates table
CREATE TABLE IF NOT EXISTS updates (
    id SERIAL PRIMARY KEY,
    ticket_id INTEGER NOT NULL,
    engineer_id INTEGER,
    message TEXT NOT NULL,
    action_type VARCHAR(50) DEFAULT 'COMMENT',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON UPDATE CASCADE ON DELETE CASCADE,
    FOREIGN KEY (engineer_id) REFERENCES engineers(id) ON UPDATE CASCADE ON DELETE SET NULL
);

CREATE INDEX idx_updates_ticket_id ON updates(ticket_id);
CREATE INDEX idx_updates_engineer_id ON updates(engineer_id);
CREATE INDEX idx_updates_action_type ON updates(action_type);
CREATE INDEX idx_updates_created_at ON updates(created_at);
CREATE INDEX idx_updates_deleted_at ON updates(deleted_at);

-- Insert default admin user (password: admin123)
INSERT INTO users (email, password, role)
VALUES ('admin@aidesK.local', '$2a$10$eImiTXuWVxfaHNYY0iNAeuK2kRSoap1lplLnw7GyNAL8/LewKciPm', 'ADMIN')
ON CONFLICT (email) DO NOTHING;
