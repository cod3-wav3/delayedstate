// Copyright (c) 2024 Emanuel Sonnek
// Licensed under the MIT License. See LICENSE file for details.
//
// Email: sonnek.emanuel@gmail.com
// Created: 2024-11-24

package delayedstate

import (
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
	if err == nil {
		t.Fatal("Expected error when adding duplicate state, got nil")
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
	state := State{Delay: 100 * time.Millisecond, Inverted: true}
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
