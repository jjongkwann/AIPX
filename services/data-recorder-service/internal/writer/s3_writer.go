package writer

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/xitongsys/parquet-go/parquet"
	"github.com/xitongsys/parquet-go/writer"

	"github.com/jjongkwann/aipx/services/data-recorder-service/internal/buffer"
	"github.com/jjongkwann/aipx/services/data-recorder-service/internal/pkg/pb"
)

// S3Writer writes market data to S3 in Parquet format
type S3Writer struct {
	s3Client *s3.Client
	uploader *manager.Uploader
	config   *S3Config

	// Metrics
	totalUploads atomic.Int64
	totalErrors  atomic.Int64
	totalBytes   atomic.Int64
	lastUploadAt atomic.Int64

	// Control
	mu     sync.RWMutex
	closed bool
}

// S3Config holds S3 configuration
type S3Config struct {
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	PartitionPrefix string
	MaxRetries      int
	RetryBackoff    time.Duration
	PartSize        int64
	Concurrency     int
}

// DefaultS3Config returns default configuration
func DefaultS3Config(bucket, region string) *S3Config {
	return &S3Config{
		Bucket:          bucket,
		Region:          region,
		PartitionPrefix: "market-data",
		MaxRetries:      3,
		RetryBackoff:    time.Second,
		PartSize:        5 * 1024 * 1024, // 5MB
		Concurrency:     5,
	}
}

// NewS3Writer creates a new S3 writer
func NewS3Writer(ctx context.Context, cfg *S3Config) (*S3Writer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Load AWS config
	var awsCfg aws.Config
	var err error

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		// Use explicit credentials
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			)),
		)
	} else {
		// Use default credential chain
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsCfg)

	// Create uploader
	uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
		u.PartSize = cfg.PartSize
		u.Concurrency = cfg.Concurrency
	})

	return &S3Writer{
		s3Client: s3Client,
		uploader: uploader,
		config:   cfg,
		closed:   false,
	}, nil
}

// WriteMessages writes messages to S3 in Parquet format
func (sw *S3Writer) WriteMessages(ctx context.Context, messages []buffer.BufferedMessage) error {
	if len(messages) == 0 {
		return nil
	}

	sw.mu.RLock()
	if sw.closed {
		sw.mu.RUnlock()
		return ErrWriterClosed
	}
	sw.mu.RUnlock()

	// Separate messages by type
	var ticks []*pb.TickData
	var orderbooks []*pb.OrderBook
	timestamp := time.Now()

	for _, msg := range messages {
		switch msg.Type {
		case buffer.MessageTypeTick:
			if msg.Tick != nil {
				ticks = append(ticks, msg.Tick)
			}
		case buffer.MessageTypeOrderBook:
			if msg.OrderBook != nil {
				orderbooks = append(orderbooks, msg.OrderBook)
			}
		}
	}

	// Write tick data
	if len(ticks) > 0 {
		if err := sw.writeTicksToParquet(ctx, ticks, timestamp); err != nil {
			sw.totalErrors.Add(1)
			return fmt.Errorf("failed to write ticks to S3: %w", err)
		}
	}

	// Write orderbook data
	if len(orderbooks) > 0 {
		if err := sw.writeOrderBooksToParquet(ctx, orderbooks, timestamp); err != nil {
			sw.totalErrors.Add(1)
			return fmt.Errorf("failed to write orderbooks to S3: %w", err)
		}
	}

	sw.totalUploads.Add(1)
	sw.lastUploadAt.Store(timestamp.Unix())
	return nil
}

// TickParquet represents a tick data record in Parquet
type TickParquet struct {
	Timestamp  int64   `parquet:"name=timestamp, type=INT64"`
	Symbol     string  `parquet:"name=symbol, type=BYTE_ARRAY, convertedtype=UTF8"`
	Price      float64 `parquet:"name=price, type=DOUBLE"`
	Volume     int64   `parquet:"name=volume, type=INT64"`
	Change     float64 `parquet:"name=change, type=DOUBLE"`
	ChangeRate float64 `parquet:"name=change_rate, type=DOUBLE"`
}

// OrderBookParquet represents an orderbook record in Parquet
type OrderBookParquet struct {
	Timestamp int64  `parquet:"name=timestamp, type=INT64"`
	Symbol    string `parquet:"name=symbol, type=BYTE_ARRAY, convertedtype=UTF8"`
	BidsJSON  string `parquet:"name=bids, type=BYTE_ARRAY, convertedtype=UTF8"`
	AsksJSON  string `parquet:"name=asks, type=BYTE_ARRAY, convertedtype=UTF8"`
}

