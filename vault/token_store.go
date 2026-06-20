package vault

import (
	"errors"
	"sync"
	"time"
)

// Token represents a Vault token.
type Token struct {
	ID         string
	ParentID   string
	ExpireTime time.Time
	Revoked    bool
}

// TokenStore manages token creation, lookup, renewal, and revocation.
type TokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*Token
}

// NewTokenStore creates a new TokenStore.
func NewTokenStore() *TokenStore {
	return &TokenStore{
		tokens: make(map[string]*Token),
	}
}

// CreateToken creates a new token with the given ID, parent ID, and TTL.
func (s *TokenStore) CreateToken(id string, parentID string, ttl time.Duration) (*Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if parentID != "" {
		parent, exists := s.tokens[parentID]
		if !exists || parent.Revoked {
			return nil, errors.New("parent token does not exist or is revoked")
		}
	}

	token := &Token{
		ID:         id,
		ParentID:   parentID,
		ExpireTime: time.Now().Add(ttl),
		Revoked:    false,
	}
	s.tokens[id] = token
	return token, nil
}

// Lookup retrieves a token by ID, validating its lineage.
func (s *TokenStore) Lookup(id string) (*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, exists := s.tokens[id]
	if !exists || token.Revoked {
		return nil, errors.New("token not found or revoked")
	}

	parentID := token.ParentID
	for parentID != "" {
		parent, exists := s.tokens[parentID]
		if !exists || parent.Revoked {
			return nil, errors.New("parent token is revoked or missing")
		}
		parentID = parent.ParentID
	}

	return token, nil
}

// Renew renews a token's lease, validating its lineage under a lock.
func (s *TokenStore) Renew(id string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, exists := s.tokens[id]
	if !exists || token.Revoked {
		return errors.New("token not found or revoked")
	}

	parentID := token.ParentID
	for parentID != "" {
		parent, exists := s.tokens[parentID]
		if !exists || parent.Revoked {
			token.Revoked = true
			return errors.New("parent token is revoked or missing")
		}
		parentID = parent.ParentID
	}

	if time.Now().After(token.ExpireTime) {
		token.Revoked = true
		return errors.New("token has expired")
	}

	token.ExpireTime = time.Now().Add(ttl)
	return nil
}

// RenewToken is an alias/wrapper for Renew to match different naming conventions.
func (s *TokenStore) RenewToken(id string, ttl time.Duration) error {
	return s.Renew(id, ttl)
}

// Revoke revokes a token and recursively revokes all its children.
func (s *TokenStore) Revoke(id string) error { 
	s.mu.Lock()
	defer s.mu.Unlock()

	token, exists := s.tokens[id]
	if !exists || token.Revoked {
		return nil
	}

	token.Revoked = true
	s.revokeChildren(id)

	return nil
}

// revokeChildren recursively marks all child tokens as revoked.
// Must be called while holding the write lock.
func (s *TokenStore) revokeChildren(parentID string) {
	for _, t := range s.tokens {
		if t.ParentID == parentID && !t.Revoked {
			t.Revoked = true
			s.revokeChildren(t.ID)
		}
	}
}
