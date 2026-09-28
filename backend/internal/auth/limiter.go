package auth

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type limitEntry struct {
	count   int
	resetAt time.Time
}

type Limiter struct {
	mu      sync.Mutex
	entries map[string]limitEntry
	limit   int
	window  time.Duration
}

func NewLimiter(limit int, window time.Duration) *Limiter {
	return &Limiter{entries: make(map[string]limitEntry), limit: limit, window: window}
}

func (l *Limiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[key]
	if !ok || now.After(entry.resetAt) {
		if !ok && len(l.entries) >= 10_000 {
			for entryKey, candidate := range l.entries {
				if now.After(candidate.resetAt) {
					delete(l.entries, entryKey)
				}
			}
			if len(l.entries) >= 10_000 {
				return false
			}
		}
		l.entries[key] = limitEntry{count: 1, resetAt: now.Add(l.window)}
		return true
	}
	if entry.count >= l.limit {
		return false
	}
	entry.count++
	l.entries[key] = entry
	return true
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
