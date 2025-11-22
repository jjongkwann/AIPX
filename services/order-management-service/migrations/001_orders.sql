-- Order Management Service Database Schema
-- Version: 001
-- Description: Initial schema for orders and audit log

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Orders table
-- Stores all order information including lifecycle states
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    strategy_id VARCHAR(50),
    symbol VARCHAR(20) NOT NULL,
    side VARCHAR(4) NOT NULL CHECK (side IN ('BUY', 'SELL')),
    order_type VARCHAR(10) NOT NULL CHECK (order_type IN ('LIMIT', 'MARKET')),
    price NUMERIC(15, 2),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    status VARCHAR(20) NOT NULL CHECK (status IN ('PENDING', 'SENT', 'FILLED', 'REJECTED', 'CANCELLED')),
    broker_order_id VARCHAR(50),
    filled_price NUMERIC(15, 2),
    filled_quantity INTEGER CHECK (filled_quantity >= 0 AND filled_quantity <= quantity),
    reject_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

-- Indexes for efficient querying
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_symbol ON orders(symbol);
CREATE INDEX idx_orders_created_at ON orders(created_at DESC);
CREATE INDEX idx_orders_user_status ON orders(user_id, status);
CREATE INDEX idx_orders_broker_order_id ON orders(broker_order_id) WHERE broker_order_id IS NOT NULL;

-- Order audit log
-- Tracks all status changes and important events for orders
CREATE TABLE IF NOT EXISTS order_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    reason TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

-- Indexes for audit log
CREATE INDEX idx_audit_order_id ON order_audit_log(order_id);
CREATE INDEX idx_audit_created_at ON order_audit_log(created_at DESC);

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update updated_at on orders table
CREATE TRIGGER update_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to automatically create audit log entries on status changes
CREATE OR REPLACE FUNCTION create_order_audit_log()
RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'UPDATE' AND OLD.status != NEW.status) THEN
        INSERT INTO order_audit_log (order_id, status, reason)
        VALUES (NEW.id, NEW.status, NEW.reject_reason);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-create audit log on status change
CREATE TRIGGER audit_order_status_change
    AFTER UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION create_order_audit_log();

-- Comments for documentation
COMMENT ON TABLE orders IS 'Stores all trading orders with their lifecycle states';
COMMENT ON TABLE order_audit_log IS 'Audit trail for order status changes and events';
COMMENT ON COLUMN orders.user_id IS 'Reference to user who created the order';
COMMENT ON COLUMN orders.strategy_id IS 'Optional reference to the trading strategy';
COMMENT ON COLUMN orders.symbol IS 'Trading symbol (e.g., AAPL, TSLA)';
COMMENT ON COLUMN orders.side IS 'Order side: BUY or SELL';
COMMENT ON COLUMN orders.order_type IS 'Order type: LIMIT or MARKET';
COMMENT ON COLUMN orders.price IS 'Limit price (NULL for market orders)';
COMMENT ON COLUMN orders.quantity IS 'Number of shares to buy/sell';
COMMENT ON COLUMN orders.status IS 'Current order status: PENDING, SENT, FILLED, REJECTED, CANCELLED';
COMMENT ON COLUMN orders.broker_order_id IS 'Broker-assigned order ID after submission';
COMMENT ON COLUMN orders.filled_price IS 'Actual execution price';
COMMENT ON COLUMN orders.filled_quantity IS 'Number of shares actually filled';
COMMENT ON COLUMN orders.reject_reason IS 'Reason for rejection if status is REJECTED';
