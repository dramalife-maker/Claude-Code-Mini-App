package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Store 是記憶體內的 session token 儲存區。
type Store struct {
	mu       sync.Mutex
	sessions map[string]time.Time
	ttl      time.Duration
}

func NewStore(ttl time.Duration) *Store {
	s := &Store{
		sessions: make(map[string]time.Time),
		ttl:      ttl,
	}
	go s.cleanup()
	return s
}

// Create 產生一個新的 session token 並回傳。
func (s *Store) Create() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	s.mu.Lock()
	s.sessions[token] = time.Now().Add(s.ttl)
	s.mu.Unlock()
	return token, nil
}

// Validate 檢查 token 是否有效且未過期。
func (s *Store) Validate(token string) bool {
	if token == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.sessions[token]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(s.sessions, token)
		return false
	}
	return true
}

// Delete 登出時移除 token。
func (s *Store) Delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// cleanup 每小時清除過期的 session。
func (s *Store) cleanup() {
	ticker := time.NewTicker(time.Hour)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for token, exp := range s.sessions {
			if now.After(exp) {
				delete(s.sessions, token)
			}
		}
		s.mu.Unlock()
	}
}
