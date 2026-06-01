-- Create monthly_reports table for archiving generated reports
CREATE TABLE IF NOT EXISTS monthly_reports (
    id VARCHAR(36) PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    month INTEGER NOT NULL,
    year INTEGER NOT NULL,
    csv_data BYTEA,
    pdf_data BYTEA,
    generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    sent_at TIMESTAMP NULL,
    sent_to_emails JSONB DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_reports_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT unique_customer_month_year UNIQUE (customer_id, month, year)
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_monthly_reports_customer_id ON monthly_reports(customer_id);
CREATE INDEX IF NOT EXISTS idx_monthly_reports_month_year ON monthly_reports(month, year);
CREATE INDEX IF NOT EXISTS idx_monthly_reports_generated_at ON monthly_reports(generated_at);
CREATE INDEX IF NOT EXISTS idx_monthly_reports_sent_at ON monthly_reports(sent_at);
CREATE INDEX IF NOT EXISTS idx_monthly_reports_created_at ON monthly_reports(created_at);
