package controller

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type loginFailures struct {
	count int
	until time.Time
}

type loginLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	now     func() time.Time
	entries map[string]loginFailures
}

func newLoginLimiter(limit int, window time.Duration, now func() time.Time) *loginLimiter {
	return &loginLimiter{limit: limit, window: window, now: now, entries: map[string]loginFailures{}}
}

func (l *loginLimiter) blocked(ip, account string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	return l.blockedKey(now, "ip:"+ip) || l.blockedKey(now, "account:"+strings.ToLower(account))
}

func (l *loginLimiter) blockedKey(now time.Time, key string) bool {
	failure, ok := l.entries[key]
	if !ok {
		return false
	}
	if !now.Before(failure.until) {
		delete(l.entries, key)
		return false
	}
	return failure.count >= l.limit
}

func (l *loginLimiter) failed(ip, account string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.add(now, "ip:"+ip)
	l.add(now, "account:"+strings.ToLower(account))
}

func (l *loginLimiter) add(now time.Time, key string) {
	failure := l.entries[key]
	if !now.Before(failure.until) {
		failure = loginFailures{until: now.Add(l.window)}
	}
	failure.count++
	l.entries[key] = failure
}

func (l *loginLimiter) succeeded(account string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, "account:"+strings.ToLower(account))
}

func loginClientIP(r *http.Request) string {
	if ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); ip != nil {
		return ip.String()
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
