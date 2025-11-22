package kis

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	market_data "shared/pkg/pb"
	"github.com/rs/zerolog/log"
)

// KISMessage represents a generic KIS WebSocket message
type KISMessage struct {
	Header KISHeader       `json:"header"`
	Body   json.RawMessage `json:"body"`
}

// KISHeader represents the header of a KIS message
type KISHeader struct {
	TrID   string `json:"tr_id"`
	TrKey  string `json:"tr_key"`
	Status string `json:"status"`
}

// KISTickBody represents tick data from KIS
type KISTickBody struct {
	Symbol     string `json:"mksc_shrn_iscd"` // 종목코드
	Price      string `json:"stck_prpr"`       // 현재가
	Volume     string `json:"cntg_vol"`        // 체결량
	Time       string `json:"stck_cntg_hour"`  // 체결시간
	Change     string `json:"prdy_vrss"`       // 전일대비
	ChangeRate string `json:"prdy_vrss_rate"`  // 등락률
}

// KISOrderBookBody represents order book data from KIS
type KISOrderBookBody struct {
	Symbol string                  `json:"mksc_shrn_iscd"` // 종목코드
	Time   string                  `json:"hour"`           // 시간
	Bids   []KISOrderBookLevel     `json:"bids"`
	Asks   []KISOrderBookLevel     `json:"asks"`
}

// KISOrderBookLevel represents a single level in the order book
type KISOrderBookLevel struct {
	Price    string `json:"askp"` // 호가
	Quantity string `json:"askp_rsqn"` // 잔량
}

// MessageType represents the type of market data message
type MessageType string

const (
	MessageTypeTick      MessageType = "tick"
	MessageTypeOrderBook MessageType = "orderbook"
	MessageTypeError     MessageType = "error"
	MessageTypeUnknown   MessageType = "unknown"
)

// ParsedMessage represents a parsed message with its type
type ParsedMessage struct {
	Type      MessageType
	TickData  *market_data.TickData
	OrderBook *market_data.OrderBook
	Error     error
}

// MessageParser handles parsing of KIS messages to Protobuf
type MessageParser struct{}

// NewMessageParser creates a new message parser
func NewMessageParser() *MessageParser {
	return &MessageParser{}
}

// Parse parses a raw KIS message into Protobuf format
func (p *MessageParser) Parse(data []byte) (*ParsedMessage, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty message")
	}

	var kisMsg KISMessage
	if err := json.Unmarshal(data, &kisMsg); err != nil {
		return &ParsedMessage{
			Type:  MessageTypeError,
			Error: fmt.Errorf("failed to unmarshal message: %w", err),
		}, nil
	}

	// Check status
	if kisMsg.Header.Status != "0" && kisMsg.Header.Status != "" {
		return &ParsedMessage{
			Type:  MessageTypeError,
			Error: fmt.Errorf("message status error: %s", kisMsg.Header.Status),
		}, nil
	}

	// Determine message type based on tr_id
	switch kisMsg.Header.TrID {
	case "H0STCNT0", "H0STASP0": // 체결 데이터
		tickData, err := p.parseTickData(kisMsg.Body)
		if err != nil {
			return &ParsedMessage{
				Type:  MessageTypeError,
				Error: fmt.Errorf("failed to parse tick data: %w", err),
			}, nil
		}
		return &ParsedMessage{
			Type:     MessageTypeTick,
			TickData: tickData,
		}, nil

	case "H0STASP1": // 호가 데이터
		orderBook, err := p.parseOrderBook(kisMsg.Body)
		if err != nil {
			return &ParsedMessage{
				Type:  MessageTypeError,
				Error: fmt.Errorf("failed to parse order book: %w", err),
			}, nil
		}
		return &ParsedMessage{
			Type:      MessageTypeOrderBook,
			OrderBook: orderBook,
		}, nil

	default:
		log.Debug().Str("tr_id", kisMsg.Header.TrID).Msg("unknown message type")
		return &ParsedMessage{
			Type: MessageTypeUnknown,
		}, nil
	}
}

