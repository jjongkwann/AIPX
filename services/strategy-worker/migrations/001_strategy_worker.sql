-- Strategy Worker Database Schema
-- Version: 001
-- Description: Initial schema for strategy execution tracking

BEGIN;

-- Strategy Executions Table
CREATE TABLE IF NOT EXISTS strategy_executions (
    execution_id UUID PRIMARY KEY,
    strategy_id UUID NOT NULL,
    user_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    config JSONB NOT NULL,
    positions JSONB DEFAULT '{}',
    total_pnl DECIMAL(15,2) DEFAULT 0.00,
    realized_pnl DECIMAL(15,2) DEFAULT 0.00,
    unrealized_pnl DECIMAL(15,2) DEFAULT 0.00,
    total_trades INT DEFAULT 0,
    winning_trades INT DEFAULT 0,
    losing_trades INT DEFAULT 0,
    total_exposure DECIMAL(15,2) DEFAULT 0.00,
    daily_pnl DECIMAL(15,2) DEFAULT 0.00,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    stopped_at TIMESTAMPTZ,
    last_rebalanced_at TIMESTAMPTZ,
    last_checked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_status CHECK (status IN ('active', 'paused', 'stopped', 'completed', 'error'))
);

-- Indexes for strategy_executions
CREATE INDEX idx_strategy_executions_user_id ON strategy_executions(user_id);
CREATE INDEX idx_strategy_executions_strategy_id ON strategy_executions(strategy_id);
CREATE INDEX idx_strategy_executions_status ON strategy_executions(status);
CREATE INDEX idx_strategy_executions_started_at ON strategy_executions(started_at);

-- Execution Orders Table
CREATE TABLE IF NOT EXISTS execution_orders (
    order_id UUID PRIMARY KEY,
    execution_id UUID NOT NULL REFERENCES strategy_executions(execution_id) ON DELETE CASCADE,
    oms_order_id VARCHAR(50),
    symbol VARCHAR(20) NOT NULL,
    side VARCHAR(4) NOT NULL,
    order_type VARCHAR(10) NOT NULL DEFAULT 'LIMIT',
    quantity INT NOT NULL,
    price DECIMAL(15,2),
    filled_quantity INT DEFAULT 0,
    filled_price DECIMAL(15,2),
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    error_message TEXT,
    submitted_at TIMESTAMPTZ,
    filled_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_side CHECK (side IN ('BUY', 'SELL')),
    CONSTRAINT valid_order_type CHECK (order_type IN ('LIMIT', 'MARKET')),
    CONSTRAINT valid_status CHECK (status IN ('PENDING', 'ACCEPTED', 'REJECTED', 'PARTIALLY_FILLED', 'FILLED', 'CANCELLED')),
    CONSTRAINT positive_quantity CHECK (quantity > 0)
);

-- Indexes for execution_orders
CREATE INDEX idx_execution_orders_execution_id ON execution_orders(execution_id);
CREATE INDEX idx_execution_orders_oms_order_id ON execution_orders(oms_order_id);
CREATE INDEX idx_execution_orders_symbol ON execution_orders(symbol);
CREATE INDEX idx_execution_orders_status ON execution_orders(status);
CREATE INDEX idx_execution_orders_created_at ON execution_orders(created_at);

-- Position Snapshots Table (for historical tracking)
CREATE TABLE IF NOT EXISTS position_snapshots (
    snapshot_id UUID PRIMARY KEY,
    execution_id UUID NOT NULL REFERENCES strategy_executions(execution_id) ON DELETE CASCADE,
    positions JSONB NOT NULL,
    total_value DECIMAL(15,2) NOT NULL,
    total_pnl DECIMAL(15,2) NOT NULL,
    unrealized_pnl DECIMAL(15,2) NOT NULL,
    snapshot_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for position_snapshots
CREATE INDEX idx_position_snapshots_execution_id ON position_snapshots(execution_id);
CREATE INDEX idx_position_snapshots_snapshot_at ON position_snapshots(snapshot_at);

-- Risk Events Table (log risk limit violations)
CREATE TABLE IF NOT EXISTS risk_events (
    event_id UUID PRIMARY KEY,
    execution_id UUID NOT NULL REFERENCES strategy_executions(execution_id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    description TEXT NOT NULL,
    data JSONB,
    action_taken VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_severity CHECK (severity IN ('INFO', 'WARNING', 'CRITICAL'))
);

-- Indexes for risk_events
CREATE INDEX idx_risk_events_execution_id ON risk_events(execution_id);
CREATE INDEX idx_risk_events_event_type ON risk_events(event_type);
CREATE INDEX idx_risk_events_severity ON risk_events(severity);
CREATE INDEX idx_risk_events_created_at ON risk_events(created_at);

-- Strategy Metrics View (for monitoring)
CREATE OR REPLACE VIEW strategy_metrics AS
SELECT
    se.execution_id,
    se.strategy_id,
    se.user_id,
    se.status,
    se.total_pnl,
    se.realized_pnl,
    se.unrealized_pnl,
    se.total_trades,
    se.winning_trades,
    se.losing_trades,
    CASE
        WHEN se.total_trades > 0
        THEN ROUND((se.winning_trades::DECIMAL / se.total_trades::DECIMAL) * 100, 2)
        ELSE 0.00
    END AS win_rate,
    se.total_exposure,
    se.daily_pnl,
    se.started_at,
    se.stopped_at,
    EXTRACT(EPOCH FROM (COALESCE(se.stopped_at, NOW()) - se.started_at)) / 3600 AS runtime_hours,
    COUNT(eo.order_id) AS total_orders,
    COUNT(CASE WHEN eo.status = 'FILLED' THEN 1 END) AS filled_orders,
    COUNT(CASE WHEN eo.status = 'REJECTED' THEN 1 END) AS rejected_orders,
    COUNT(CASE WHEN eo.status = 'CANCELLED' THEN 1 END) AS cancelled_orders
FROM strategy_executions se
LEFT JOIN execution_orders eo ON se.execution_id = eo.execution_id
GROUP BY se.execution_id, se.strategy_id, se.user_id, se.status,
         se.total_pnl, se.realized_pnl, se.unrealized_pnl,
         se.total_trades, se.winning_trades, se.losing_trades,
         se.total_exposure, se.daily_pnl, se.started_at, se.stopped_at;

-- Update timestamp trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply update triggers
CREATE TRIGGER update_strategy_executions_updated_at
    BEFORE UPDATE ON strategy_executions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_execution_orders_updated_at
    BEFORE UPDATE ON execution_orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation
COMMENT ON TABLE strategy_executions IS 'Tracks active strategy executions and their state';
COMMENT ON TABLE execution_orders IS 'Records all orders submitted for strategy execution';
COMMENT ON TABLE position_snapshots IS 'Historical snapshots of positions for performance tracking';
COMMENT ON TABLE risk_events IS 'Audit log of risk events and violations';
COMMENT ON VIEW strategy_metrics IS 'Aggregated metrics view for strategy monitoring';

COMMIT;
