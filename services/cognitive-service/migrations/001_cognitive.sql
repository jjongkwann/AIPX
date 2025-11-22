-- Cognitive Service Database Schema
-- Version: 001
-- Description: Initial schema for user profiles, chat sessions, and strategies

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- User Profiles Table
CREATE TABLE IF NOT EXISTS user_profiles (
    user_id UUID PRIMARY KEY,
    risk_tolerance VARCHAR(20) NOT NULL CHECK (risk_tolerance IN ('conservative', 'moderate', 'aggressive')),
    capital BIGINT NOT NULL CHECK (capital > 0),
    preferred_sectors TEXT[],
    investment_horizon VARCHAR(20) CHECK (investment_horizon IN ('short', 'medium', 'long')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index on user_id for fast lookups
CREATE INDEX IF NOT EXISTS idx_user_profiles_user_id ON user_profiles(user_id);

-- Chat Sessions Table
CREATE TABLE IF NOT EXISTS chat_sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    message_count INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT fk_session_user FOREIGN KEY (user_id)
        REFERENCES user_profiles(user_id)
        ON DELETE CASCADE
);

-- Create indexes for session queries
CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_id ON chat_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_started_at ON chat_sessions(started_at DESC);

-- Strategies Table
CREATE TABLE IF NOT EXISTS strategies (
    strategy_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    config JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'approved', 'active', 'paused', 'terminated')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_at TIMESTAMPTZ,
    activated_at TIMESTAMPTZ,
    CONSTRAINT fk_strategy_user FOREIGN KEY (user_id)
        REFERENCES user_profiles(user_id)
        ON DELETE CASCADE
);

-- Create indexes for strategy queries
CREATE INDEX IF NOT EXISTS idx_strategies_user_id ON strategies(user_id);
CREATE INDEX IF NOT EXISTS idx_strategies_status ON strategies(status);
CREATE INDEX IF NOT EXISTS idx_strategies_created_at ON strategies(created_at DESC);

-- Create GIN index on config JSONB for fast queries
CREATE INDEX IF NOT EXISTS idx_strategies_config ON strategies USING GIN (config);

-- Chat Messages Table (for history)
CREATE TABLE IF NOT EXISTS chat_messages (
    message_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_message_session FOREIGN KEY (session_id)
        REFERENCES chat_sessions(session_id)
        ON DELETE CASCADE
);

-- Create indexes for message queries
CREATE INDEX IF NOT EXISTS idx_chat_messages_session_id ON chat_messages(session_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_timestamp ON chat_messages(timestamp DESC);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update updated_at
CREATE TRIGGER update_user_profiles_updated_at
    BEFORE UPDATE ON user_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation
COMMENT ON TABLE user_profiles IS 'User investment profiles and preferences';
COMMENT ON TABLE chat_sessions IS 'Chat session tracking for conversations';
COMMENT ON TABLE strategies IS 'Generated trading strategy configurations';
COMMENT ON TABLE chat_messages IS 'Chat message history for sessions';

COMMENT ON COLUMN user_profiles.risk_tolerance IS 'User risk tolerance level: conservative, moderate, or aggressive';
COMMENT ON COLUMN user_profiles.capital IS 'Investment capital in USD';
COMMENT ON COLUMN user_profiles.preferred_sectors IS 'Array of preferred investment sectors';
COMMENT ON COLUMN user_profiles.investment_horizon IS 'Investment time horizon: short, medium, or long';

COMMENT ON COLUMN strategies.config IS 'JSONB containing complete strategy configuration';
COMMENT ON COLUMN strategies.status IS 'Strategy lifecycle status';
