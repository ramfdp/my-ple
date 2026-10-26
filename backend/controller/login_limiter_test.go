package controller

import (
	"testing"
	"time"
)

func TestLoginLimiterBlocksIPAndAccountAfterLimit(t *testing.T) {
	now := time.Date(2026, 10, 26, 10, 0, 0, 0, time.UTC)
	limiter := newLoginLimiter(2, 15*time.Minute, func() time.Time { return now })

	if limiter.blocked("192.0.2.1", "user") {
		t.Fatal("first attempt blocked")
	}
	limiter.failed("192.0.2.1", "user")
	if limiter.blocked("192.0.2.1", "user") {
		t.Fatal("second attempt blocked")
	}
	limiter.failed("192.0.2.1", "user")
	if !limiter.blocked("192.0.2.1", "other") {
		t.Fatal("IP was not blocked")
	}
	if !limiter.blocked("192.0.2.2", "user") {
		t.Fatal("account was not blocked")
	}
}

func TestLoginLimiterExpiresAndClearsAccount(t *testing.T) {
	now := time.Date(2026, 10, 26, 10, 0, 0, 0, time.UTC)
	limiter := newLoginLimiter(1, 15*time.Minute, func() time.Time { return now })
	limiter.failed("192.0.2.1", "user")
	limiter.succeeded("user")

	if limiter.blocked("192.0.2.2", "user") {
		t.Fatal("successful login did not clear account failures")
	}
	if !limiter.blocked("192.0.2.1", "other") {
		t.Fatal("successful login unexpectedly cleared IP failures")
	}

	now = now.Add(15 * time.Minute)
	if limiter.blocked("192.0.2.1", "other") {
		t.Fatal("expired failure window still blocked")
	}
}
