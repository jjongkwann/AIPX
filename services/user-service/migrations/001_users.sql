-- Migration 001: Users and Security Infrastructure
-- Created: 2025-01-19
-- Description: Creates users, api_keys, and refresh_tokens tables with security features

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    email_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Constraints
    CONSTRAINT email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);

-- Indexes for users table
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_active ON users(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_users_created_at ON users(created_at DESC);

-- API Keys table (for broker credentials)
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    broker VARCHAR(20) NOT NULL,  -- 'KIS', 'eBest', etc
    key_encrypted TEXT NOT NULL,
    secret_encrypted TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,

    -- Constraints
    CONSTRAINT valid_broker CHECK (broker IN ('KIS', 'eBest', 'NH', 'KB')),
    CONSTRAINT unique_user_broker UNIQUE (user_id, broker)
);

-- Indexes for api_keys table
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_broker ON api_keys(broker);
CREATE INDEX idx_api_keys_active ON api_keys(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_api_keys_user_broker ON api_keys(user_id, broker);

-- Refresh tokens table
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    ip_address INET,
    user_agent TEXT,

    -- Constraints
    CONSTRAINT valid_expiry CHECK (expires_at > created_at)
);

-- Indexes for refresh_tokens table
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at) WHERE revoked_at IS NULL;
CREATE INDEX idx_refresh_tokens_active ON refresh_tokens(user_id, expires_at) WHERE revoked_at IS NULL;

-- Audit log table for security events
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    event_type VARCHAR(50) NOT NULL,  -- 'login', 'logout', 'password_change', 'api_key_created', etc
    event_data JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),

    -- Constraints
    CONSTRAINT valid_event_type CHECK (event_type IN (
        'login', 'logout', 'register', 'password_change', 'password_reset',
        'api_key_created', 'api_key_deleted', 'email_verified',
        'account_locked', 'account_unlocked', 'failed_login'
    ))
);

-- Indexes for audit_logs table
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_user_event ON audit_logs(user_id, event_type, created_at DESC);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for automatic timestamp updates
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to clean up expired refresh tokens
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS void AS $$
BEGIN
    DELETE FROM refresh_tokens
    WHERE expires_at < NOW() - INTERVAL '7 days';
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE users IS 'User accounts with secure password storage';
COMMENT ON TABLE api_keys IS 'Encrypted broker API credentials';
COMMENT ON TABLE refresh_tokens IS 'JWT refresh tokens with revocation support';
COMMENT ON TABLE audit_logs IS 'Security audit trail for user actions';

COMMENT ON COLUMN users.password_hash IS 'Argon2id hashed password';
COMMENT ON COLUMN api_keys.key_encrypted IS 'AES-256-GCM encrypted API key';
COMMENT ON COLUMN api_keys.secret_encrypted IS 'AES-256-GCM encrypted API secret';
COMMENT ON COLUMN refresh_tokens.token_hash IS 'SHA-256 hashed refresh token';
