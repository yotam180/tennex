-- Add users table for authentication
-- This extends the initial schema with user management

-- Users table: store user accounts and authentication data
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      TEXT UNIQUE NOT NULL,
    email         TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    full_name     TEXT,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_users_username ON users (username) WHERE is_active = true;
CREATE INDEX idx_users_email ON users (email) WHERE is_active = true;
CREATE INDEX idx_users_created_at ON users (created_at);

-- Add constraints and checks
ALTER TABLE users ADD CONSTRAINT users_username_check 
    CHECK (length(username) >= 3 AND length(username) <= 30 AND username ~ '^[a-zA-Z0-9_]+$');

ALTER TABLE users ADD CONSTRAINT users_email_check 
    CHECK (email ~ '^[A-Za-z0-9._+%-]+@[A-Za-z0-9.-]+[.][A-Za-z]+$');

ALTER TABLE users ADD CONSTRAINT users_password_hash_check 
    CHECK (length(password_hash) >= 1);

-- Trigger to automatically update updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation  
COMMENT ON TABLE users IS 'User accounts for authentication and authorization';
COMMENT ON COLUMN users.id IS 'Unique user identifier';
COMMENT ON COLUMN users.username IS 'Unique username for login (alphanumeric + underscore)';
COMMENT ON COLUMN users.email IS 'User email address (unique)';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hash of user password';
COMMENT ON COLUMN users.is_active IS 'Whether the user account is active';
