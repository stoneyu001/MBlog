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
	stop          chan struct{}
	done          chan struct{}
	closeOnce     sync.Once
	stateMu       sync.RWMutex
	stopped       bool
}

// NewTrackingService 创建埋点服务
func NewTrackingService(repo *repository.TrackingRepo) *TrackingService {
	ts := &TrackingService{
		repo:          repo,
		buffer:        make([]*UnpartitionedTrackEvent, 0, 1000),
		dataChan:      make(chan *UnpartitionedTrackEvent, 50000),
		batchSize:     200,
		flushInterval: 10 * time.Second,
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}
	go ts.batchProcessor()
	return ts
}

// TrackUnpartitionedEvent 异步记录埋点事件
func (ts *TrackingService) TrackUnpartitionedEvent(event *UnpartitionedTrackEvent) {
	ts.stateMu.RLock()
	defer ts.stateMu.RUnlock()
	if ts.stopped {
		return
	}

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
	defer close(ts.done)
	for {
		select {
		case event := <-ts.dataChan:
			ts.bufferMu.Lock()
			ts.buffer = append(ts.buffer, event)
			if len(ts.buffer) >= ts.batchSize {
				batch := ts.buffer
				ts.buffer = make([]*UnpartitionedTrackEvent, 0, ts.batchSize)
				ts.bufferMu.Unlock()
				ts.flush(batch)
			} else {
				ts.bufferMu.Unlock()
			}
		case <-ticker.C:
			ts.flushBuffered()
		case <-ts.stop:
			ts.drainPendingEvents()
			ts.flushBuffered()
			return
		}
	}
}

func (ts *TrackingService) flushBuffered() {
	ts.flush(ts.takeBuffer())
}

func (ts *TrackingService) takeBuffer() []*UnpartitionedTrackEvent {
	ts.bufferMu.Lock()
	defer ts.bufferMu.Unlock()
	batch := ts.buffer
	ts.buffer = make([]*UnpartitionedTrackEvent, 0, ts.batchSize)
	return batch
}

func (ts *TrackingService) drainPendingEvents() {
	for {
		select {
		case event := <-ts.dataChan:
			ts.bufferMu.Lock()
			ts.buffer = append(ts.buffer, event)
			ts.bufferMu.Unlock()
		default:
			return
		}
	}
}

// Close 停止批处理器，并在数据库关闭前同步写入剩余事件。
func (ts *TrackingService) Close() {
	ts.closeOnce.Do(func() {
		ts.stateMu.Lock()
		ts.stopped = true
		close(ts.stop)
		ts.stateMu.Unlock()
		<-ts.done
	})
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
