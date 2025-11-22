-- Create tick_data table
CREATE TABLE IF NOT EXISTS tick_data (
    time TIMESTAMPTZ NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    price NUMERIC(20, 8) NOT NULL,
    volume BIGINT NOT NULL,
    change NUMERIC(20, 8),
    change_rate NUMERIC(10, 4)
);

-- Create hypertable for tick_data
SELECT create_hypertable('tick_data', 'time', if_not_exists => TRUE);

-- Create index on tick_data for efficient symbol-based queries
CREATE INDEX IF NOT EXISTS idx_tick_symbol_time ON tick_data (symbol, time DESC);

-- Create additional index for price queries
CREATE INDEX IF NOT EXISTS idx_tick_symbol_price ON tick_data (symbol, price);

-- Create orderbook table
CREATE TABLE IF NOT EXISTS orderbook (
    time TIMESTAMPTZ NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    bids JSONB NOT NULL,
    asks JSONB NOT NULL
);

-- Create hypertable for orderbook
SELECT create_hypertable('orderbook', 'time', if_not_exists => TRUE);

-- Create index on orderbook for efficient symbol-based queries
CREATE INDEX IF NOT EXISTS idx_orderbook_symbol_time ON orderbook (symbol, time DESC);

-- Create GIN index for JSONB queries
CREATE INDEX IF NOT EXISTS idx_orderbook_bids ON orderbook USING GIN (bids);
CREATE INDEX IF NOT EXISTS idx_orderbook_asks ON orderbook USING GIN (asks);

-- Set compression policy (compress data older than 7 days)
SELECT add_compression_policy('tick_data', INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_compression_policy('orderbook', INTERVAL '7 days', if_not_exists => TRUE);

-- Set retention policy (delete data older than 90 days)
SELECT add_retention_policy('tick_data', INTERVAL '90 days', if_not_exists => TRUE);
SELECT add_retention_policy('orderbook', INTERVAL '90 days', if_not_exists => TRUE);

-- Create continuous aggregate for 1-minute OHLCV
CREATE MATERIALIZED VIEW IF NOT EXISTS tick_1min
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 minute', time) AS bucket,
    symbol,
    FIRST(price, time) AS open,
    MAX(price) AS high,
    MIN(price) AS low,
    LAST(price, time) AS close,
    SUM(volume) AS volume,
    COUNT(*) AS tick_count
FROM tick_data
GROUP BY bucket, symbol
WITH NO DATA;

-- Create refresh policy for continuous aggregate
SELECT add_continuous_aggregate_policy('tick_1min',
    start_offset => INTERVAL '2 hours',
    end_offset => INTERVAL '1 minute',
    schedule_interval => INTERVAL '1 minute',
    if_not_exists => TRUE
);

-- Create continuous aggregate for 5-minute OHLCV
CREATE MATERIALIZED VIEW IF NOT EXISTS tick_5min
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('5 minutes', time) AS bucket,
    symbol,
    FIRST(price, time) AS open,
    MAX(price) AS high,
    MIN(price) AS low,
    LAST(price, time) AS close,
    SUM(volume) AS volume,
    COUNT(*) AS tick_count
FROM tick_data
GROUP BY bucket, symbol
WITH NO DATA;

SELECT add_continuous_aggregate_policy('tick_5min',
    start_offset => INTERVAL '10 hours',
    end_offset => INTERVAL '5 minutes',
    schedule_interval => INTERVAL '5 minutes',
    if_not_exists => TRUE
);

-- Create continuous aggregate for 1-hour OHLCV
CREATE MATERIALIZED VIEW IF NOT EXISTS tick_1hour
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS bucket,
    symbol,
    FIRST(price, time) AS open,
    MAX(price) AS high,
    MIN(price) AS low,
    LAST(price, time) AS close,
    SUM(volume) AS volume,
    COUNT(*) AS tick_count
FROM tick_data
GROUP BY bucket, symbol
WITH NO DATA;

SELECT add_continuous_aggregate_policy('tick_1hour',
    start_offset => INTERVAL '24 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE
);

-- Grant permissions (adjust user as needed)
-- GRANT SELECT, INSERT ON tick_data TO aipx;
-- GRANT SELECT, INSERT ON orderbook TO aipx;
-- GRANT SELECT ON tick_1min TO aipx;
-- GRANT SELECT ON tick_5min TO aipx;
-- GRANT SELECT ON tick_1hour TO aipx;
