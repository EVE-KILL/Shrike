package main

import (
	"context"
	"sync"
)

// Image uploads stay entirely inside the disposable fixture. This implements
// the real API boundary without configuring or contacting production storage.
type fixtureDomainAssets struct {
	mu    sync.RWMutex
	files map[string][]byte
}

func (s *fixtureDomainAssets) Put(_ context.Context, key string, body []byte, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.files == nil {
		s.files = make(map[string][]byte)
	}
	s.files[key] = append([]byte(nil), body...)
	return nil
}
func (s *fixtureDomainAssets) Get(_ context.Context, key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]byte(nil), s.files[key]...), nil
}
func (s *fixtureDomainAssets) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.files, key)
	return nil
}
