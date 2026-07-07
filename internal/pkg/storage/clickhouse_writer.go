package storage

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"go.uber.org/zap"
)

// LogTypeAccess 访问日志
const LogTypeAccess = "access_logs"

// LogTypeAttack 攻击日志
const LogTypeAttack = "attack_logs"

// LogTypeDDoS DDoS 日志
const LogTypeDDoS = "ddos_logs"

// LogTypeLayer4 四层转发日志
const LogTypeLayer4 = "layer4_logs"

// LogTypeLayer4Intercept 四层拦截日志
const LogTypeLayer4Intercept = "layer4_intercept_logs"

// LogTypeAI AI 调用日志
const LogTypeAI = "ai_logs"

// LogBatch 一批待写入的日志
type LogBatch struct {
	LogType string
	Items   []interface{}
}

// LogEntry 通用日志写入接口（用于 batch 构造）
type LogEntry interface {
	LogType() string
	AppendToBatch(ctx context.Context, batch driver.Batch) error
}

// ============ ClickHouse 批量写入器 ============

// LogWriterConfig 日志写入器配置
type LogWriterConfig struct {
	BufferSize    int // 缓冲通道大小
	BatchSize     int // 每批写入条数
	FlushInterval int // 触发刷新的间隔（秒）
	Retry         int // 写入失败重试次数
}

// LogWriter ClickHouse 批量写入器
type LogWriter struct {
	conn   driver.Conn
	logger *zap.Logger
	cfg    LogWriterConfig

	// 各类型日志缓冲队列
	chAccess chan LogEntry
	chAttack chan LogEntry
	chDDoS   chan LogEntry
	chL4     chan LogEntry
	chL4Int  chan LogEntry
	chAI     chan LogEntry

	// 通用批量队列：按类型分组写入
	batchCh chan LogBatch

	stopCh chan struct{}
	doneCh chan struct{}

	// 统计指标
	metrics *LogWriterMetrics
}

// LogWriterMetrics 写入器运行指标
type LogWriterMetrics struct {
	TotalReceived  uint64
	TotalWritten   uint64
	TotalErrors    uint64
	TotalRetries   uint64
	BufferDropped  uint64
	BatchesFlushed uint64
}

// NewLogWriter 创建日志写入器
func NewLogWriter(conn driver.Conn, cfg LogWriterConfig, logger *zap.Logger) *LogWriter {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 10000
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 1000
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 5
	}
	if cfg.Retry < 0 {
		cfg.Retry = 3
	}

	w := &LogWriter{
		conn:     conn,
		logger:   logger,
		cfg:      cfg,
		chAccess: make(chan LogEntry, cfg.BufferSize),
		chAttack: make(chan LogEntry, cfg.BufferSize),
		chDDoS:   make(chan LogEntry, cfg.BufferSize),
		chL4:     make(chan LogEntry, cfg.BufferSize),
		chL4Int:  make(chan LogEntry, cfg.BufferSize),
		chAI:     make(chan LogEntry, cfg.BufferSize),
		batchCh:  make(chan LogBatch, 64),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		metrics:  &LogWriterMetrics{},
	}

	return w
}

// Start 启动写入器
func (w *LogWriter) Start() {
	w.logger.Info("log writer started",
		zap.Int("buffer_size", w.cfg.BufferSize),
		zap.Int("batch_size", w.cfg.BatchSize),
		zap.Int("flush_interval", w.cfg.FlushInterval),
		zap.Int("retry", w.cfg.Retry),
	)

	// 启动各类型日志的批处理器
	go w.batchLoop(w.chAccess, LogTypeAccess)
	go w.batchLoop(w.chAttack, LogTypeAttack)
	go w.batchLoop(w.chDDoS, LogTypeDDoS)
	go w.batchLoop(w.chL4, LogTypeLayer4)
	go w.batchLoop(w.chL4Int, LogTypeLayer4Intercept)
	go w.batchLoop(w.chAI, LogTypeAI)
}

// Stop 停止写入器并刷新剩余数据
func (w *LogWriter) Stop() {
	w.logger.Info("log writer stopping...")

	close(w.stopCh)

	// 关闭所有输入通道（同时会触发 batchLoop 的 flush 残留数据）
	close(w.chAccess)
	close(w.chAttack)
	close(w.chDDoS)
	close(w.chL4)
	close(w.chL4Int)
	close(w.chAI)

	close(w.doneCh)

	w.logger.Info("log writer stopped",
		zap.Uint64("received", w.metrics.TotalReceived),
		zap.Uint64("written", w.metrics.TotalWritten),
		zap.Uint64("errors", w.metrics.TotalErrors),
	)
}

