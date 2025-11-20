package testutil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// MockSlackServer creates a mock Slack webhook server
type MockSlackServer struct {
	server         *httptest.Server
	receivedMsgs   []map[string]interface{}
	mu             sync.Mutex
	shouldFail     bool
	failureCode    int
	failureMessage string
}

// NewMockSlackServer creates a new mock Slack webhook server
func NewMockSlackServer(t *testing.T) *MockSlackServer {
	mock := &MockSlackServer{
		receivedMsgs: make([]map[string]interface{}, 0),
		failureCode:  http.StatusInternalServerError,
	}

	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		defer mock.mu.Unlock()

		// Check method
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Simulate failure if configured
		if mock.shouldFail {
			w.WriteHeader(mock.failureCode)
			if mock.failureMessage != "" {
				w.Write([]byte(mock.failureMessage))
			}
			return
		}

		// Read body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Parse JSON
		var msg map[string]interface{}
		if err := json.Unmarshal(body, &msg); err != nil {
			t.Errorf("Failed to parse JSON: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Store message
		mock.receivedMsgs = append(mock.receivedMsgs, msg)

		// Return success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	return mock
}

// URL returns the mock server URL
func (m *MockSlackServer) URL() string {
	return m.server.URL
}

// Close closes the mock server
func (m *MockSlackServer) Close() {
	m.server.Close()
}

// GetReceivedMessages returns all received messages
func (m *MockSlackServer) GetReceivedMessages() []map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	msgs := make([]map[string]interface{}, len(m.receivedMsgs))
	copy(msgs, m.receivedMsgs)
	return msgs
}

// GetMessageCount returns the number of received messages
func (m *MockSlackServer) GetMessageCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.receivedMsgs)
}

// Reset resets the received messages
func (m *MockSlackServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.receivedMsgs = make([]map[string]interface{}, 0)
}

// SetShouldFail configures the server to fail requests
func (m *MockSlackServer) SetShouldFail(fail bool, code int, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
	m.failureCode = code
	m.failureMessage = message
}

// MockTelegramServer creates a mock Telegram Bot API server
type MockTelegramServer struct {
	server       *httptest.Server
	receivedMsgs []map[string]interface{}
	mu           sync.Mutex
	shouldFail   bool
	botToken     string
}

// NewMockTelegramServer creates a new mock Telegram Bot API server
func NewMockTelegramServer(t *testing.T, botToken string) *MockTelegramServer {
	mock := &MockTelegramServer{
		receivedMsgs: make([]map[string]interface{}, 0),
		botToken:     botToken,
	}

	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		defer mock.mu.Unlock()

		// Check if it's getMe endpoint
		if strings.HasSuffix(r.URL.Path, "/getMe") {
			if mock.shouldFail {
				response := map[string]interface{}{
					"ok":          false,
					"error_code":  401,
					"description": "Unauthorized",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			response := map[string]interface{}{
				"ok": true,
				"result": map[string]interface{}{
					"id":         123456789,
					"is_bot":     true,
					"first_name": "Test Bot",
					"username":   "test_bot",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Check if it's sendMessage endpoint
		if strings.HasSuffix(r.URL.Path, "/sendMessage") {
			if mock.shouldFail {
				response := map[string]interface{}{
					"ok":          false,
					"error_code":  400,
					"description": "Bad Request: chat not found",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			// Read body
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("Failed to read request body: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Parse JSON
			var msg map[string]interface{}
			if err := json.Unmarshal(body, &msg); err != nil {
				t.Errorf("Failed to parse JSON: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Store message
			mock.receivedMsgs = append(mock.receivedMsgs, msg)

			// Return success
			response := map[string]interface{}{
				"ok": true,
				"result": map[string]interface{}{
					"message_id": len(mock.receivedMsgs),
					"chat": map[string]interface{}{
						"id":   msg["chat_id"],
						"type": "private",
					},
					"text": msg["text"],
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	return mock
}

// URL returns the mock server URL
func (m *MockTelegramServer) URL() string {
	return m.server.URL
}

// Close closes the mock server
func (m *MockTelegramServer) Close() {
	m.server.Close()
}

// GetReceivedMessages returns all received messages
func (m *MockTelegramServer) GetReceivedMessages() []map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	msgs := make([]map[string]interface{}, len(m.receivedMsgs))
	copy(msgs, m.receivedMsgs)
	return msgs
}

// GetMessageCount returns the number of received messages
func (m *MockTelegramServer) GetMessageCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.receivedMsgs)
}

// Reset resets the received messages
func (m *MockTelegramServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.receivedMsgs = make([]map[string]interface{}, 0)
}

// SetShouldFail configures the server to fail requests
func (m *MockTelegramServer) SetShouldFail(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
}

// MockSMTPServer creates a mock SMTP server
type MockSMTPServer struct {
	Address       string
	receivedMails []string
	mu            sync.Mutex
	shouldFail    bool
}

// NewMockSMTPServer creates a new mock SMTP server (simplified version)
func NewMockSMTPServer() *MockSMTPServer {
	return &MockSMTPServer{
		Address:       "localhost:2525",
		receivedMails: make([]string, 0),
	}
}

// GetReceivedMails returns all received emails
func (m *MockSMTPServer) GetReceivedMails() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	mails := make([]string, len(m.receivedMails))
	copy(mails, m.receivedMails)
	return mails
}

// GetMailCount returns the number of received emails
func (m *MockSMTPServer) GetMailCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.receivedMails)
}

// Reset resets the received emails
func (m *MockSMTPServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.receivedMails = make([]string, 0)
}

// SetShouldFail configures the server to fail
func (m *MockSMTPServer) SetShouldFail(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
}

// RecordMail records an email (called by test code)
func (m *MockSMTPServer) RecordMail(mail string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.receivedMails = append(m.receivedMails, mail)
}

// GetBotAPIURL returns the Telegram Bot API URL for the mock server
func (m *MockTelegramServer) GetBotAPIURL() string {
	return fmt.Sprintf("%s/bot%s/", m.server.URL, m.botToken)
}
