package dream

import (
	_ "embed"
	"log/slog"
	"sync"
	"time"
)

func init() {
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			GarbageCollectCache()
		}
	}()
}

//go:embed system-prompt.txt
var SystemPrompt string

type Response struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}

type CacheItem struct {
	Time     time.Time
	Response *Response
}

var respCacheMu sync.Mutex
var respCache = make(map[string]CacheItem)

func GetCachedResponse(key string) (*Response, bool) {
	respCacheMu.Lock()
	defer respCacheMu.Unlock()
	if item, ok := respCache[key]; ok {
		return item.Response, true
	}
	return nil, false
}

func CacheResponse(key string, resp *Response) {
	respCacheMu.Lock()
	defer respCacheMu.Unlock()
	respCache[key] = CacheItem{
		Time:     time.Now(),
		Response: resp,
	}
}

func GarbageCollectCache() {
	respCacheMu.Lock()
	defer respCacheMu.Unlock()
	maxAge := 24 * time.Hour
	highWaterMark := 100 * 1024 * 1024
	lowWaterMark := 80 * 1024 * 1024

	totalSize := 0
	oldKeys := 0
	for key, item := range respCache {
		// just delete all old things first
		if maxAge < time.Since(item.Time) {
			delete(respCache, key)
			oldKeys += 1
		} else {
			totalSize += len(item.Response.Body)
		}
	}

	garbageKeys := 0
	if highWaterMark < totalSize {
		// then delete arbitrary keys until we're below the low water mark
		toDelete := totalSize - lowWaterMark
		for key, item := range respCache {
			delete(respCache, key)
			garbageKeys += 1
			toDelete -= len(item.Response.Body)
			if toDelete <= 0 {
				break
			}
		}
	}

	if 0 < oldKeys || 0 < garbageKeys {
		slog.Info("Cleaned up cache", "oldKeys", oldKeys, "garbageKeys", garbageKeys)
	}
}
