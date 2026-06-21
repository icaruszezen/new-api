package debug_capture

import "sync"

// recentCapacity 是「最近请求」环形缓冲的固定容量。
const recentCapacity = 20

// errorStore 无条数上限地追加保存所有错误 / 计费异常请求。
type errorStoreT struct {
	mu      sync.Mutex
	records []CaptureRecord
	nextID  uint64
}

func (s *errorStoreT) Append(r CaptureRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	r.ID = s.nextID
	s.records = append(s.records, r)
}

// Snapshot 返回倒序（最新在前）的记录副本。
func (s *errorStoreT) Snapshot() []CaptureRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.records)
	out := make([]CaptureRecord, 0, n)
	for i := n - 1; i >= 0; i-- {
		out = append(out, s.records[i])
	}
	return out
}

func (s *errorStoreT) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = nil
}

// recentStore 用固定容量环形缓冲保留最近 recentCapacity 条请求。
type recentStoreT struct {
	mu     sync.Mutex
	buf    []CaptureRecord
	cap    int
	pos    int // 下一个写入位置
	full   bool
	nextID uint64
}

func newRecentStore(capacity int) *recentStoreT {
	return &recentStoreT{
		buf: make([]CaptureRecord, capacity),
		cap: capacity,
	}
}

func (s *recentStoreT) Push(r CaptureRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	r.ID = s.nextID
	s.buf[s.pos] = r
	s.pos = (s.pos + 1) % s.cap
	if s.pos == 0 {
		s.full = true
	}
}

// Snapshot 返回倒序（最新在前）的记录副本。
func (s *recentStoreT) Snapshot() []CaptureRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := s.pos
	if s.full {
		n = s.cap
	}
	out := make([]CaptureRecord, 0, n)
	for i := 0; i < n; i++ {
		idx := (s.pos - 1 - i + s.cap) % s.cap
		out = append(out, s.buf[idx])
	}
	return out
}

var (
	errorStore  = &errorStoreT{}
	recentStore = newRecentStore(recentCapacity)
)

// ErrorRecords 返回错误捕获记录（最新在前）。
func ErrorRecords() []CaptureRecord {
	return errorStore.Snapshot()
}

// ClearErrorRecords 清空所有错误捕获记录。
func ClearErrorRecords() {
	errorStore.Clear()
}

// RecentRecords 返回最近请求记录（最新在前，最多 recentCapacity 条）。
func RecentRecords() []CaptureRecord {
	return recentStore.Snapshot()
}