// parseTickData parses tick data from KIS format to Protobuf
func (p *MessageParser) parseTickData(body json.RawMessage) (*market_data.TickData, error) {
	var kisBody KISTickBody
	if err := json.Unmarshal(body, &kisBody); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tick body: %w", err)
	}

	// Validate required fields
	if kisBody.Symbol == "" {
		return nil, fmt.Errorf("missing symbol")
	}

	// Parse price
	price, err := parseFloat(kisBody.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %w", err)
	}

	// Parse volume
	volume, err := parseInt(kisBody.Volume)
	if err != nil {
		return nil, fmt.Errorf("invalid volume: %w", err)
	}

	// Parse timestamp
	timestamp, err := parseTime(kisBody.Time)
	if err != nil {
		log.Warn().Err(err).Msg("failed to parse time, using current time")
		timestamp = time.Now().UnixNano()
	}

	// Parse change
	change, _ := parseFloat(kisBody.Change)
	changeRate, _ := parseFloat(kisBody.ChangeRate)

	return &market_data.TickData{
		Symbol:     kisBody.Symbol,
		Price:      price,
		Volume:     volume,
		Timestamp:  timestamp,
		Change:     change,
		ChangeRate: changeRate,
	}, nil
}

// parseOrderBook parses order book data from KIS format to Protobuf
func (p *MessageParser) parseOrderBook(body json.RawMessage) (*market_data.OrderBook, error) {
	var kisBody KISOrderBookBody
	if err := json.Unmarshal(body, &kisBody); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order book body: %w", err)
	}

	// Validate required fields
	if kisBody.Symbol == "" {
		return nil, fmt.Errorf("missing symbol")
	}

	// Parse timestamp
	timestamp, err := parseTime(kisBody.Time)
	if err != nil {
		timestamp = time.Now().UnixNano()
	}

	// Parse bids
	bids := make([]*market_data.Level, 0, len(kisBody.Bids))
	for _, bid := range kisBody.Bids {
		price, err := parseFloat(bid.Price)
		if err != nil {
			continue
		}
		quantity, err := parseInt(bid.Quantity)
		if err != nil {
			continue
		}
		bids = append(bids, &market_data.Level{
			Price:    price,
			Quantity: quantity,
		})
	}

	// Parse asks
	asks := make([]*market_data.Level, 0, len(kisBody.Asks))
	for _, ask := range kisBody.Asks {
		price, err := parseFloat(ask.Price)
		if err != nil {
			continue
		}
		quantity, err := parseInt(ask.Quantity)
		if err != nil {
			continue
		}
		asks = append(asks, &market_data.Level{
			Price:    price,
			Quantity: quantity,
		})
	}

	return &market_data.OrderBook{
		Symbol:    kisBody.Symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: timestamp,
	}, nil
}

// parseFloat parses a string to float64
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}

// parseInt parses a string to int64
func parseInt(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseInt(s, 10, 64)
}

// parseTime parses KIS time format (HHMMSS or HHMMSSMMM) to Unix nanoseconds
func parseTime(timeStr string) (int64, error) {
	if timeStr == "" {
		return 0, fmt.Errorf("empty time string")
	}

	now := time.Now()
	var hour, min, sec, msec int

	// Parse based on length
	switch len(timeStr) {
	case 6: // HHMMSS
		_, err := fmt.Sscanf(timeStr, "%02d%02d%02d", &hour, &min, &sec)
		if err != nil {
			return 0, err
		}
	case 9: // HHMMSSMMM
		_, err := fmt.Sscanf(timeStr, "%02d%02d%02d%03d", &hour, &min, &sec, &msec)
		if err != nil {
			return 0, err
		}
	default:
		return 0, fmt.Errorf("invalid time format: %s", timeStr)
	}

	// Create time with today's date
	t := time.Date(now.Year(), now.Month(), now.Day(), hour, min, sec, msec*1000000, now.Location())
	return t.UnixNano(), nil
}

// ValidateTickData validates tick data
func ValidateTickData(data *market_data.TickData) error {
	if data == nil {
		return fmt.Errorf("nil tick data")
	}
	if data.Symbol == "" {
		return fmt.Errorf("empty symbol")
	}
	if data.Price <= 0 {
		return fmt.Errorf("invalid price: %f", data.Price)
	}
	if data.Volume < 0 {
		return fmt.Errorf("invalid volume: %d", data.Volume)
	}
	if data.Timestamp <= 0 {
		return fmt.Errorf("invalid timestamp: %d", data.Timestamp)
	}
	return nil
}

// ValidateOrderBook validates order book data
func ValidateOrderBook(data *market_data.OrderBook) error {
	if data == nil {
		return fmt.Errorf("nil order book")
	}
	if data.Symbol == "" {
		return fmt.Errorf("empty symbol")
	}
	if len(data.Bids) == 0 && len(data.Asks) == 0 {
		return fmt.Errorf("empty order book")
	}
	if data.Timestamp <= 0 {
		return fmt.Errorf("invalid timestamp: %d", data.Timestamp)
	}
	return nil
}
