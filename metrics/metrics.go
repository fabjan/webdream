package metrics

import (
	"expvar"
	"net/http"
	"sync"
	"time"

	"github.com/zserge/metric"
)

type statEvent struct {
	time  time.Time
	count int
}

type statTracker struct {
	mu       sync.Mutex
	stats    []statEvent
	duration time.Duration
}

func newStatTracker(duration time.Duration) *statTracker {
	tracker := &statTracker{
		stats:    make([]statEvent, 0),
		duration: duration,
	}
	go tracker.cleanup()
	return tracker
}

func (et *statTracker) recordEvent(count int) {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.stats = append(et.stats, statEvent{time.Now(), count})
}

func (et *statTracker) currentTotal() int {
	et.mu.Lock()
	defer et.mu.Unlock()
	now := time.Now()
	threshold := now.Add(-et.duration)
	count := 0
	for _, event := range et.stats {
		if event.time.After(threshold) {
			count += event.count
		}
	}
	return count
}

func (et *statTracker) cleanup() {
	for {
		time.Sleep(time.Second)
		et.mu.Lock()
		now := time.Now()
		threshold := now.Add(-et.duration)
		i := 0
		for _, event := range et.stats {
			if event.time.After(threshold) {
				break
			}
			i++
		}
		et.stats = et.stats[i:]
		et.mu.Unlock()
	}
}

var requestsPerMinute *statTracker
var requestsPerDay *statTracker
var tokensPerMinute *statTracker
var tokensPerDay *statTracker

func init() {
	expvar.Publish("llm_requests_total", metric.NewCounter("1m1s", "24h1m"))
	expvar.Publish("llm_tokens_total", metric.NewHistogram("1m1s", "24h1m"))
	requestsPerMinute = newStatTracker(time.Minute)
	requestsPerDay = newStatTracker(24 * time.Hour)
	tokensPerMinute = newStatTracker(time.Minute)
	tokensPerDay = newStatTracker(24 * time.Hour)
}

func Handler() http.Handler {
	return metric.Handler(metric.Exposed)
}

func RecordRequest() {
	requestsPerMinute.recordEvent(1)
	requestsPerDay.recordEvent(1)
	expvar.Get("llm_requests_total").(metric.Metric).Add(1)
}

func CountRequestsInLastMinute() int {
	return requestsPerMinute.currentTotal()
}

func CountRequestsInLastDay() int {
	return requestsPerDay.currentTotal()
}

func RecordTokens(n int) {
	tokensPerMinute.recordEvent(n)
	tokensPerDay.recordEvent(n)
	expvar.Get("llm_tokens_total").(metric.Metric).Add(float64(n))
}

func CountTokensInLastMinute() int {
	return tokensPerMinute.currentTotal()
}

func CountTokensInLastDay() int {
	return tokensPerDay.currentTotal()
}
