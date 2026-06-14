package cqrshtmx

import (
	"container/heap"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type limiterEntry struct {
	lim      *rate.Limiter
	lastUsed time.Time
	heapRef  *evictionEntry
}

// perKeyLimiter holds a token-bucket limiter per extracted key.
// Stale entries are evicted when the map is accessed after their TTL expires.
type perKeyLimiter struct {
	mu           sync.RWMutex
	limit        rate.Limit
	burst        uint
	retryAfter   string
	keyExtractor KeyExtractor
	limiters     map[string]*limiterEntry
	heap         *evictionHeap
	ttl          time.Duration
	maxKeys      uint
}

func newPerKeyLimiter(
	l rate.Limit,
	burst uint,
	extractor KeyExtractor,
	retryAfter string,
	ttl time.Duration,
	maxKeys uint,
) *perKeyLimiter {
	return &perKeyLimiter{
		mu:           sync.RWMutex{},
		limit:        l,
		burst:        burst,
		retryAfter:   retryAfter,
		keyExtractor: extractor,
		limiters:     make(map[string]*limiterEntry),
		heap:         &evictionHeap{},
		ttl:          ttl,
		maxKeys:      maxKeys,
	}
}

func (p *perKeyLimiter) allow(r *http.Request) (bool, string) {
	var key string
	if p.keyExtractor != nil {
		key = p.keyExtractor(r)
	}
	// Explicitly empty key means "skip rate limiting for this request".
	if key == "" && p.keyExtractor != nil {
		return true, ""
	}

	lim := p.limiter(key)
	if lim.Allow() {
		return true, ""
	}

	return false, p.retryAfter
}

func (p *perKeyLimiter) limiter(key string) *rate.Limiter {
	p.mu.RLock()
	entry, ok := p.limiters[key]
	// Check freshness while still holding RLock: entry.lastUsed is mutated under
	// the write lock below, so reading it after RUnlock would race with a
	// concurrent refresh.
	if ok && time.Since(entry.lastUsed) < p.ttl {
		p.mu.RUnlock()
		return entry.lim
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	p.evictStale()

	if entry, ok := p.limiters[key]; ok {
		entry.lastUsed = time.Now()
		if entry.heapRef != nil {
			entry.heapRef.lastUsed = entry.lastUsed
			heap.Fix(p.heap, entry.heapRef.index)
		}
		return entry.lim
	}

	p.evictOldestIfAtCapacity()

	lim := rate.NewLimiter(p.limit, int(p.burst))
	now := time.Now()
	heapRef := &evictionEntry{key: key, lastUsed: now, index: -1}
	newEntry := &limiterEntry{lim: lim, lastUsed: now, heapRef: heapRef}
	p.limiters[key] = newEntry
	heap.Push(p.heap, heapRef)

	return lim
}

// Len returns the number of active rate-limited keys.
func (p *perKeyLimiter) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.limiters)
}

func (p *perKeyLimiter) evictStale() {
	now := time.Now()
	for p.heap.Len() > 0 {
		oldest := (*p.heap)[0]
		if now.Sub(oldest.lastUsed) <= p.ttl {
			break
		}
		heap.Pop(p.heap)
		if entry, ok := p.limiters[oldest.key]; ok && entry.lastUsed.Equal(oldest.lastUsed) {
			delete(p.limiters, oldest.key)
		}
	}
}

func (p *perKeyLimiter) evictOldestIfAtCapacity() {
	if p.maxKeys == 0 || uint(len(p.limiters)) < p.maxKeys {
		return
	}
	for p.heap.Len() > 0 {
		oldest, ok := heap.Pop(p.heap).(*evictionEntry)
		if !ok {
			continue
		}
		if entry, exists := p.limiters[oldest.key]; exists &&
			entry.lastUsed.Equal(oldest.lastUsed) {
			delete(p.limiters, oldest.key)
			return
		}
	}
}

// --- Min-heap for O(log n) eviction ---

type evictionEntry struct {
	key      string
	lastUsed time.Time
	index    int
}

type evictionHeap []*evictionEntry

func (h *evictionHeap) Len() int           { return len(*h) }
func (h *evictionHeap) Less(i, j int) bool { return (*h)[i].lastUsed.Before((*h)[j].lastUsed) }
func (h *evictionHeap) Swap(i, j int) {
	(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
	(*h)[i].index = i
	(*h)[j].index = j
}

func (h *evictionHeap) Push(x any) {
	entry, ok := x.(*evictionEntry)
	if !ok {
		return
	}
	entry.index = len(*h)
	*h = append(*h, entry)
}

func (h *evictionHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[:n-1]
	return item
}
