-- Initialize database for WhatsApp/WhatsMe0ow usage
-- This will create any necessary extensions and initial setup

-- Create the whatsmeow schema if needed
CREATE SCHEMA IF NOT EXISTS whatsmeow;

-- Grant permissions
GRANT ALL PRIVILEGES ON SCHEMA whatsmeow TO tennex;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA whatsmeow TO tennex;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA whatsmeow TO tennex;

-- Log the initialization
\echo 'WhatsApp database initialized successfully'
