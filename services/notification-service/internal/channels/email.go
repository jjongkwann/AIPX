package channels

import (
	"context"
	"crypto/tls"
	"fmt"
	"html"
	"math"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"shared/pkg/logger"
)

// EmailConfig holds email-specific configuration
type EmailConfig struct {
	*ChannelConfig
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
	UseTLS       bool
	UseStartTLS  bool
}

// EmailChannel implements NotificationChannel for Email
type EmailChannel struct {
	config *EmailConfig
	logger *logger.Logger
}

// EmailMessage represents an email message
type EmailMessage struct {
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	HTMLBody    string
	TextBody    string
	Headers     map[string]string
	Attachments []EmailAttachment
}

// EmailAttachment represents an email attachment
type EmailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// NewEmailChannel creates a new Email notification channel
func NewEmailChannel(config *EmailConfig, logger *logger.Logger) (*EmailChannel, error) {
	if config == nil {
		config = &EmailConfig{
			ChannelConfig: DefaultChannelConfig(),
			SMTPPort:      587,
			UseTLS:        false,
			UseStartTLS:   true,
		}
	}

	channel := &EmailChannel{
		config: config,
		logger: logger,
	}

	if err := channel.Validate(); err != nil {
		return nil, fmt.Errorf("invalid email config: %w", err)
	}

	return channel, nil
}

// Send sends a notification via email
func (e *EmailChannel) Send(ctx context.Context, notification *Notification) error {
	// Get recipient emails from notification data
	recipients := e.getRecipients(notification)
	if len(recipients) == 0 {
		return fmt.Errorf("%w: no recipients configured", ErrChannelNotConfigured)
	}

	// Create email message
	emailMsg := e.createEmailMessage(notification, recipients)

	// Retry logic with exponential backoff
	var lastErr error
	maxRetries := e.config.RetryAttempts
	if notification.MaxRetries > 0 {
		maxRetries = notification.MaxRetries
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}

			e.logger.Warn().
				Int("attempt", attempt+1).
				Dur("backoff", backoff).
				Msg("Retrying email notification")
		}

		// Send email
		if err := e.sendEmail(ctx, emailMsg); err != nil {
			lastErr = err
			e.logger.Error().
				Err(err).
				Int("attempt", attempt+1).
				Msg("Failed to send email notification")
			continue
		}

		// Success
		e.logger.Info().
			Str("channel", "email").
			Str("user_id", notification.UserID).
			Str("event_type", notification.EventType).
			Strs("recipients", recipients).
			Msg("Email notification sent successfully")
		return nil
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries+1, lastErr)
}

// sendEmail sends the email via SMTP
func (e *EmailChannel) sendEmail(ctx context.Context, msg *EmailMessage) error {
	// Build SMTP server address
	addr := fmt.Sprintf("%s:%d", e.config.SMTPHost, e.config.SMTPPort)

	// Build email message
	emailData := e.buildMIMEMessage(msg)

	// Combine all recipients
	allRecipients := append(append(msg.To, msg.Cc...), msg.Bcc...)

	// Connect based on TLS configuration
	if e.config.UseTLS {
		return e.sendWithTLS(addr, emailData, allRecipients)
	}

	return e.sendWithStartTLS(addr, emailData, allRecipients)
}

