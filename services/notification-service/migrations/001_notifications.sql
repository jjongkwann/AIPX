-- Migration: 001_notifications.sql
-- Description: Create notification preferences and history tables
-- Date: 2025-11-19

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Notification preferences table
-- Stores user notification channel preferences and configurations
CREATE TABLE IF NOT EXISTS notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('slack', 'telegram', 'email')),
    enabled BOOLEAN DEFAULT TRUE,
    config JSONB,  -- Channel-specific config (webhook URL, chat ID, email address, etc)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Ensure one preference per user per channel
    UNIQUE(user_id, channel)
);

CREATE INDEX idx_notif_pref_user_id ON notification_preferences(user_id);
CREATE INDEX idx_notif_pref_channel ON notification_preferences(channel);
CREATE INDEX idx_notif_pref_enabled ON notification_preferences(enabled) WHERE enabled = TRUE;

-- Notification history table
-- Stores all sent notifications for audit and debugging
CREATE TABLE IF NOT EXISTS notification_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('slack', 'telegram', 'email')),
    event_type VARCHAR(50) NOT NULL CHECK (
        event_type IN ('order_filled', 'order_rejected', 'order_cancelled',
                      'risk_alert', 'position_opened', 'position_closed',
                      'system_alert', 'maintenance_notice')
    ),
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('sent', 'failed', 'pending', 'retry')),
    error_message TEXT,
    retry_count INT DEFAULT 0,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_notif_hist_user_id ON notification_history(user_id);
CREATE INDEX idx_notif_hist_created ON notification_history(created_at DESC);
CREATE INDEX idx_notif_hist_status ON notification_history(status);
CREATE INDEX idx_notif_hist_event_type ON notification_history(event_type);
CREATE INDEX idx_notif_hist_user_status ON notification_history(user_id, status);

-- Create updated_at trigger for notification_preferences
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_notification_preferences_updated_at
    BEFORE UPDATE ON notification_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create indexes for JSON queries
CREATE INDEX idx_notif_pref_config_webhook ON notification_preferences
    USING gin (config) WHERE config ? 'webhook_url';

CREATE INDEX idx_notif_hist_payload ON notification_history
    USING gin (payload);

-- Comments for documentation
COMMENT ON TABLE notification_preferences IS 'User notification channel preferences and configurations';
COMMENT ON TABLE notification_history IS 'Audit log of all sent notifications';
COMMENT ON COLUMN notification_preferences.config IS 'JSON config: {webhook_url, chat_id, email, api_token, etc}';
COMMENT ON COLUMN notification_history.payload IS 'Original event data that triggered the notification';
COMMENT ON COLUMN notification_history.retry_count IS 'Number of retry attempts for failed notifications';
