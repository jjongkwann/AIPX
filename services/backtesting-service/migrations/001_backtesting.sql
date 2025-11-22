-- Backtesting Service Database Schema
-- Version: 001
-- Description: Initial schema for backtest sessions, trades, and results

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Backtest Sessions Table
CREATE TABLE IF NOT EXISTS backtest_sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    strategy_id UUID NOT NULL,
    name VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),

    -- Configuration
    initial_capital DECIMAL(18, 2) NOT NULL CHECK (initial_capital > 0),
    commission_rate DECIMAL(8, 6) NOT NULL DEFAULT 0.0003,
    tax_rate DECIMAL(8, 6) NOT NULL DEFAULT 0.0023,

    -- Time range
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    symbols TEXT[] NOT NULL,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    CONSTRAINT valid_date_range CHECK (start_date < end_date)
);

-- Create indexes for session queries
CREATE INDEX IF NOT EXISTS idx_backtest_sessions_user_id ON backtest_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_backtest_sessions_strategy_id ON backtest_sessions(strategy_id);
CREATE INDEX IF NOT EXISTS idx_backtest_sessions_status ON backtest_sessions(status);
CREATE INDEX IF NOT EXISTS idx_backtest_sessions_created_at ON backtest_sessions(created_at DESC);

-- Backtest Results Table
CREATE TABLE IF NOT EXISTS backtest_results (
    result_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL UNIQUE,

    -- Final state
    final_equity DECIMAL(18, 2) NOT NULL,
    final_cash DECIMAL(18, 2) NOT NULL,

    -- P&L metrics
    total_pnl DECIMAL(18, 2) NOT NULL,
    realized_pnl DECIMAL(18, 2) NOT NULL,
    unrealized_pnl DECIMAL(18, 2) NOT NULL,
    total_commission DECIMAL(18, 2) NOT NULL,

    -- Performance metrics
    total_return DECIMAL(10, 6) NOT NULL,
    annualized_return DECIMAL(10, 6),
    sharpe_ratio DECIMAL(10, 6),
    sortino_ratio DECIMAL(10, 6),
    max_drawdown DECIMAL(10, 6) NOT NULL,
    max_drawdown_duration_days INTEGER,

    -- Trade statistics
    total_trades INTEGER NOT NULL DEFAULT 0,
    winning_trades INTEGER NOT NULL DEFAULT 0,
    losing_trades INTEGER NOT NULL DEFAULT 0,
    win_rate DECIMAL(5, 4),
    avg_win DECIMAL(18, 2),
    avg_loss DECIMAL(18, 2),
    profit_factor DECIMAL(10, 4),

    -- Risk metrics
    volatility DECIMAL(10, 6),
    var_95 DECIMAL(18, 2),
    cvar_95 DECIMAL(18, 2),

    -- Equity curve (stored as JSONB array)
    equity_curve JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_result_session FOREIGN KEY (session_id)
        REFERENCES backtest_sessions(session_id)
        ON DELETE CASCADE
);

-- Create index for result queries
CREATE INDEX IF NOT EXISTS idx_backtest_results_session_id ON backtest_results(session_id);

-- Backtest Trades Table
CREATE TABLE IF NOT EXISTS backtest_trades (
    trade_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,

    -- Trade details
    timestamp TIMESTAMPTZ NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    side VARCHAR(4) NOT NULL CHECK (side IN ('buy', 'sell')),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    price DECIMAL(18, 4) NOT NULL CHECK (price > 0),
    commission DECIMAL(18, 4) NOT NULL,

    -- P&L (only for sells)
    pnl DECIMAL(18, 2),

    -- Position after trade
    position_quantity INTEGER NOT NULL DEFAULT 0,
    position_avg_price DECIMAL(18, 4),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_trade_session FOREIGN KEY (session_id)
        REFERENCES backtest_sessions(session_id)
        ON DELETE CASCADE
);

-- Create indexes for trade queries
CREATE INDEX IF NOT EXISTS idx_backtest_trades_session_id ON backtest_trades(session_id);
CREATE INDEX IF NOT EXISTS idx_backtest_trades_symbol ON backtest_trades(symbol);
CREATE INDEX IF NOT EXISTS idx_backtest_trades_timestamp ON backtest_trades(timestamp);
CREATE INDEX IF NOT EXISTS idx_backtest_trades_session_timestamp ON backtest_trades(session_id, timestamp);

-- Backtest Positions Table (final positions snapshot)
CREATE TABLE IF NOT EXISTS backtest_positions (
    position_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    avg_price DECIMAL(18, 4) NOT NULL CHECK (avg_price > 0),
    market_value DECIMAL(18, 2),
    unrealized_pnl DECIMAL(18, 2),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_position_session FOREIGN KEY (session_id)
        REFERENCES backtest_sessions(session_id)
        ON DELETE CASCADE,
    CONSTRAINT unique_session_symbol UNIQUE (session_id, symbol)
);

-- Create index for position queries
CREATE INDEX IF NOT EXISTS idx_backtest_positions_session_id ON backtest_positions(session_id);

-- Comments for documentation
COMMENT ON TABLE backtest_sessions IS 'Backtest session configurations and status';
COMMENT ON TABLE backtest_results IS 'Aggregated backtest performance results';
COMMENT ON TABLE backtest_trades IS 'Individual trade records during backtest';
COMMENT ON TABLE backtest_positions IS 'Final position snapshot after backtest';

COMMENT ON COLUMN backtest_results.sharpe_ratio IS 'Risk-adjusted return (excess return / volatility)';
COMMENT ON COLUMN backtest_results.sortino_ratio IS 'Downside risk-adjusted return';
COMMENT ON COLUMN backtest_results.max_drawdown IS 'Maximum peak-to-trough decline';
COMMENT ON COLUMN backtest_results.profit_factor IS 'Gross profit / gross loss ratio';
COMMENT ON COLUMN backtest_results.var_95 IS 'Value at Risk at 95% confidence';
COMMENT ON COLUMN backtest_results.cvar_95 IS 'Conditional VaR (Expected Shortfall) at 95%';
