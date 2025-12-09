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
		unpartDataChan: make(chan *UnpartitionedTrackEvent, 50000), // 提升5倍容量，减少丢弃风险
		batchSize:      200,                                        // 设置为200以匹配批处理大小
		flushTime:      10 * time.Second,                           // 增加到10秒以减少数据库压力
	}

	// 启动批处理协程
	go ts.unpartBatchProcessor()

	return ts
}

// TrackUnpartitionedEvent 记录一个不分区跟踪事件
func (ts *TrackingService) TrackUnpartitionedEvent(event *UnpartitionedTrackEvent) {
	log.Printf("收到埋点事件: type=%s, session=%s, path=%s, element=%s, metadata=%s",
		event.EventType, event.SessionID, event.PagePath, event.ElementPath, event.Metadata)

	// 异步处理，带背压机制
	select {
	case ts.unpartDataChan <- event:
		log.Printf("事件已加入队列: type=%s, session=%s", event.EventType, event.SessionID)
	default:
		// 队列已满时，同步写入数据库而不是丢弃事件
		log.Printf("警告: 埋点队列已满，切换到同步写入: type=%s, session=%s", event.EventType, event.SessionID)
		ts.insertSingleEvent(event)
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
		log.Printf("事务启动失败: %v，尝试继续处理", err)
		// 事务失败也尝试单条插入
		for _, event := range events {
			ts.insertSingleEvent(event)
		}
		return
	}

	// 确保事务最终会被处理
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("回滚事务失败: %v", rbErr)
			}
		}
	}()

	// 设置客户端编码为UTF8
	_, err = tx.Exec("SET client_encoding = 'UTF8'")
	if err != nil {
		log.Printf("设置客户端编码失败: %v", err)
	}

	// 准备批量插入语句
	stmt, err := tx.Prepare(`
		INSERT INTO track_event 
		(session_id, user_id, event_type, element_path, page_path, referrer, 
		metadata, user_agent, ip_address, created_at, custom_properties, 
		platform, device_info, event_duration)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11::jsonb, 
		$12, $13::jsonb, $14)
	`)

	if err != nil {
		log.Printf("准备语句失败: %v，尝试单条插入", err)
		// 准备语句失败也尝试单条插入
		for _, event := range events {
			ts.insertSingleEvent(event)
		}
		return
	}
	defer stmt.Close()

	successCount := 0
	failCount := 0

	// 批量插入
	for _, event := range events {
		// 确保JSON字段不为空
		if event.Metadata == "" {
			event.Metadata = "{}"
		}
		if event.CustomProperties == "" {
			event.CustomProperties = "{}"
		}
		if event.DeviceInfo == "" {
			event.DeviceInfo = "{}"
		}

		// 记录设备指纹（user_id）和页面路径，用于诊断中文问题
		log.Printf("处理事件: user_id=%s, page_path=%s", event.UserID, event.PagePath)

		_, execErr := stmt.Exec(
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
		)

		if execErr != nil {
			log.Printf("插入事件失败: %v\n事件详情: type=%s, session=%s, metadata=%s",
				execErr, event.EventType, event.SessionID, event.Metadata)
			failCount++
		} else {
			successCount++
		}
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		log.Printf("提交事务失败: %v, 但部分数据可能已经写入", err)
		return
	}

	log.Printf("批量写入完成，成功: %d, 失败: %d", successCount, failCount)
}

// insertSingleEvent 插入单条事件，用于批处理失败时的备选方案
func (ts *TrackingService) insertSingleEvent(event *UnpartitionedTrackEvent) {
	// 确保JSON字段不为空
	if event.Metadata == "" {
		event.Metadata = "{}"
	}
	if event.CustomProperties == "" {
		event.CustomProperties = "{}"
	}
	if event.DeviceInfo == "" {
		event.DeviceInfo = "{}"
	}

	// 先设置连接的编码
	_, err := ts.db.Exec("SET client_encoding = 'UTF8'")
	if err != nil {
		log.Printf("单条插入设置编码失败: %v", err)
	}

	// 记录中文内容
	log.Printf("单条插入事件: page_path=%s", event.PagePath)

	// 直接执行插入
	_, err = ts.db.Exec(`
		INSERT INTO track_event 
		(session_id, user_id, event_type, element_path, page_path, referrer, 
		metadata, user_agent, ip_address, created_at, custom_properties, 
		platform, device_info, event_duration)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11::jsonb, 
		$12, $13::jsonb, $14)
	`,
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
	)

	if err != nil {
		log.Printf("单条插入失败: %v\n事件详情: type=%s, session=%s",
			err, event.EventType, event.SessionID)
	} else {
		log.Printf("单条插入成功: type=%s, session=%s", event.EventType, event.SessionID)
	}
}
