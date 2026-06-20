package vault

import (
	"sync"
	"testing"
	"time"
)

func TestTokenStore_RaceCondition(t *testing.T) { 
	store := NewTokenStore()

	parentID := "parent-token"
	childID := "child-token"

	_, err := store.CreateToken(parentID, "", 10*time.Second)
	if err != nil {
		t.Fatalf("failed to create parent token: %v", err)
	}

	_, err = store.CreateToken(childID, parentID, 10*time.Second)
	if err != nil {
		t.Fatalf("failed to create child token: %v", err)
	}

	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopChan:
				return
			default:
				_ = store.Renew(childID, 10*time.Second)
			}
		}
	}()

	time.Sleep(10 * time.Millisecond)

	err = store.Revoke(parentID)
	if err != nil {
		t.Fatalf("failed to revoke parent token: %v", err)
	}

	close(stopChan)
	wg.Wait()

	_, err = store.Lookup(childID)
	if err == nil {
		t.Error("expected child token lookup to fail after parent revocation, but it succeeded")
	}

	err = store.Renew(childID, 10*time.Second)
	if err == nil {
		t.Error("expected child token renewal to fail after parent revocation, but it succeeded")
	}
}

func TestTokenStore_LineageValidation(t *testing.T) {
	store := NewTokenStore()

	gpID := "grandparent"
	pID := "parent"
	cID := "child"

	_, _ = store.CreateToken(gpID, "", 10*time.Second)
	_, _ = store.CreateToken(pID, gpID, 10*time.Second)
	_, _ = store.CreateToken(cID, pID, 10*time.Second)

	_ = store.Revoke(gpID)

	_, err := store.Lookup(cID)
	if err == nil {
		t.Error("expected child lookup to fail after grandparent revocation")
	}

	err = store.Renew(cID, 10*time.Second)
	if err == nil {
		t.Error("expected child renewal to fail after grandparent revocation")
	}
}
