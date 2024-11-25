// Copyright (c) 2024 Emanuel Sonnek
// Licensed under the MIT License. See LICENSE file for details.
//
// Email: sonnek.emanuel@gmail.com
// Created: 2024-11-24

package delayedstate

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewStateController(t *testing.T) {
	sc := NewStateController()
	if sc == nil {
		t.Fatal("Expected StateController instance, got nil")
	}
	if sc.states == nil {
		t.Fatal("Expected states map to be initialized")
	}
}

func TestAddState(t *testing.T) {
	sc := NewStateController()
	state := State{Delay: time.Second}

	err := sc.AddState("state1", state)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Try adding the same state again
	err = sc.AddState("state1", state)
	if !errors.Is(err, ErrStateExists) {
		t.Fatalf("Expected ErrStateExists, got %v", err)
	}
}

func TestRemoveState(t *testing.T) {
	sc := NewStateController()
	state := State{Delay: time.Second}

	sc.AddState("state1", state)
	sc.RemoveState("state1")

	if _, exists := sc.states["state1"]; exists {
		t.Fatal("Expected state1 to be removed")
	}

	// Removing non-existent state should not cause panic
	sc.RemoveState("nonexistent")
}

func TestRemoveStateFiresOnStateChange(t *testing.T) {
	called := false
	cb := func(name string, active bool) {
		if name == "state1" && !active {
			called = true
		}
	}

	sc := NewStateController(WithOnStateChange(cb))
	sc.AddState("state1", State{Delay: time.Second})
	sc.SetState("state1", true)
	called = false // reset after activation callback

	sc.RemoveState("state1")

	if !called {
		t.Fatal("Expected onStateChange to be called when removing active state")
	}
}

func TestRemoveStateNoCallbackWhenInactive(t *testing.T) {
	callCount := 0
	cb := func(name string, active bool) {
		callCount++
	}

	sc := NewStateController(WithOnStateChange(cb))
	sc.AddState("state1", State{Delay: time.Second})
	callCount = 0

	sc.RemoveState("state1")

	if callCount != 0 {
		t.Fatalf("Expected no callback when removing inactive state, got %d calls", callCount)
	}
}

