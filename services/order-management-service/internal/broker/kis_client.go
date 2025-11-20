package broker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sony/gobreaker"
)

// KISClient handles communication with Korea Investment & Securities API
type KISClient struct {
	baseURL    string
	appKey     string
	appSecret  string
	accountNo  string
	httpClient *http.Client
	breaker    *gobreaker.CircuitBreaker
	maxRetries int
	retryDelay time.Duration
}

// Config holds the KIS client configuration
type Config struct {
	BaseURL    string
	AppKey     string
	AppSecret  string
	AccountNo  string
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
}

// OrderRequest represents a KIS order request
type OrderRequest struct {
	Symbol   string  `json:"symbol"`
	Side     string  `json:"side"`      // "BUY" or "SELL"
	Type     string  `json:"type"`      // "LIMIT" or "MARKET"
	Price    float64 `json:"price"`     // 0 for market orders
	Quantity int     `json:"quantity"`
}

// OrderResponse represents a KIS order response
type OrderResponse struct {
	OrderID        string    `json:"order_id"`
	Status         string    `json:"status"`
	FilledPrice    float64   `json:"filled_price,omitempty"`
	FilledQuantity int       `json:"filled_quantity,omitempty"`
	Message        string    `json:"message,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// OrderStatus represents order status from KIS
type OrderStatus struct {
	OrderID        string    `json:"order_id"`
	Status         string    `json:"status"`
	FilledPrice    float64   `json:"filled_price"`
	FilledQuantity int       `json:"filled_quantity"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// NewKISClient creates a new KIS API client
func NewKISClient(config *Config) (*KISClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// Create circuit breaker
	breakerSettings := gobreaker.Settings{
		Name:        "KIS-API",
		MaxRequests: 3,
		Interval:    time.Minute,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Warn().
				Str("circuit_breaker", name).
				Str("from", from.String()).
				Str("to", to.String()).
				Msg("Circuit breaker state changed")
		},
	}

	client := &KISClient{
		baseURL:    config.BaseURL,
		appKey:     config.AppKey,
		appSecret:  config.AppSecret,
		accountNo:  config.AccountNo,
		httpClient: httpClient,
		breaker:    gobreaker.NewCircuitBreaker(breakerSettings),
		maxRetries: config.MaxRetries,
		retryDelay: config.RetryDelay,
	}

	log.Info().
		Str("base_url", config.BaseURL).
		Msg("KIS client initialized")

	return client, nil
}

// SubmitOrder submits an order to KIS
func (c *KISClient) SubmitOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error) {
	var lastErr error

	// Retry logic with exponential backoff
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := c.retryDelay * time.Duration(1<<uint(attempt-1))
			log.Info().
				Int("attempt", attempt).
				Dur("delay", delay).
				Msg("Retrying order submission")
			time.Sleep(delay)
		}

		// Execute through circuit breaker
		result, err := c.breaker.Execute(func() (interface{}, error) {
			return c.submitOrderInternal(ctx, req)
		})

		if err == nil {
			return result.(*OrderResponse), nil
		}

		lastErr = err
		log.Warn().
			Err(err).
			Int("attempt", attempt).
			Msg("Order submission failed")
	}

	return nil, fmt.Errorf("order submission failed after %d attempts: %w", c.maxRetries, lastErr)
}

