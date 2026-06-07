package tracking

import (
	"log"
	"sync"
	"time"

	"blog/internal/repository"
)

// TrackingService 埋点事件批处理服务
type TrackingService struct {
	repo          *repository.TrackingRepo
	buffer        []*UnpartitionedTrackEvent
	bufferMu      sync.Mutex
	dataChan      chan *UnpartitionedTrackEvent
	batchSize     int
	flushInterval time.Duration
}

// NewTrackingService 创建埋点服务
func NewTrackingService(repo *repository.TrackingRepo) *TrackingService {
	ts := &TrackingService{
		repo:          repo,
		buffer:        make([]*UnpartitionedTrackEvent, 0, 1000),
		dataChan:      make(chan *UnpartitionedTrackEvent, 50000),
		batchSize:     200,
		flushInterval: 10 * time.Second,
	}
	go ts.batchProcessor()
	return ts
}

// TrackUnpartitionedEvent 异步记录埋点事件
func (ts *TrackingService) TrackUnpartitionedEvent(event *UnpartitionedTrackEvent) {
	select {
	case ts.dataChan <- event:
	default:
		log.Printf("警告: 埋点队列已满，同步写入: type=%s, session=%s", event.EventType, event.SessionID)
		ts.repo.InsertOne(toRow(event))
	}
}

func (ts *TrackingService) batchProcessor() {
	ticker := time.NewTicker(ts.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case event := <-ts.dataChan:
			ts.bufferMu.Lock()
			ts.buffer = append(ts.buffer, event)
			if len(ts.buffer) >= ts.batchSize {
				batch := ts.buffer
				ts.buffer = make([]*UnpartitionedTrackEvent, 0, ts.batchSize)
				ts.bufferMu.Unlock()
				go ts.flush(batch)
			} else {
				ts.bufferMu.Unlock()
			}
		case <-ticker.C:
			ts.bufferMu.Lock()
			if len(ts.buffer) > 0 {
				batch := ts.buffer
				ts.buffer = make([]*UnpartitionedTrackEvent, 0, ts.batchSize)
				ts.bufferMu.Unlock()
				go ts.flush(batch)
			} else {
				ts.bufferMu.Unlock()
			}
		}
	}
}

func (ts *TrackingService) flush(events []*UnpartitionedTrackEvent) {
	if len(events) == 0 {
		return
	}
	rows := make([]repository.TrackEventRow, len(events))
	for i, e := range events {
		rows[i] = toRow(e)
	}
	ts.repo.BatchInsertEvents(rows)
}

func toRow(e *UnpartitionedTrackEvent) repository.TrackEventRow {
	return repository.TrackEventRow{
		SessionID:        e.SessionID,
		UserID:           e.UserID,
		EventType:        e.EventType,
		ElementPath:      e.ElementPath,
		PagePath:         e.PagePath,
		Referrer:         e.Referrer,
		Metadata:         e.Metadata,
		UserAgent:        e.UserAgent,
		IPAddress:        e.IPAddress,
		CreatedAt:        e.CreatedAt,
		CustomProperties: e.CustomProperties,
		Platform:         e.Platform,
		DeviceInfo:       e.DeviceInfo,
		EventDuration:    e.EventDuration,
		DeviceID:         e.DeviceID,
		Version:          e.Version,
		DeviceType:       e.DeviceType,
	}
}