func TestSetState(t *testing.T) {
	sc := NewStateController()
	state := State{Delay: 100 * time.Millisecond}
	sc.AddState("state1", state)

	// Set state to active
	err := sc.SetState("state1", true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !sc.IsActive("state1") {
		t.Fatal("Expected state1 to be active")
	}

	// Set state to inactive
	err = sc.SetState("state1", false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// isActive should still be true before delay
	if !sc.IsActive("state1") {
		t.Fatal("Expected state1 to remain active before delay")
	}

	// Wait for delay duration
	time.Sleep(150 * time.Millisecond)

	if sc.IsActive("state1") {
		t.Fatal("Expected state1 to be inactive after delay")
	}
}

func TestSetStateInverted(t *testing.T) {
	sc := NewStateController()
	state := State{Delay: 100 * time.Millisecond, DelayOnActivation: true}
	sc.AddState("state1", state)

	// Set state to active
	err := sc.SetState("state1", true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// isActive should still be false before delay
	if sc.IsActive("state1") {
		t.Fatal("Expected state1 to be inactive before delay")
	}

	// Wait for delay duration
	time.Sleep(150 * time.Millisecond)

	if !sc.IsActive("state1") {
		t.Fatal("Expected state1 to be active after delay")
	}

	// Set state to inactive
	err = sc.SetState("state1", false)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// isActive should be false immediately
	if sc.IsActive("state1") {
		t.Fatal("Expected state1 to be inactive immediately")
	}
}

func TestIsActive(t *testing.T) {
	sc := NewStateController()
	state1 := State{Delay: time.Second}
	state2 := State{Delay: time.Second}

	sc.AddState("state1", state1)
	sc.AddState("state2", state2)

	sc.SetState("state1", true)
	sc.SetState("state2", false)

	if !sc.IsActive("state1") {
		t.Fatal("Expected state1 to be active")
	}

	if sc.IsActive("state2") {
		t.Fatal("Expected state2 to be inactive")
	}
}

func TestOnStateNotExistCallback(t *testing.T) {
	stateCreated := false
	onStateNotExist := func(name string) (State, error) {
		stateCreated = true
		return State{Delay: time.Millisecond * 10}, nil
	}

	sc := NewStateController(WithOnStateNotExist(onStateNotExist))

	err := sc.SetState("newState", true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !stateCreated {
		t.Fatal("Expected onStateNotExist callback to be called")
	}

	if !sc.IsActive("newState") {
		t.Fatal("Expected newState to be active")
	}
}

func TestDelayedTimerCancellation(t *testing.T) {
	sc := NewStateController()
	state := State{Delay: 200 * time.Millisecond}
	sc.AddState("state1", state)

	// Activate first
	sc.SetState("state1", true)

	// Set state to inactive to start delayed timer
	sc.SetState("state1", false)

	// Before delay elapses, set state to active
	time.Sleep(100 * time.Millisecond)
	sc.SetState("state1", true)

	// Wait to see if delayed deactivation still occurs
	time.Sleep(150 * time.Millisecond)

	if !sc.IsActive("state1") {
		t.Fatal("Expected state1 to remain active after timer cancellation")
	}
}

func TestUpdateState(t *testing.T) {
	sc := NewStateController()
	sc.AddState("state1", State{Delay: 100 * time.Millisecond})

	// Update delay
	err := sc.UpdateState("state1", State{Delay: 500 * time.Millisecond})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	s, _ := sc.GetState("state1")
	if s.Delay != 500*time.Millisecond {
		t.Fatalf("Expected delay 500ms, got %v", s.Delay)
	}

	// Update non-existent state
	err = sc.UpdateState("nonexistent", State{})
	if err == nil {
		t.Fatal("Expected error when updating non-existent state")
	}
}

func TestUpdateStateCancelsTimer(t *testing.T) {
	sc := NewStateController()
	sc.AddState("state1", State{Delay: 100 * time.Millisecond})
	sc.SetState("state1", true)
	sc.SetState("state1", false) // starts deactivation timer

	// Update should cancel the pending timer
	sc.UpdateState("state1", State{Delay: 500 * time.Millisecond, IsActive: true})

	time.Sleep(150 * time.Millisecond)

	// State should still be active because timer was cancelled and IsActive was set to true
	if !sc.IsActive("state1") {
		t.Fatal("Expected state1 to remain active after UpdateState cancelled timer")
	}
}

func TestOnStateChangeCallback(t *testing.T) {
	var mu sync.Mutex
	var changes []struct {
		name   string
		active bool
	}

	cb := func(name string, active bool) {
		mu.Lock()
		defer mu.Unlock()
		changes = append(changes, struct {
			name   string
			active bool
		}{name, active})
	}

	sc := NewStateController(WithOnStateChange(cb))
	sc.AddState("state1", State{Delay: 50 * time.Millisecond})

	sc.SetState("state1", true)

	mu.Lock()
	if len(changes) != 1 || changes[0].name != "state1" || !changes[0].active {
		t.Fatalf("Expected activation callback, got %+v", changes)
	}
	mu.Unlock()

	sc.SetState("state1", false)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if len(changes) != 2 || changes[1].name != "state1" || changes[1].active {
		t.Fatalf("Expected deactivation callback, got %+v", changes)
	}
	mu.Unlock()
}

func TestOnStateChangeNotCalledOnDuplicate(t *testing.T) {
	callCount := 0
	cb := func(name string, active bool) {
		callCount++
	}

	sc := NewStateController(WithOnStateChange(cb))
	sc.AddState("state1", State{Delay: time.Second})

	sc.SetState("state1", true)
	sc.SetState("state1", true) // duplicate, should not trigger callback

	if callCount != 1 {
		t.Fatalf("Expected 1 callback call, got %d", callCount)
	}
}

func TestGetState(t *testing.T) {
	sc := NewStateController()
	sc.AddState("state1", State{Delay: time.Second, DelayOnActivation: true})

	s, err := sc.GetState("state1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if s.Delay != time.Second || !s.DelayOnActivation {
		t.Fatalf("Unexpected state config: %+v", s)
	}

	_, err = sc.GetState("nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent state")
	}
}

func TestHasState(t *testing.T) {
	sc := NewStateController()

	if sc.HasState("state1") {
		t.Fatal("Expected HasState to return false for non-existent state")
	}

	sc.AddState("state1", State{})

	if !sc.HasState("state1") {
		t.Fatal("Expected HasState to return true for existing state")
	}

	sc.RemoveState("state1")

	if sc.HasState("state1") {
		t.Fatal("Expected HasState to return false after removal")
	}
}

func TestStateNames(t *testing.T) {
	sc := NewStateController()
	sc.AddState("a", State{})
	sc.AddState("b", State{})
	sc.AddState("c", State{})

	names := sc.StateNames()
	if len(names) != 3 {
		t.Fatalf("Expected 3 state names, got %d", len(names))
	}

	nameSet := make(map[string]struct{}, len(names))
	for _, n := range names {
		nameSet[n] = struct{}{}
	}
	for _, expected := range []string{"a", "b", "c"} {
		if _, ok := nameSet[expected]; !ok {
			t.Fatalf("Expected state name %q in result", expected)
		}
	}
}

func TestReset(t *testing.T) {
	sc := NewStateController()
	sc.AddState("state1", State{Delay: time.Second})
	sc.SetState("state1", true)

	err := sc.Reset("state1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if sc.IsActive("state1") {
		t.Fatal("Expected state1 to be inactive after Reset")
	}
}

func TestResetCancelsPendingTimer(t *testing.T) {
	sc := NewStateController()
	sc.AddState("state1", State{Delay: 200 * time.Millisecond})
	sc.SetState("state1", true)
	sc.SetState("state1", false) // starts delayed deactivation timer

	// Reset immediately cancels the timer and deactivates
	sc.Reset("state1")

	if sc.IsActive("state1") {
		t.Fatal("Expected state1 to be inactive after Reset")
	}

	// Ensure no delayed re-deactivation fires after the timer would have elapsed
	time.Sleep(250 * time.Millisecond)

	// Activate and check it still works correctly
	sc.SetState("state1", true)
	if !sc.IsActive("state1") {
		t.Fatal("Expected state1 to be activatable after Reset")
	}
}

func TestResetNonExistent(t *testing.T) {
	sc := NewStateController()
	err := sc.Reset("nonexistent")
	if !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("Expected ErrStateNotFound, got %v", err)
	}
}

func TestSentinelErrors(t *testing.T) {
	sc := NewStateController()
	sc.AddState("x", State{})

	// ErrStateExists
	err := sc.AddState("x", State{})
	if !errors.Is(err, ErrStateExists) {
		t.Fatalf("Expected ErrStateExists, got %v", err)
	}

	// ErrStateNotFound via SetState
	err = sc.SetState("missing", true)
	if !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("Expected ErrStateNotFound from SetState, got %v", err)
	}

	// ErrStateNotFound via UpdateState
	err = sc.UpdateState("missing", State{})
	if !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("Expected ErrStateNotFound from UpdateState, got %v", err)
	}

	// ErrStateNotFound via GetState
	_, err = sc.GetState("missing")
	if !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("Expected ErrStateNotFound from GetState, got %v", err)
	}

	// ErrStateNotFound via Reset
	err = sc.Reset("missing")
	if !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("Expected ErrStateNotFound from Reset, got %v", err)
	}
}

