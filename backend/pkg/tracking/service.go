package tracking

import (
	"database/sql"
	"log"
	"sync"
	"time"
)

// TrackingService 处理埋点数据的服务
type TrackingService struct {
	db             *sql.DB
	unpartBuffer   []*UnpartitionedTrackEvent
	unpartBufferMu sync.Mutex
	unpartDataChan chan *UnpartitionedTrackEvent
	batchSize      int
	flushTime      time.Duration
}

// NewTrackingService 创建新的跟踪服务
func NewTrackingService(db *sql.DB) *TrackingService {
	ts := &TrackingService{
		db:             db,
		unpartBuffer:   make([]*UnpartitionedTrackEvent, 0, 1000),
		unpartDataChan: make(chan *UnpartitionedTrackEvent, 10000),
		batchSize:      200,              // 设置为200以匹配批处理大小
		flushTime:      10 * time.Second, // 增加到10秒以减少数据库压力
	}

	// 启动批处理协程
	go ts.unpartBatchProcessor()

	return ts
}

// TrackUnpartitionedEvent 记录一个不分区跟踪事件
func (ts *TrackingService) TrackUnpartitionedEvent(event *UnpartitionedTrackEvent) {
	log.Printf("收到埋点事件: type=%s, session=%s, path=%s, element=%s, metadata=%s",
		event.EventType, event.SessionID, event.PagePath, event.ElementPath, event.Metadata)

	// 异步处理
	select {
	case ts.unpartDataChan <- event:
		log.Printf("事件已加入队列: type=%s, session=%s", event.EventType, event.SessionID)
	default:
		log.Printf("警告: 埋点队列已满，丢弃事件: type=%s, session=%s", event.EventType, event.SessionID)
	}
}

// unpartBatchProcessor 不分区批处理器
func (ts *TrackingService) unpartBatchProcessor() {
	ticker := time.NewTicker(ts.flushTime)
	defer ticker.Stop()

	for {
		select {
		case event := <-ts.unpartDataChan:
			ts.unpartBufferMu.Lock()
			ts.unpartBuffer = append(ts.unpartBuffer, event)

			// 达到批处理大小则刷新
			if len(ts.unpartBuffer) >= ts.batchSize {
				buffer := ts.unpartBuffer
				ts.unpartBuffer = make([]*UnpartitionedTrackEvent, 0, ts.batchSize)
				ts.unpartBufferMu.Unlock()
				go ts.flushUnpartBuffer(buffer) // 使用协程异步刷新，防止阻塞主处理循环
			} else {
				ts.unpartBufferMu.Unlock()
			}

		case <-ticker.C:
			// 定时刷新
			ts.unpartBufferMu.Lock()
			if len(ts.unpartBuffer) > 0 {
				buffer := ts.unpartBuffer
				ts.unpartBuffer = make([]*UnpartitionedTrackEvent, 0, ts.batchSize)
				ts.unpartBufferMu.Unlock()
				go ts.flushUnpartBuffer(buffer) // 使用协程异步刷新
			} else {
				ts.unpartBufferMu.Unlock()
			}
		}
	}
}

// flushUnpartBuffer 将不分区缓冲区数据批量写入数据库
func (ts *TrackingService) flushUnpartBuffer(events []*UnpartitionedTrackEvent) {
	if len(events) == 0 {
		return
	}

	log.Printf("开始批量写入，事件数量: %d", len(events))

	// 开始事务
	tx, err := ts.db.Begin()
	if err != nil {
		log.Printf("事务启动失败: %v", err)
		return
	}

	// 使用defer来确保事务最终会被处理（提交或回滚）
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("批量写入崩溃恢复: %v", r)
		}
	}()

	// 准备批量插入语句
	stmt, err := tx.Prepare(`
		INSERT INTO track_events_unpartitioned 
		(session_id, user_id, event_type, element_path, page_path, referrer, 
		metadata, user_agent, ip_address, created_at, custom_properties, 
		platform, device_info, event_duration, event_source, app_version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`)

	if err != nil {
		tx.Rollback()
		log.Printf("准备语句失败: %v", err)
		return
	}
	defer stmt.Close()

	successCount := 0
	errorCount := 0

	// 批量插入
	for _, event := range events {
		log.Printf("尝试插入事件: type=%s, session=%s, metadata=%s",
			event.EventType, event.SessionID, event.Metadata)

		_, err = stmt.Exec(
			event.SessionID,
			event.UserID,
			event.EventType,
			event.ElementPath,
			event.PagePath,
			event.Referrer,
			event.Metadata,
			event.UserAgent,
			event.IPAddress,
			event.CreatedAt,
			event.CustomProperties,
			event.Platform,
			event.DeviceInfo,
			event.EventDuration,
			event.EventSource,
			event.AppVersion,
		)

		if err != nil {
			errorCount++
			log.Printf("插入事件失败: %v\n事件详情: type=%s, session=%s, metadata=%s",
				err, event.EventType, event.SessionID, event.Metadata)
		} else {
			successCount++
			log.Printf("事件插入成功: type=%s, session=%s", event.EventType, event.SessionID)
		}
	}

	// 如果有错误但不是全部错误，尝试提交成功的部分
	if errorCount > 0 && errorCount < len(events) {
		log.Printf("部分事件插入失败 (%d/%d)，尝试提交成功部分", errorCount, len(events))
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		log.Printf("提交事务失败: %v", err)
		return
	}

	log.Printf("批量写入完成，成功: %d, 失败: %d", successCount, errorCount)
}