// submitOrderInternal performs the actual HTTP request to submit an order
func (c *KISClient) submitOrderInternal(ctx context.Context, req *OrderRequest) (*OrderResponse, error) {
	// Convert order type to KIS format
	kisOrderType := c.convertOrderType(req.Type)
	if kisOrderType == "" {
		return nil, fmt.Errorf("unsupported order type: %s", req.Type)
	}

	// Build request payload (KIS-specific format)
	payload := map[string]interface{}{
		"CANO":       c.accountNo[:8],        // 계좌번호 앞 8자리
		"ACNT_PRDT_CD": c.accountNo[8:],     // 계좌번호 뒤 2자리
		"PDNO":       req.Symbol,             // 종목코드
		"ORD_DVSN":   kisOrderType,           // 주문구분
		"ORD_QTY":    fmt.Sprintf("%d", req.Quantity), // 주문수량
		"ORD_UNPR":   fmt.Sprintf("%.0f", req.Price),  // 주문단가
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Determine endpoint based on order side
	endpoint := "/uapi/domestic-stock/v1/trading/order-cash"
	if req.Side == "SELL" {
		endpoint = "/uapi/domestic-stock/v1/trading/order-cash"
	}

	url := c.baseURL + endpoint

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	c.addHeaders(httpReq, req.Side)

	// Sign request
	if err := c.signRequest(httpReq, jsonPayload); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// Execute request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KIS API error: status=%d, body=%s", httpResp.StatusCode, string(body))
	}

	// Parse response
	var kisResp struct {
		RetCode string `json:"rt_cd"`
		MsgCode string `json:"msg_cd"`
		Msg1    string `json:"msg1"`
		Output  struct {
			OrderNo    string `json:"ODNO"` // 주문번호
			OrderTime  string `json:"ORD_TMD"` // 주문시각
		} `json:"output"`
	}

	if err := json.Unmarshal(body, &kisResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check response code
	if kisResp.RetCode != "0" {
		return nil, fmt.Errorf("KIS API error: code=%s, msg=%s", kisResp.MsgCode, kisResp.Msg1)
	}

	// Build response
	response := &OrderResponse{
		OrderID:   kisResp.Output.OrderNo,
		Status:    "SENT",
		Message:   kisResp.Msg1,
		Timestamp: time.Now(),
	}

	log.Info().
		Str("order_id", response.OrderID).
		Str("symbol", req.Symbol).
		Msg("Order submitted successfully")

	return response, nil
}

// CancelOrder cancels an existing order
func (c *KISClient) CancelOrder(ctx context.Context, orderID string) (*OrderResponse, error) {
	endpoint := "/uapi/domestic-stock/v1/trading/order-rvsecncl"
	url := c.baseURL + endpoint

	payload := map[string]interface{}{
		"CANO":        c.accountNo[:8],
		"ACNT_PRDT_CD": c.accountNo[8:],
		"ORGN_ODNO":   orderID, // 원주문번호
		"RVSE_CNCL_DVSN_CD": "02", // 취소구분 (02=취소)
		"ORD_QTY":     "0",
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addHeaders(httpReq, "CANCEL")
	if err := c.signRequest(httpReq, jsonPayload); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KIS API error: status=%d, body=%s", httpResp.StatusCode, string(body))
	}

	response := &OrderResponse{
		OrderID:   orderID,
		Status:    "CANCELLED",
		Timestamp: time.Now(),
	}

	log.Info().Str("order_id", orderID).Msg("Order cancelled successfully")
	return response, nil
}

// GetOrderStatus retrieves the status of an order
func (c *KISClient) GetOrderStatus(ctx context.Context, orderID string) (*OrderStatus, error) {
	// In production, this would query KIS API for order status
	// For now, return a stub implementation
	log.Debug().Str("order_id", orderID).Msg("Getting order status")

	// This is a simplified stub - in production, implement actual KIS API call
	return &OrderStatus{
		OrderID:   orderID,
		Status:    "FILLED",
		UpdatedAt: time.Now(),
	}, nil
}

// addHeaders adds required headers to the request
func (c *KISClient) addHeaders(req *http.Request, side string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("appkey", c.appKey)
	req.Header.Set("appsecret", c.appSecret)

	// Transaction ID varies by operation
	var trId string
	switch side {
	case "BUY":
		trId = "TTTC0802U" // 현금매수주문
	case "SELL":
		trId = "TTTC0801U" // 현금매도주문
	case "CANCEL":
		trId = "TTTC0803U" // 주문취소
	default:
		trId = "TTTC0802U"
	}
	req.Header.Set("tr_id", trId)
}

// signRequest signs the request using HMAC
func (c *KISClient) signRequest(req *http.Request, body []byte) error {
	// KIS uses a specific signing format
	// In production, follow exact KIS API documentation for signing
	timestamp := time.Now().Unix()

	// Create signature
	message := fmt.Sprintf("%d%s", timestamp, string(body))
	h := hmac.New(sha256.New, []byte(c.appSecret))
	h.Write([]byte(message))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", timestamp))

	return nil
}

// convertOrderType converts internal order type to KIS format
func (c *KISClient) convertOrderType(orderType string) string {
	switch orderType {
	case "MARKET":
		return "01" // 시장가
	case "LIMIT":
		return "00" // 지정가
	default:
		return ""
	}
}