func TestLen(t *testing.T) {
	sc := NewStateController()
	if sc.Len() != 0 {
		t.Fatalf("Expected Len() = 0, got %d", sc.Len())
	}

	sc.AddState("a", State{})
	sc.AddState("b", State{})

	if sc.Len() != 2 {
		t.Fatalf("Expected Len() = 2, got %d", sc.Len())
	}

	sc.RemoveState("a")
	if sc.Len() != 1 {
		t.Fatalf("Expected Len() = 1, got %d", sc.Len())
	}
}

func TestClear(t *testing.T) {
	sc := NewStateController()
	sc.AddState("a", State{Delay: time.Second})
	sc.AddState("b", State{Delay: time.Second})
	sc.SetState("a", true)
	sc.SetState("b", true)

	sc.Clear()

	if sc.Len() != 0 {
		t.Fatalf("Expected all states removed, got Len() = %d", sc.Len())
	}
	if sc.HasState("a") || sc.HasState("b") {
		t.Fatal("Expected states to be removed after Clear")
	}
}

func TestClearFiresOnStateChangeForActiveStates(t *testing.T) {
	var mu sync.Mutex
	deactivated := make(map[string]bool)

	cb := func(name string, active bool) {
		if !active {
			mu.Lock()
			deactivated[name] = true
			mu.Unlock()
		}
	}

	sc := NewStateController(WithOnStateChange(cb))
	sc.AddState("a", State{Delay: time.Second})
	sc.AddState("b", State{Delay: time.Second})
	sc.AddState("c", State{Delay: time.Second})
	sc.SetState("a", true)
	sc.SetState("b", true)
	// "c" stays inactive

	sc.Clear()

	mu.Lock()
	defer mu.Unlock()
	if !deactivated["a"] || !deactivated["b"] {
		t.Fatal("Expected onStateChange called for active states on Clear")
	}
	if deactivated["c"] {
		t.Fatal("Expected no onStateChange for inactive state on Clear")
	}
}

