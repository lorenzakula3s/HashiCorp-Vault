package main

import (
	"context"
	"fmt"
	"sync/atomic"
)

// Scheduler represents the task scheduler with epoch-based fencing.
type Scheduler struct {
	epoch uint64
}

// RunTask executes a task only if the current node is the valid leader.
func (s *Scheduler) RunTask(ctx context.Context, expectedEpoch uint64) error {
	// Check if context is canceled or epoch has changed
	select {
	case <-ctx.Done():
		return fmt.Errorf("scheduler halted: context canceled")
	default:
		if atomic.LoadUint64(&s.epoch) != expectedEpoch {
			return fmt.Errorf("scheduler halted: epoch mismatch, leadership lost")
		}
	}

	fmt.Println("Executing task for epoch", expectedEpoch)
	return nil
}

func main() {
	fmt.Println("Vault Scheduler initialized with fencing.")
}