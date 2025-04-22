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
	// 启动表分区维护协程
	go ts.partitionMaintainer()

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
		log.Printf("错误: 开始事务失败: %v", err)
		return
	}

	// 准备批量插入语句
	stmt, err := tx.Prepare(`
		INSERT INTO track_events 
		(session_id, user_id, event_type, element_path, page_path, referrer, metadata, user_agent, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`)

	if err != nil {
		log.Printf("错误: 准备插入语句失败: %v", err)
		tx.Rollback()
		return
	}
	defer stmt.Close()

	// 批量插入
	for _, event := range events {
		metadata, err := json.Marshal(event.Metadata)
		if err != nil {
			log.Printf("警告: 元数据JSON序列化失败: %v", err)
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
			log.Printf("警告: 插入跟踪事件失败: %v", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		log.Printf("错误: 提交事务失败: %v", err)
		tx.Rollback()
		return
	}

	log.Printf("成功: 批量保存了 %d 条埋点数据", len(events))
}

// partitionMaintainer 分区维护器 - 每天定时创建下一天的分区
func (ts *TrackingService) partitionMaintainer() {
	// 每天凌晨1点检查并创建新分区
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, now.Location()).Add(24 * time.Hour)

		// 休眠到下次检查时间
		time.Sleep(next.Sub(now))

		// 创建明天和后天的分区表
		ts.createPartition(1)
		ts.createPartition(2)
	}
}

// createPartition 创建指定日期偏移的分区表
func (ts *TrackingService) createPartition(dayOffset int) {
	targetDay := time.Now().AddDate(0, 0, dayOffset)
	partitionDay := targetDay.Format("20060102")
	nextDay := targetDay.AddDate(0, 0, 1).Format("20060102")

	query := `
	CREATE TABLE IF NOT EXISTS track_events_` + partitionDay + ` 
	PARTITION OF track_events
	FOR VALUES FROM ('` + partitionDay + ` 00:00:00') TO ('` + nextDay + ` 00:00:00');
	`

	_, err := ts.db.Exec(query)
	if err != nil {
		log.Printf("错误: 创建分区表失败: %v", err)
	} else {
		log.Printf("成功: 创建分区表 track_events_%s", partitionDay)
	}
}
