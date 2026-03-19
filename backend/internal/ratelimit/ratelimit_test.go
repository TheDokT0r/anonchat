package ratelimit_test

import (
	"testing"
	"time"
	"anonchat/backend/internal/ratelimit"
)

func TestMessageLimiter_AllowsUnderLimit(t *testing.T) {
	lim := ratelimit.NewMessageLimiter(10, time.Second)
	for i := 0; i < 10; i++ {
		if !lim.Allow("conn1") {
			t.Fatalf("should allow message %d", i+1)
		}
	}
}

func TestMessageLimiter_BlocksOverLimit(t *testing.T) {
	lim := ratelimit.NewMessageLimiter(5, time.Second)
	for i := 0; i < 5; i++ {
		lim.Allow("conn1")
	}
	if lim.Allow("conn1") {
		t.Fatal("should block message over limit")
	}
}

func TestMessageLimiter_SeparateConnections(t *testing.T) {
	lim := ratelimit.NewMessageLimiter(1, time.Second)
	if !lim.Allow("conn1") {
		t.Fatal("conn1 should be allowed")
	}
	if !lim.Allow("conn2") {
		t.Fatal("conn2 should be allowed independently")
	}
}

func TestIPRoomLimiter_AllowsUnderLimit(t *testing.T) {
	lim := ratelimit.NewIPRoomLimiter(10, time.Minute)
	for i := 0; i < 10; i++ {
		if !lim.Allow("192.168.1.1") {
			t.Fatalf("should allow room creation %d", i+1)
		}
	}
}

func TestIPRoomLimiter_BlocksOverLimit(t *testing.T) {
	lim := ratelimit.NewIPRoomLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		lim.Allow("192.168.1.1")
	}
	if lim.Allow("192.168.1.1") {
		t.Fatal("should block room creation over limit")
	}
}

func TestMessageLimiter_Remove(t *testing.T) {
	lim := ratelimit.NewMessageLimiter(10, time.Second)
	lim.Allow("conn1")
	lim.Remove("conn1")
	if !lim.Allow("conn1") {
		t.Fatal("should allow after removal and re-add")
	}
}
