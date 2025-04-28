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
		batchSize:      1000,
		flushTime:      5 * time.Second,
	}

	// 启动批处理协程
	go ts.unpartBatchProcessor()

	return ts
}

// TrackUnpartitionedEvent 记录一个不分区跟踪事件
func (ts *TrackingService) TrackUnpartitionedEvent(event *UnpartitionedTrackEvent) {
	// 异步处理
	select {
	case ts.unpartDataChan <- event:
		// 成功添加到队列
	default:
		// 队列已满，记录丢弃数据
		log.Printf("警告: 不分区埋点队列已满，丢弃事件: %s", event.EventType)
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
				ts.flushUnpartBuffer(buffer)
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
				ts.flushUnpartBuffer(buffer)
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

	log.Printf("开始批量写入 %d 条数据", len(events))

	// 开始事务
	tx, err := ts.db.Begin()
	if err != nil {
		log.Printf("事务启动失败: %v", err)
		return
	}

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

	// 批量插入
	for i, event := range events {
		metadata, err := json.Marshal(event.Metadata)
		if err != nil {
			log.Printf("元数据序列化失败: %v", err)
			continue
		}

		customProps, err := json.Marshal(event.CustomProperties)
		if err != nil {
			log.Printf("自定义属性序列化失败: %v", err)
			continue
		}

		deviceInfo, err := json.Marshal(event.DeviceInfo)
		if err != nil {
			log.Printf("设备信息序列化失败: %v", err)
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
			customProps,
			event.Platform,
			deviceInfo,
			event.EventDuration,
			event.EventSource,
			event.AppVersion,
		)

		if err != nil {
			log.Printf("插入事件(%d/%d)失败: %v [事件类型: %s]", i+1, len(events), err, event.EventType)
			continue
		} else {
			if i == 0 || i == len(events)-1 {
				log.Printf("成功插入事件(%d/%d): 类型=%s, 用户ID=%s, 会话ID=%s",
					i+1, len(events), event.EventType, event.UserID, event.SessionID)
			}
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		log.Printf("提交事务失败: %v", err)
		return
	}

	log.Printf("成功写入 %d 条不分区埋点数据", len(events))
}