func TestClearCancelsPendingTimers(t *testing.T) {
	sc := NewStateController()
	sc.AddState("a", State{Delay: 100 * time.Millisecond})
	sc.SetState("a", true)
	sc.SetState("a", false) // starts deactivation timer

	sc.Clear()

	// After timer would have fired, no panic or state change should occur
	time.Sleep(150 * time.Millisecond)

	if sc.HasState("a") {
		t.Fatal("Expected state to be gone after Clear")
	}
}

func TestOnStateChangeCallbackDeadlockSafe(t *testing.T) {
	// Callback that calls back into the StateController must not deadlock.
	sc := NewStateController()
	sc.AddState("state1", State{Delay: 50 * time.Millisecond})

	done := make(chan struct{})
	cb := func(name string, active bool) {
		// Re-entrant call into the controller — must not deadlock.
		_ = sc.IsActive(name)
		_ = sc.HasState(name)
		close(done)
	}
	sc = NewStateController(WithOnStateChange(cb))
	sc.AddState("state1", State{Delay: 50 * time.Millisecond})

	sc.SetState("state1", true)

	select {
	case <-done:
		// success
	case <-time.After(time.Second):
		t.Fatal("Deadlock detected: callback did not complete within timeout")
	}
}

func TestOnStateChangeCallbackDeadlockSafeDelayed(t *testing.T) {
	// Callback triggered by a delayed timer must not deadlock when re-entering the controller.
	done := make(chan struct{})

	var sc *StateController
	cb := func(name string, active bool) {
		_ = sc.IsActive(name)
		if !active {
			// Only close on deactivation (the delayed event we're testing).
			close(done)
		}
	}

	sc = NewStateController(WithOnStateChange(cb))
	sc.AddState("state1", State{Delay: 50 * time.Millisecond})
	sc.SetState("state1", true)
	sc.SetState("state1", false) // triggers delayed deactivation

	select {
	case <-done:
		// success
	case <-time.After(time.Second):
		t.Fatal("Deadlock detected in delayed callback")
	}
}

func TestResetFiresOnStateChange(t *testing.T) {
	called := false
	cb := func(name string, active bool) {
		if name == "state1" && !active {
			called = true
		}
	}

	sc := NewStateController(WithOnStateChange(cb))
	sc.AddState("state1", State{Delay: time.Second})
	sc.SetState("state1", true)
	called = false // reset after activation

	sc.Reset("state1")

	if !called {
		t.Fatal("Expected onStateChange to be called on Reset")
	}
}

func TestActiveStates(t *testing.T) {
	sc := NewStateController()
	sc.AddState("a", State{Delay: time.Second})
	sc.AddState("b", State{Delay: time.Second})
	sc.AddState("c", State{Delay: time.Second})

	active := sc.ActiveStates()
	if len(active) != 0 {
		t.Fatalf("Expected 0 active states, got %d", len(active))
	}

	sc.SetState("a", true)
	sc.SetState("c", true)

	active = sc.ActiveStates()
	if len(active) != 2 {
		t.Fatalf("Expected 2 active states, got %d", len(active))
	}

	nameSet := make(map[string]struct{})
	for _, n := range active {
		nameSet[n] = struct{}{}
	}
	if _, ok := nameSet["a"]; !ok {
		t.Fatal("Expected 'a' in active states")
	}
	if _, ok := nameSet["c"]; !ok {
		t.Fatal("Expected 'c' in active states")
	}
	if _, ok := nameSet["b"]; ok {
		t.Fatal("Did not expect 'b' in active states")
	}
}

func TestOnStateNotExistCallbackDeadlockSafe(t *testing.T) {
	// onStateNotExist callback that calls back into the controller must not deadlock.
	done := make(chan struct{})

	var sc *StateController
	notExistCb := func(name string) (State, error) {
		// Re-entrant calls into the controller — must not deadlock.
		_ = sc.HasState("other")
		_ = sc.IsActive("other")
		_ = sc.Len()
		return State{Delay: 50 * time.Millisecond}, nil
	}

	sc = NewStateController(WithOnStateNotExist(notExistCb))
	sc.AddState("other", State{Delay: time.Second})

	go func() {
		err := sc.SetState("newState", true)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		close(done)
	}()

	select {
	case <-done:
		if !sc.IsActive("newState") {
			t.Fatal("Expected newState to be active")
		}
	case <-time.After(time.Second):
		t.Fatal("Deadlock detected: onStateNotExist callback did not complete")
	}
}

