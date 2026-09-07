package store

import (
	"context"
	"sync"
	"time"
)

// Memory is an in-memory Store used by tests and the STORE_BACKEND=memory mode.
type Memory struct {
	mu sync.RWMutex
	m  map[string]time.Time
}

func NewMemory() *Memory {
	return &Memory{m: make(map[string]time.Time)}
}

func (s *Memory) Upsert(_ context.Context, username string, dob time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[username] = dob
	return nil
}

func (s *Memory) Get(_ context.Context, username string) (time.Time, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dob, ok := s.m[username]
	return dob, ok, nil
}

func (s *Memory) Ping(_ context.Context) error { return nil }

func (s *Memory) Close() error { return nil }
