package auth

import (
	"testing"
	"time"
)

func TestBeginPasswordWorkLimitsConcurrencyAndRate(t *testing.T) {
	handler := Handler{
		passwordWorkLimiter: NewLimiter(2, time.Hour),
		passwordWorkSlots:   make(chan struct{}, 1),
	}

	release, ok := handler.beginPasswordWork()
	if !ok {
		t.Fatal("first password operation was not admitted")
	}
	if _, ok := handler.beginPasswordWork(); ok {
		t.Fatal("concurrent password operation was admitted beyond the limit")
	}
	release()

	release, ok = handler.beginPasswordWork()
	if !ok {
		t.Fatal("second password operation was not admitted")
	}
	release()
	if _, ok := handler.beginPasswordWork(); ok {
		t.Fatal("password operation was admitted beyond the global rate limit")
	}
}