func TestSetStateConcurrentRemove(t *testing.T) {
	// SetState must not panic if the state is removed between RLock check and Lock.
	sc := NewStateController()
	sc.AddState("state1", State{Delay: time.Second})

	// Remove the state
	sc.RemoveState("state1")

	// SetState on removed state must return error, not panic.
	err := sc.SetState("state1", true)
	if !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("Expected ErrStateNotFound, got %v", err)
	}
}

func TestUpdateStateFiresOnStateChange(t *testing.T) {
	var mu sync.Mutex
	var changes []struct {
		name   string
		active bool
	}

	cb := func(name string, active bool) {
		mu.Lock()
		defer mu.Unlock()
		changes = append(changes, struct {
			name   string
			active bool
		}{name, active})
	}

	sc := NewStateController(WithOnStateChange(cb))
	sc.AddState("state1", State{Delay: time.Second})
	sc.SetState("state1", true)

	mu.Lock()
	changes = nil // reset
	mu.Unlock()

	// Deactivate via UpdateState
	sc.UpdateState("state1", State{Delay: time.Second, IsActive: false})

	mu.Lock()
	if len(changes) != 1 || changes[0].name != "state1" || changes[0].active {
		t.Fatalf("Expected deactivation callback from UpdateState, got %+v", changes)
	}
	mu.Unlock()

	// Re-activate via UpdateState
	sc.UpdateState("state1", State{Delay: time.Second, IsActive: true})

	mu.Lock()
	if len(changes) != 2 || changes[1].name != "state1" || !changes[1].active {
		t.Fatalf("Expected activation callback from UpdateState, got %+v", changes)
	}
	mu.Unlock()
}

func TestUpdateStateNoCallbackWhenUnchanged(t *testing.T) {
	callCount := 0
	cb := func(name string, active bool) {
		callCount++
	}

	sc := NewStateController(WithOnStateChange(cb))
	sc.AddState("state1", State{Delay: time.Second})
	callCount = 0

	// Update config only, IsActive stays false → no callback
	sc.UpdateState("state1", State{Delay: 2 * time.Second})
	if callCount != 0 {
		t.Fatalf("Expected no callback when IsActive unchanged, got %d calls", callCount)
	}
}

func TestPendingStates(t *testing.T) {
	sc := NewStateController()
	sc.AddState("a", State{Delay: 500 * time.Millisecond})
	sc.AddState("b", State{Delay: 500 * time.Millisecond})
	sc.AddState("c", State{Delay: 500 * time.Millisecond})

	pending := sc.PendingStates()
	if len(pending) != 0 {
		t.Fatalf("Expected 0 pending states, got %d", len(pending))
	}

	// Activate a and b, then deactivate to start delayed timer
	sc.SetState("a", true)
	sc.SetState("b", true)
	sc.SetState("a", false) // starts delayed deactivation for a
	sc.SetState("b", false) // starts delayed deactivation for b

	pending = sc.PendingStates()
	if len(pending) != 2 {
		t.Fatalf("Expected 2 pending states, got %d", len(pending))
	}

	nameSet := make(map[string]struct{})
	for _, n := range pending {
		nameSet[n] = struct{}{}
	}
	if _, ok := nameSet["a"]; !ok {
		t.Fatal("Expected 'a' in pending states")
	}
	if _, ok := nameSet["b"]; !ok {
		t.Fatal("Expected 'b' in pending states")
	}
	if _, ok := nameSet["c"]; ok {
		t.Fatal("Did not expect 'c' in pending states")
	}
}

func TestPendingStatesDelayOnActivation(t *testing.T) {
	sc := NewStateController()
	sc.AddState("a", State{Delay: 500 * time.Millisecond, DelayOnActivation: true})

	sc.SetState("a", true) // starts delayed activation

	pending := sc.PendingStates()
	if len(pending) != 1 || pending[0] != "a" {
		t.Fatalf("Expected ['a'] in pending states, got %v", pending)
	}
}

func TestActiveStatesReturnsEmptySlice(t *testing.T) {
	sc := NewStateController()
	sc.AddState("a", State{})

	active := sc.ActiveStates()
	if active == nil {
		t.Fatal("Expected non-nil empty slice, got nil")
	}
	if len(active) != 0 {
		t.Fatalf("Expected 0 active states, got %d", len(active))
	}
}

func TestPendingStatesReturnsEmptySlice(t *testing.T) {
	sc := NewStateController()
	sc.AddState("a", State{})

	pending := sc.PendingStates()
	if pending == nil {
		t.Fatal("Expected non-nil empty slice, got nil")
	}
	if len(pending) != 0 {
		t.Fatalf("Expected 0 pending states, got %d", len(pending))
	}
}