// WriteEntry 写入一条日志（非阻塞，缓冲满则丢弃并计数）
func (w *LogWriter) WriteEntry(e LogEntry) bool {
	var ch chan LogEntry
	switch e.LogType() {
	case LogTypeAccess:
		ch = w.chAccess
	case LogTypeAttack:
		ch = w.chAttack
	case LogTypeDDoS:
		ch = w.chDDoS
	case LogTypeLayer4:
		ch = w.chL4
	case LogTypeLayer4Intercept:
		ch = w.chL4Int
	case LogTypeAI:
		ch = w.chAI
	default:
		w.logger.Warn("unknown log type", zap.String("type", e.LogType()))
		return false
	}

	select {
	case ch <- e:
		return true
	default:
		w.metrics.BufferDropped++
		w.logger.Warn("log buffer full, entry dropped",
			zap.String("type", e.LogType()),
			zap.Uint64("dropped", w.metrics.BufferDropped),
		)
		return false
	}
}

// GetMetrics 获取指标
func (w *LogWriter) GetMetrics() LogWriterMetrics {
	return *w.metrics
}

// ============ batchLoop ============

// batchLoop 从指定 channel 读取日志并按 batch_size 或 flush_interval 写入
func (w *LogWriter) batchLoop(ch <-chan LogEntry, logType string) {
	batch := make([]LogEntry, 0, w.cfg.BatchSize)
	ticker := time.NewTicker(time.Duration(w.cfg.FlushInterval) * time.Second)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		w.flushBatch(logType, batch)
		batch = batch[:0]
	}

	for {
		select {
		case <-w.stopCh:
			// 排空 channel 残留数据后退出
			for e := range ch {
				batch = append(batch, e)
				if len(batch) >= w.cfg.BatchSize {
					flush()
				}
			}
			flush()
			return
		case e, ok := <-ch:
			if !ok {
				// channel 已关闭，flush 残留后退出
				flush()
				return
			}
			batch = append(batch, e)
			if len(batch) >= w.cfg.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// flushBatch 执行一次批量写入（带重试）
func (w *LogWriter) flushBatch(logType string, entries []LogEntry) {
	if len(entries) == 0 {
		return
	}

	var lastErr error
	for attempt := 0; attempt <= w.cfg.Retry; attempt++ {
		err := w.writeToClickHouse(logType, entries)
		if err == nil {
			w.metrics.BatchesFlushed++
			w.metrics.TotalWritten += uint64(len(entries))
			if attempt > 0 {
				w.metrics.TotalRetries += uint64(attempt)
				w.logger.Info("batch written after retry",
					zap.String("type", logType),
					zap.Int("count", len(entries)),
					zap.Int("attempts", attempt),
				)
			}
			return
		}
		lastErr = err
		w.logger.Warn("batch write failed, retrying",
			zap.String("type", logType),
			zap.Int("count", len(entries)),
			zap.Int("attempt", attempt+1),
			zap.Int("max_retry", w.cfg.Retry),
			zap.Error(err),
		)
		time.Sleep(time.Duration(attempt+1) * time.Second) // 指数退避
	}

	w.metrics.TotalErrors++
	w.logger.Error("batch write failed after all retries",
		zap.String("type", logType),
		zap.Int("count", len(entries)),
		zap.Error(lastErr),
	)
}

// writeToClickHouse 通过 driver.Batch 接口批量插入
func (w *LogWriter) writeToClickHouse(logType string, entries []LogEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	batch, err := w.conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s", logType))
	if err != nil {
		return fmt.Errorf("prepare batch failed: %w", err)
	}

	for _, e := range entries {
		if err := e.AppendToBatch(ctx, batch); err != nil {
			return fmt.Errorf("append to batch failed: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("batch send failed: %w", err)
	}

	return nil
}

// ============ 辅助函数 ============

// ipv4ToUInt32 将点分十进制 IPv4 转换为 ClickHouse IPv4 (UInt32) 所需的整数。
// 用于 clickhouse-go v2 的 IPv4 类型列；空字符串返回 0。
func ipv4ToUInt32(ip string) uint32 {
	if ip == "" {
		return 0
	}
	// net.ParseIP 返回 16 字节；转成 4 字节 IPv4
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return 0
	}
	parsed = parsed.To4()
	if parsed == nil {
		return 0
	}
	// 大端序：第一字节是最高位
	return uint32(parsed[0])<<24 | uint32(parsed[1])<<16 | uint32(parsed[2])<<8 | uint32(parsed[3])
}

// parsePort 解析端口字符串，失败返回 0
func parsePort(p string) uint16 {
	if p == "" {
		return 0
	}
	n, err := strconv.Atoi(p)
	if err != nil || n < 0 || n > 65535 {
		return 0
	}
	return uint16(n)
}

// ensureNoNull 将空字符串替换为 ClickHouse String 列可接受的非空值
// (clickhouse-go v2 对 LowCardity/String 的 nil/空写入通常会自动处理，但保险起见)
func ensureStr(s string) string {
	return strings.TrimSpace(s)
}
