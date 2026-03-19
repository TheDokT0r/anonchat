package ratelimit

import (
	"sync"
	"time"
)

type MessageLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	maxRate  int
	interval time.Duration
}

type bucket struct {
	tokens    int
	lastReset time.Time
}

func NewMessageLimiter(maxRate int, interval time.Duration) *MessageLimiter {
	return &MessageLimiter{
		buckets:  make(map[string]*bucket),
		maxRate:  maxRate,
		interval: interval,
	}
}

func (m *MessageLimiter) Allow(connID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	b, ok := m.buckets[connID]
	if !ok {
		m.buckets[connID] = &bucket{tokens: m.maxRate - 1, lastReset: now}
		return true
	}

	if now.Sub(b.lastReset) >= m.interval {
		b.tokens = m.maxRate
		b.lastReset = now
	}

	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

func (m *MessageLimiter) Remove(connID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.buckets, connID)
}

type IPRoomLimiter struct {
	mu       sync.Mutex
	windows  map[string][]time.Time
	maxRate  int
	interval time.Duration
}

func NewIPRoomLimiter(maxRate int, interval time.Duration) *IPRoomLimiter {
	return &IPRoomLimiter{
		windows:  make(map[string][]time.Time),
		maxRate:  maxRate,
		interval: interval,
	}
}

func (r *IPRoomLimiter) Allow(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.interval)

	entries := r.windows[ip]
	valid := entries[:0]
	for _, t := range entries {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= r.maxRate {
		r.windows[ip] = valid
		return false
	}

	r.windows[ip] = append(valid, now)
	return true
}
