package tracking

import (
	"database/sql"
	"encoding/json"
	"log"
	"sync"
	"time"
)

// TrackingService 处理埋点数据的服务
type TrackingService struct {
	db        *sql.DB
	buffer    []*TrackEvent
	bufferMu  sync.Mutex
	dataChan  chan *TrackEvent
	batchSize int
	flushTime time.Duration
}

// NewTrackingService 创建新的跟踪服务
func NewTrackingService(db *sql.DB) *TrackingService {
	ts := &TrackingService{
		db:        db,
		buffer:    make([]*TrackEvent, 0, 1000),
		dataChan:  make(chan *TrackEvent, 10000),
		batchSize: 1000,
		flushTime: 5 * time.Second,
	}

	// 启动批处理协程
	go ts.batchProcessor()

	return ts
}

// TrackEvent 记录一个跟踪事件
func (ts *TrackingService) TrackEvent(event *TrackEvent) {
	// 异步处理
	select {
	case ts.dataChan <- event:
		// 成功添加到队列
	default:
		// 队列已满，记录丢弃数据
		log.Printf("警告: 埋点队列已满，丢弃事件: %s", event.EventType)
	}
}

// batchProcessor 批处理器
func (ts *TrackingService) batchProcessor() {
	ticker := time.NewTicker(ts.flushTime)
	defer ticker.Stop()

	for {
		select {
		case event := <-ts.dataChan:
			ts.bufferMu.Lock()
			ts.buffer = append(ts.buffer, event)

			// 达到批处理大小则刷新
			if len(ts.buffer) >= ts.batchSize {
				buffer := ts.buffer
				ts.buffer = make([]*TrackEvent, 0, ts.batchSize)
				ts.bufferMu.Unlock()
				ts.flushBuffer(buffer)
			} else {
				ts.bufferMu.Unlock()
			}

		case <-ticker.C:
			// 定时刷新
			ts.bufferMu.Lock()
			if len(ts.buffer) > 0 {
				buffer := ts.buffer
				ts.buffer = make([]*TrackEvent, 0, ts.batchSize)
				ts.bufferMu.Unlock()
				ts.flushBuffer(buffer)
			} else {
				ts.bufferMu.Unlock()
			}
		}
	}
}

// flushBuffer 将缓冲区数据批量写入数据库
func (ts *TrackingService) flushBuffer(events []*TrackEvent) {
	if len(events) == 0 {
		return
	}

	// 开始事务
	tx, err := ts.db.Begin()
	if err != nil {
		log.Printf("事务启动失败: %v", err)
		return
	}

	// 准备批量插入语句
	stmt, err := tx.Prepare(`
		INSERT INTO track_events 
		(session_id, user_id, event_type, element_path, page_path, referrer, metadata, user_agent, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`)

	if err != nil {
		tx.Rollback()
		return
	}
	defer stmt.Close()

	// 批量插入
	for _, event := range events {
		metadata, err := json.Marshal(event.Metadata)
		if err != nil {
			continue
		}

		_, err = stmt.Exec(
			event.SessionID,
			event.UserID,
			event.EventType,
			event.ElementPath,
			event.PagePath,
			event.Referrer,
			metadata,
			event.UserAgent,
			event.IPAddress,
			event.CreatedAt,
		)

		if err != nil {
			continue
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return
	}
}
