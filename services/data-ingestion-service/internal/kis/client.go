package kis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"data-ingestion-service/internal/config"
	"github.com/rs/zerolog/log"
)

// ConnectionStatus represents the WebSocket connection status
type ConnectionStatus int

const (
	StatusDisconnected ConnectionStatus = iota
	StatusConnecting
	StatusConnected
	StatusReconnecting
)

// MessageHandler is a function that processes incoming messages
type MessageHandler func(ctx context.Context, msgType int, data []byte) error

// Client represents a KIS WebSocket client
type Client struct {
	config  *config.KISConfig
	auth    *AuthManager
	handler MessageHandler

	conn          *websocket.Conn
	status        ConnectionStatus
	mu            sync.RWMutex
	messageChan   chan []byte
	reconnectChan chan struct{}

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewClient creates a new KIS WebSocket client
func NewClient(cfg *config.KISConfig, auth *AuthManager, handler MessageHandler) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		config:        cfg,
		auth:          auth,
		handler:       handler,
		status:        StatusDisconnected,
		messageChan:   make(chan []byte, cfg.MessageBufferSize),
		reconnectChan: make(chan struct{}, 1),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start establishes the WebSocket connection and starts message processing
func (c *Client) Start() error {
	if err := c.connect(); err != nil {
		return fmt.Errorf("initial connection failed: %w", err)
	}

	// Start message reader
	c.wg.Add(1)
	go c.readMessages()

	// Start message processor
	c.wg.Add(1)
	go c.processMessages()

	// Start heartbeat
	c.wg.Add(1)
	go c.heartbeat()

	// Start reconnect handler
	c.wg.Add(1)
	go c.handleReconnect()

	log.Info().Msg("KIS WebSocket client started")

	return nil
}

// Stop gracefully stops the client
func (c *Client) Stop() error {
	log.Info().Msg("stopping KIS WebSocket client")

	c.cancel()

	// Close connection
	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()

	// Wait for goroutines to finish
	c.wg.Wait()

	close(c.messageChan)

	log.Info().Msg("KIS WebSocket client stopped")

	return nil
}

// GetStatus returns the current connection status
func (c *Client) GetStatus() ConnectionStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

// connect establishes a WebSocket connection
func (c *Client) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.status = StatusConnecting

	token, err := c.auth.GetToken()
	if err != nil {
		return fmt.Errorf("failed to get auth token: %w", err)
	}

	log.Info().Str("url", c.config.WebSocketURL).Msg("connecting to KIS WebSocket")

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, resp, err := dialer.Dial(c.config.WebSocketURL, nil)
	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	// Send authentication message
	if err := c.sendAuth(conn, token); err != nil {
		conn.Close()
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	c.conn = conn
	c.status = StatusConnected

	log.Info().Msg("successfully connected to KIS WebSocket")

	return nil
}

// sendAuth sends authentication message to KIS
func (c *Client) sendAuth(conn *websocket.Conn, token string) error {
	authMsg := map[string]interface{}{
		"header": map[string]string{
			"approval_key": token,
			"custtype":     "P",
			"tr_type":      "1",
			"content-type": "utf-8",
		},
		"body": map[string]interface{}{
			"input": map[string]string{
				"tr_id":   "H0STCNT0",
				"tr_key":  "",
			},
		},
	}

	return conn.WriteJSON(authMsg)
}

// readMessages reads messages from WebSocket
func (c *Client) readMessages() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			msgType, message, err := conn.ReadMessage()
			if err != nil {
				log.Error().Err(err).Msg("failed to read message")
				c.triggerReconnect()
				continue
			}

			select {
			case c.messageChan <- message:
			case <-c.ctx.Done():
				return
			default:
				log.Warn().Msg("message buffer full, dropping message")
			}

			_ = msgType // For future use
		}
	}
}

// processMessages processes incoming messages
func (c *Client) processMessages() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case message, ok := <-c.messageChan:
			if !ok {
				return
			}

			if err := c.handler(c.ctx, websocket.TextMessage, message); err != nil {
				log.Error().Err(err).Msg("failed to handle message")
			}
		}
	}
}

// heartbeat sends periodic ping messages
func (c *Client) heartbeat() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			conn := c.conn
			status := c.status
			c.mu.RUnlock()

			if status != StatusConnected || conn == nil {
				continue
			}

			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Error().Err(err).Msg("failed to send ping")
				c.triggerReconnect()
			}
		}
	}
}

// handleReconnect handles reconnection logic
func (c *Client) handleReconnect() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.reconnectChan:
			c.reconnect()
		}
	}
}

// triggerReconnect triggers a reconnection attempt
func (c *Client) triggerReconnect() {
	select {
	case c.reconnectChan <- struct{}{}:
	default:
		// Reconnect already triggered
	}
}

// reconnect attempts to reconnect with exponential backoff
func (c *Client) reconnect() {
	c.mu.Lock()
	if c.status == StatusReconnecting {
		c.mu.Unlock()
		return
	}
	c.status = StatusReconnecting

	// Close existing connection
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()

	log.Warn().Msg("starting reconnection process")

	delay := c.config.ReconnectDelay
	maxRetries := c.config.ReconnectRetries

	for attempt := 1; attempt <= maxRetries; attempt++ {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		log.Info().
			Int("attempt", attempt).
			Int("max_retries", maxRetries).
			Dur("delay", delay).
			Msg("attempting to reconnect")

		if err := c.connect(); err != nil {
			log.Error().
				Err(err).
				Int("attempt", attempt).
				Msg("reconnection attempt failed")

			if attempt < maxRetries {
				time.Sleep(delay)
				delay *= 2 // Exponential backoff
				if delay > 60*time.Second {
					delay = 60 * time.Second
				}
			}
			continue
		}

		log.Info().
			Int("attempt", attempt).
			Msg("successfully reconnected")
		return
	}

	log.Error().
		Int("max_retries", maxRetries).
		Msg("failed to reconnect after max retries")

	c.mu.Lock()
	c.status = StatusDisconnected
	c.mu.Unlock()
}

// Subscribe subscribes to specific market data
func (c *Client) Subscribe(symbols []string, dataType string) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	// Prepare subscription message
	subMsg := map[string]interface{}{
		"header": map[string]string{
			"tr_type": "3",
		},
		"body": map[string]interface{}{
			"tr_id": dataType,
			"tr_key": symbols,
		},
	}

	if err := conn.WriteJSON(subMsg); err != nil {
		return fmt.Errorf("failed to send subscription: %w", err)
	}

	log.Info().
		Strs("symbols", symbols).
		Str("data_type", dataType).
		Msg("subscribed to market data")

	return nil
}