// sendWithTLS sends email using TLS from the start
func (e *EmailChannel) sendWithTLS(addr string, emailData []byte, recipients []string) error {
	// Create TLS connection
	tlsConfig := &tls.Config{
		ServerName: e.config.SMTPHost,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect with TLS: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, e.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Quit()

	// Authenticate
	if e.config.SMTPUsername != "" && e.config.SMTPPassword != "" {
		auth := smtp.PlainAuth("", e.config.SMTPUsername, e.config.SMTPPassword, e.config.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(e.config.FromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send data
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to start data command: %w", err)
	}

	if _, err := w.Write(emailData); err != nil {
		return fmt.Errorf("failed to write email data: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return nil
}

// sendWithStartTLS sends email using STARTTLS
func (e *EmailChannel) sendWithStartTLS(addr string, emailData []byte, recipients []string) error {
	// Connect to SMTP server
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Quit()

	// Send EHLO
	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("failed to send EHLO: %w", err)
	}

	// Start TLS if configured
	if e.config.UseStartTLS {
		tlsConfig := &tls.Config{
			ServerName: e.config.SMTPHost,
		}

		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Authenticate
	if e.config.SMTPUsername != "" && e.config.SMTPPassword != "" {
		auth := smtp.PlainAuth("", e.config.SMTPUsername, e.config.SMTPPassword, e.config.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(e.config.FromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send data
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to start data command: %w", err)
	}

	if _, err := w.Write(emailData); err != nil {
		return fmt.Errorf("failed to write email data: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return nil
}

// createEmailMessage creates an email message from notification
func (e *EmailChannel) createEmailMessage(notification *Notification, recipients []string) *EmailMessage {
	subject := notification.Title
	htmlBody := e.formatHTMLEmail(notification)
	textBody := e.formatPlainTextEmail(notification)

	return &EmailMessage{
		To:       recipients,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
		Headers: map[string]string{
			"X-Priority":     e.getPriorityHeader(notification.Priority),
			"X-Event-Type":   notification.EventType,
			"X-Notification": "AIPX-Trading-Platform",
		},
	}
}

// buildMIMEMessage builds a MIME multipart email message
func (e *EmailChannel) buildMIMEMessage(msg *EmailMessage) []byte {
	var sb strings.Builder

	// From header
	from := mail.Address{
		Name:    e.config.FromName,
		Address: e.config.FromEmail,
	}
	sb.WriteString(fmt.Sprintf("From: %s\r\n", from.String()))

	// To header
	if len(msg.To) > 0 {
		sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	}

	// Cc header
	if len(msg.Cc) > 0 {
		sb.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.Cc, ", ")))
	}

	// Subject header
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))

	// Custom headers
	for key, value := range msg.Headers {
		sb.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// MIME headers
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: multipart/alternative; boundary=boundary-aipx\r\n")
	sb.WriteString("\r\n")

	// Plain text part
	sb.WriteString("--boundary-aipx\r\n")
	sb.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	sb.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(msg.TextBody)
	sb.WriteString("\r\n\r\n")

	// HTML part
	sb.WriteString("--boundary-aipx\r\n")
	sb.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	sb.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(msg.HTMLBody)
	sb.WriteString("\r\n\r\n")

	// End boundary
	sb.WriteString("--boundary-aipx--\r\n")

	return []byte(sb.String())
}

// formatHTMLEmail formats notification as HTML email
func (e *EmailChannel) formatHTMLEmail(notification *Notification) string {
	color := e.getColorForPriority(notification.Priority)
	emoji := e.getEmojiForPriority(notification.Priority)

	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n")
	sb.WriteString("<html>\n<head>\n")
	sb.WriteString("<meta charset=\"UTF-8\">\n")
	sb.WriteString("<style>\n")
	sb.WriteString("body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }\n")
	sb.WriteString(".container { max-width: 600px; margin: 0 auto; padding: 20px; }\n")
	sb.WriteString(".header { background-color: " + color + "; color: white; padding: 20px; border-radius: 5px 5px 0 0; }\n")
	sb.WriteString(".content { background-color: #f9f9f9; padding: 20px; border: 1px solid #ddd; }\n")
	sb.WriteString(".footer { background-color: #f1f1f1; padding: 10px; font-size: 12px; color: #666; border-radius: 0 0 5px 5px; }\n")
	sb.WriteString(".field { margin: 10px 0; }\n")
	sb.WriteString(".field-label { font-weight: bold; color: #555; }\n")
	sb.WriteString(".field-value { color: #333; }\n")
	sb.WriteString("</style>\n")
	sb.WriteString("</head>\n<body>\n")
	sb.WriteString("<div class=\"container\">\n")

	// Header
	sb.WriteString("<div class=\"header\">\n")
	sb.WriteString(fmt.Sprintf("<h2>%s %s</h2>\n", emoji, html.EscapeString(notification.Title)))
	sb.WriteString("</div>\n")

	// Content
	sb.WriteString("<div class=\"content\">\n")
	sb.WriteString(fmt.Sprintf("<p>%s</p>\n", html.EscapeString(notification.Message)))

	// Metadata fields
	if len(notification.Data) > 0 {
		sb.WriteString("<hr>\n")
		sb.WriteString("<h3>Details</h3>\n")

		fields := e.extractImportantFields(notification.Data)
		for key, value := range fields {
			sb.WriteString("<div class=\"field\">\n")
			sb.WriteString(fmt.Sprintf("<span class=\"field-label\">%s:</span> ", formatFieldName(key)))
			sb.WriteString(fmt.Sprintf("<span class=\"field-value\">%v</span>\n", value))
			sb.WriteString("</div>\n")
		}
	}

	sb.WriteString("</div>\n")

	// Footer
	sb.WriteString("<div class=\"footer\">\n")
	sb.WriteString(fmt.Sprintf("<p>Event: %s | Time: %s</p>\n",
		html.EscapeString(notification.EventType),
		notification.Timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString("<p>This is an automated notification from AIPX Trading Platform</p>\n")
	sb.WriteString("</div>\n")

	sb.WriteString("</div>\n</body>\n</html>")

	return sb.String()
}

// formatPlainTextEmail formats notification as plain text email
func (e *EmailChannel) formatPlainTextEmail(notification *Notification) string {
	var sb strings.Builder

	emoji := e.getEmojiForPriority(notification.Priority)
	sb.WriteString(fmt.Sprintf("%s %s\n", emoji, notification.Title))
	sb.WriteString(strings.Repeat("=", 60))
	sb.WriteString("\n\n")

	sb.WriteString(notification.Message)
	sb.WriteString("\n\n")

	// Metadata fields
	if len(notification.Data) > 0 {
		sb.WriteString("Details:\n")
		sb.WriteString(strings.Repeat("-", 60))
		sb.WriteString("\n")

		fields := e.extractImportantFields(notification.Data)
		for key, value := range fields {
			sb.WriteString(fmt.Sprintf("%s: %v\n", formatFieldName(key), value))
		}
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString(strings.Repeat("-", 60))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("Event: %s\n", notification.EventType))
	sb.WriteString(fmt.Sprintf("Time: %s\n", notification.Timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString("\nThis is an automated notification from AIPX Trading Platform\n")

	return sb.String()
}

// extractImportantFields extracts important fields from data
func (e *EmailChannel) extractImportantFields(data map[string]interface{}) map[string]interface{} {
	fields := make(map[string]interface{})

	importantKeys := []string{
		"order_id", "symbol", "side", "quantity", "price",
		"reason", "status", "position_id", "profit_loss",
	}

	for _, key := range importantKeys {
		if value, ok := data[key]; ok {
			fields[key] = value
		}
	}

	return fields
}

// getColorForPriority returns HTML color for priority
func (e *EmailChannel) getColorForPriority(priority Priority) string {
	switch priority {
	case PriorityLow:
		return "#4CAF50" // green
	case PriorityNormal:
		return "#2196F3" // blue
	case PriorityHigh:
		return "#FF9800" // orange
	case PriorityCritical:
		return "#F44336" // red
	default:
		return "#9E9E9E" // gray
	}
}

// getEmojiForPriority returns emoji for priority
func (e *EmailChannel) getEmojiForPriority(priority Priority) string {
	switch priority {
	case PriorityLow:
		return "🟢"
	case PriorityNormal:
		return "🔵"
	case PriorityHigh:
		return "⚠️"
	case PriorityCritical:
		return "🚨"
	default:
		return "ℹ️"
	}
}

// getPriorityHeader returns X-Priority header value
func (e *EmailChannel) getPriorityHeader(priority Priority) string {
	switch priority {
	case PriorityLow:
		return "5" // Lowest
	case PriorityNormal:
		return "3" // Normal
	case PriorityHigh:
		return "2" // High
	case PriorityCritical:
		return "1" // Highest
	default:
		return "3"
	}
}

// getRecipients gets email recipients from notification data
func (e *EmailChannel) getRecipients(notification *Notification) []string {
	// Try to get from notification data first
	if email, ok := notification.Data["email"].(string); ok && email != "" {
		return []string{email}
	}

	// Try to get array of emails
	if emails, ok := notification.Data["emails"].([]string); ok && len(emails) > 0 {
		return emails
	}

	// Try to get from interface slice
	if emailsInterface, ok := notification.Data["emails"].([]interface{}); ok {
		var emails []string
		for _, e := range emailsInterface {
			if strEmail, ok := e.(string); ok {
				emails = append(emails, strEmail)
			}
		}
		if len(emails) > 0 {
			return emails
		}
	}

	return nil
}

// Name returns the channel name
func (e *EmailChannel) Name() string {
	return "email"
}

// Validate validates the email channel configuration
func (e *EmailChannel) Validate() error {
	if e.config == nil {
		return ErrChannelNotConfigured
	}

	if !e.config.Enabled {
		return nil
	}

	// Validate SMTP configuration
	if e.config.SMTPHost == "" {
		return fmt.Errorf("%w: SMTP host required", ErrInvalidConfig)
	}

	if e.config.SMTPPort <= 0 || e.config.SMTPPort > 65535 {
		return fmt.Errorf("%w: invalid SMTP port", ErrInvalidConfig)
	}

	// Validate from email
	if e.config.FromEmail == "" {
		return fmt.Errorf("%w: from email required", ErrInvalidConfig)
	}

	if _, err := mail.ParseAddress(e.config.FromEmail); err != nil {
		return fmt.Errorf("%w: invalid from email address", ErrInvalidConfig)
	}

	return nil
}

// HealthCheck checks if SMTP server is accessible
func (e *EmailChannel) HealthCheck(ctx context.Context) error {
	if !e.config.Enabled {
		return fmt.Errorf("email channel is disabled")
	}

	// Test SMTP connection
	addr := fmt.Sprintf("%s:%d", e.config.SMTPHost, e.config.SMTPPort)

	// Try to connect with timeout
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("email health check failed: cannot connect to SMTP server: %w", err)
	}
	conn.Close()

	e.logger.Debug().
		Str("smtp_host", e.config.SMTPHost).
		Int("smtp_port", e.config.SMTPPort).
		Msg("Email health check passed")

	return nil
}
