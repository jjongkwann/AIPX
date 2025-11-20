package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockKISServer represents a mock KIS API server
type MockKISServer struct {
	Server   *httptest.Server
	Behavior MockBehavior
}

// MockBehavior defines the behavior of the mock server
type MockBehavior int

const (
	BehaviorSuccess MockBehavior = iota
	BehaviorInsufficientFunds
	BehaviorInvalidSymbol
	BehaviorTimeout
	BehaviorRateLimit
	BehaviorServerError
)

// NewMockKISServer creates a new mock KIS API server
func NewMockKISServer(t *testing.T, behavior MockBehavior) *MockKISServer {
	mock := &MockKISServer{
		Behavior: behavior,
	}

	mock.Server = httptest.NewServer(http.HandlerFunc(mock.handleRequest))

	return mock
}

// handleRequest handles incoming requests to the mock server
func (m *MockKISServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch m.Behavior {
	case BehaviorSuccess:
		m.handleSuccess(w, r)
	case BehaviorInsufficientFunds:
		m.handleInsufficientFunds(w, r)
	case BehaviorInvalidSymbol:
		m.handleInvalidSymbol(w, r)
	case BehaviorTimeout:
		// Simulate timeout by not responding
		return
	case BehaviorRateLimit:
		m.handleRateLimit(w, r)
	case BehaviorServerError:
		m.handleServerError(w, r)
	}
}

// handleSuccess simulates a successful order submission
func (m *MockKISServer) handleSuccess(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"rt_cd":  "0",
		"msg_cd": "00000",
		"msg1":   "주문이 정상적으로 처리되었습니다.",
		"output": map[string]interface{}{
			"ODNO":    "0000000001",
			"ORD_TMD": "153000",
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleInsufficientFunds simulates insufficient funds error
func (m *MockKISServer) handleInsufficientFunds(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"rt_cd":  "1",
		"msg_cd": "40150000",
		"msg1":   "주문가능금액을 초과하였습니다.",
		"output": map[string]interface{}{},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleInvalidSymbol simulates invalid symbol error
func (m *MockKISServer) handleInvalidSymbol(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"rt_cd":  "1",
		"msg_cd": "40140000",
		"msg1":   "종목코드가 올바르지 않습니다.",
		"output": map[string]interface{}{},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleRateLimit simulates rate limit exceeded
func (m *MockKISServer) handleRateLimit(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"rt_cd":  "1",
		"msg_cd": "EGW00123",
		"msg1":   "API 호출 한도를 초과하였습니다.",
		"output": map[string]interface{}{},
	}

	w.WriteHeader(http.StatusTooManyRequests)
	json.NewEncoder(w).Encode(resp)
}

// handleServerError simulates server error
func (m *MockKISServer) handleServerError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Internal Server Error"))
}

// Close closes the mock server
func (m *MockKISServer) Close() {
	if m.Server != nil {
		m.Server.Close()
	}
}

// SetBehavior changes the behavior of the mock server
func (m *MockKISServer) SetBehavior(behavior MockBehavior) {
	m.Behavior = behavior
}
