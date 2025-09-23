-- Initialize backend database schema
-- This extends the WhatsApp initialization with backend-specific tables

-- Create the backend schema
CREATE SCHEMA IF NOT EXISTS backend;

-- Grant permissions to tennex user
GRANT ALL PRIVILEGES ON SCHEMA backend TO tennex;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA backend TO tennex;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA backend TO tennex;

-- Set search path to include backend schema
ALTER USER tennex SET search_path = backend, whatsmeow, public;

-- Log the initialization
\echo 'Backend database schema initialized successfully'