// writeTicksToParquet writes tick data to S3 as Parquet
func (sw *S3Writer) writeTicksToParquet(ctx context.Context, ticks []*pb.TickData, timestamp time.Time) error {
	// Create buffer for Parquet data
	var buf bytes.Buffer

	// Create Parquet writer
	pw, err := writer.NewParquetWriterFromWriter(&buf, new(TickParquet), 4)
	if err != nil {
		return fmt.Errorf("failed to create parquet writer: %w", err)
	}

	pw.CompressionType = parquet.CompressionCodec_SNAPPY

	// Write records
	for _, tick := range ticks {
		record := &TickParquet{
			Timestamp:  tick.Timestamp,
			Symbol:     tick.Symbol,
			Price:      tick.Price,
			Volume:     tick.Volume,
			Change:     tick.Change,
			ChangeRate: tick.ChangeRate,
		}

		if err := pw.Write(record); err != nil {
			pw.WriteStop()
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	if err := pw.WriteStop(); err != nil {
		return fmt.Errorf("failed to finalize parquet file: %w", err)
	}

	// Generate S3 key with time-based partitioning
	key := sw.generateS3Key("tick", timestamp)

	// Upload to S3
	if err := sw.uploadToS3(ctx, key, buf.Bytes()); err != nil {
		return err
	}

	sw.totalBytes.Add(int64(buf.Len()))
	return nil
}

// writeOrderBooksToParquet writes orderbook data to S3 as Parquet
func (sw *S3Writer) writeOrderBooksToParquet(ctx context.Context, orderbooks []*pb.OrderBook, timestamp time.Time) error {
	var buf bytes.Buffer

	pw, err := writer.NewParquetWriterFromWriter(&buf, new(OrderBookParquet), 4)
	if err != nil {
		return fmt.Errorf("failed to create parquet writer: %w", err)
	}

	pw.CompressionType = parquet.CompressionCodec_SNAPPY

	for _, ob := range orderbooks {
		// Convert levels to JSON strings
		bidsJSON, err := orderbookLevelsToJSON(ob.Bids)
		if err != nil {
			pw.WriteStop()
			return fmt.Errorf("failed to marshal bids: %w", err)
		}

		asksJSON, err := orderbookLevelsToJSON(ob.Asks)
		if err != nil {
			pw.WriteStop()
			return fmt.Errorf("failed to marshal asks: %w", err)
		}

		record := &OrderBookParquet{
			Timestamp: ob.Timestamp,
			Symbol:    ob.Symbol,
			BidsJSON:  string(bidsJSON),
			AsksJSON:  string(asksJSON),
		}

		if err := pw.Write(record); err != nil {
			pw.WriteStop()
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	if err := pw.WriteStop(); err != nil {
		return fmt.Errorf("failed to finalize parquet file: %w", err)
	}

	key := sw.generateS3Key("orderbook", timestamp)

	if err := sw.uploadToS3(ctx, key, buf.Bytes()); err != nil {
		return err
	}

	sw.totalBytes.Add(int64(buf.Len()))
	return nil
}

// generateS3Key generates a time-partitioned S3 key
func (sw *S3Writer) generateS3Key(dataType string, t time.Time) string {
	// Format: market-data/tick/year=2025/month=11/day=19/hour=16/data_20251119161234.parquet
	return fmt.Sprintf(
		"%s/%s/year=%04d/month=%02d/day=%02d/hour=%02d/data_%s.parquet",
		sw.config.PartitionPrefix,
		dataType,
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Format("20060102150405"),
	)
}

// uploadToS3 uploads data to S3 with retries
func (sw *S3Writer) uploadToS3(ctx context.Context, key string, data []byte) error {
	var lastErr error

	for attempt := 0; attempt < sw.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(sw.config.RetryBackoff * time.Duration(attempt))
		}

		_, err := sw.uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: aws.String(sw.config.Bucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(data),
		})

		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("failed to upload to S3 after %d retries: %w", sw.config.MaxRetries, lastErr)
}

// Close closes the writer
func (sw *S3Writer) Close() error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if sw.closed {
		return nil
	}

	sw.closed = true
	return nil
}

// HealthCheck verifies S3 connection
func (sw *S3Writer) HealthCheck(ctx context.Context) error {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	if sw.closed {
		return ErrWriterClosed
	}

	// Try to list bucket
	_, err := sw.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(sw.config.Bucket),
	})

	return err
}

// Metrics returns writer metrics
func (sw *S3Writer) Metrics() S3WriterMetrics {
	return S3WriterMetrics{
		TotalUploads: sw.totalUploads.Load(),
		TotalErrors:  sw.totalErrors.Load(),
		TotalBytes:   sw.totalBytes.Load(),
		LastUploadAt: time.Unix(sw.lastUploadAt.Load(), 0),
	}
}

// S3WriterMetrics contains S3 writer statistics
type S3WriterMetrics struct {
	TotalUploads int64
	TotalErrors  int64
	TotalBytes   int64
	LastUploadAt time.Time
}
