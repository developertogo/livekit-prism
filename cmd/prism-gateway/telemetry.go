package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// AgentTurnRecord represents an analytical turn execution trace.
type AgentTurnRecord struct {
	RoomName    string    `json:"room_name"`
	SessionID   string    `json:"session_id"`
	ToolName    string    `json:"tool_name"`
	SpeechDelta string    `json:"speech_delta"`
	DurationMs  float64   `json:"duration_ms"`
	TTFTMs      float64   `json:"ttft_ms"`
	CardBytes   uint32    `json:"card_bytes"`
	Timestamp   time.Time `json:"timestamp"`
}

// TelemetryRecorder handles batch writing to ClickHouse.
type TelemetryRecorder struct {
	conn       driver.Conn
	queue      chan AgentTurnRecord
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	enabled    bool
}

// NewTelemetryRecorder initializes a ClickHouse connection and batch worker.
func NewTelemetryRecorder(dsn string, bufferSize int) *TelemetryRecorder {
	ctx, cancel := context.WithCancel(context.Background())

	opts, err := clickhouse.ParseDSN(dsn)
	var conn driver.Conn
	enabled := false

	if err != nil {
		log.Printf("[Telemetry] ClickHouse DSN parse error (running in fallback mode): %v", err)
	} else {
		conn, err = clickhouse.Open(opts)
		if err != nil {
			log.Printf("[Telemetry] ClickHouse open error (running in fallback mode): %v", err)
		} else if err := conn.Ping(ctx); err != nil {
			log.Printf("[Telemetry] ClickHouse ping failed (ClickHouse offline, running in fallback mode): %v", err)
		} else {
			log.Printf("[Telemetry] Connected to ClickHouse at %s", opts.Addr)
			enabled = true
			if err := initClickHouseSchema(ctx, conn); err != nil {
				log.Printf("[Telemetry] Failed to initialize table schema: %v", err)
			}
		}
	}

	tr := &TelemetryRecorder{
		conn:    conn,
		queue:   make(chan AgentTurnRecord, bufferSize),
		ctx:     ctx,
		cancel:  cancel,
		enabled: enabled,
	}

	tr.wg.Add(1)
	go tr.worker()

	return tr
}

func initClickHouseSchema(ctx context.Context, conn driver.Conn) error {
	createDB := `CREATE DATABASE IF NOT EXISTS livekit_telemetry;`
	if err := conn.Exec(ctx, createDB); err != nil {
		return fmt.Errorf("create database error: %w", err)
	}

	createTable := `
	CREATE TABLE IF NOT EXISTS livekit_telemetry.agent_turns (
		room_name String,
		session_id String,
		tool_name String,
		speech_delta String,
		duration_ms Float64,
		ttft_ms Float64,
		card_bytes UInt32,
		timestamp DateTime64(3)
	) ENGINE = MergeTree()
	ORDER BY (timestamp, room_name, tool_name);
	`
	if err := conn.Exec(ctx, createTable); err != nil {
		return fmt.Errorf("create table error: %w", err)
	}
	log.Printf("[Telemetry] Initialized ClickHouse table livekit_telemetry.agent_turns")
	return nil
}

// Record queues a turn metric for asynchronous flushing.
func (tr *TelemetryRecorder) Record(rec AgentTurnRecord) {
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now()
	}

	select {
	case tr.queue <- rec:
	default:
		log.Printf("[Telemetry] Queue full, dropping metric for tool: %s", rec.ToolName)
	}
}

func (tr *TelemetryRecorder) worker() {
	defer tr.wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var batch []AgentTurnRecord

	flush := func() {
		if len(batch) == 0 {
			return
		}

		if !tr.enabled || tr.conn == nil {
			log.Printf("[Telemetry (Offline Fallback)] Logged %d records in memory (latest tool: %s, duration: %.2fms)",
				len(batch), batch[len(batch)-1].ToolName, batch[len(batch)-1].DurationMs)
			batch = batch[:0]
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		b, err := tr.conn.PrepareBatch(ctx, "INSERT INTO livekit_telemetry.agent_turns")
		if err != nil {
			log.Printf("[Telemetry] PrepareBatch failed: %v", err)
			batch = batch[:0]
			return
		}

		for _, rec := range batch {
			err := b.Append(
				rec.RoomName,
				rec.SessionID,
				rec.ToolName,
				rec.SpeechDelta,
				rec.DurationMs,
				rec.TTFTMs,
				rec.CardBytes,
				rec.Timestamp,
			)
			if err != nil {
				log.Printf("[Telemetry] Batch append failed: %v", err)
			}
		}

		if err := b.Send(); err != nil {
			log.Printf("[Telemetry] Batch send to ClickHouse failed: %v", err)
		} else {
			log.Printf("[Telemetry] Flushed %d records to ClickHouse", len(batch))
		}

		batch = batch[:0]
	}

	for {
		select {
		case <-tr.ctx.Done():
			// Drain queue
			for {
				select {
				case rec := <-tr.queue:
					batch = append(batch, rec)
				default:
					flush()
					return
				}
			}
		case rec := <-tr.queue:
			batch = append(batch, rec)
			if len(batch) >= 50 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// Close gracefully flushes and stops the telemetry recorder.
func (tr *TelemetryRecorder) Close() {
	tr.cancel()
	tr.wg.Wait()
	if tr.conn != nil {
		_ = tr.conn.Close()
	}
}
