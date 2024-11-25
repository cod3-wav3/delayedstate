// Copyright (c) 2024 Emanuel Sonnek
// Licensed under the MIT License. See LICENSE file for details.
//
// Email: sonnek.emanuel@gmail.com
// Created: 2024-11-24

package delayedstate

import (
	"errors"
	"testing"
	"time"
)

func TestWithOnStateNotExistOptionSetsCallback(t *testing.T) {
	mockCallback := func(name string) (State, error) {
		return State{}, nil
	}

	sc := NewStateController(WithOnStateNotExist(mockCallback))

	if sc.onStateNotExist == nil {
		t.Fatal("Expected onStateNotExist to be set")
	}
}

func TestOnStateNotExistIsCalled(t *testing.T) {
	callbackCalled := false
	var callbackName string

	mockCallback := func(name string) (State, error) {
		callbackCalled = true
		callbackName = name
		return State{Delay: time.Millisecond * 10}, nil
	}

	sc := NewStateController(WithOnStateNotExist(mockCallback))

	err := sc.SetState("nonexistentState", true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !callbackCalled {
		t.Fatal("Expected onStateNotExist callback to be called")
	}

	if callbackName != "nonexistentState" {
		t.Fatalf("Expected callback to be called with 'nonexistentState', got '%s'", callbackName)
	}
}

func TestOnStateNotExistCreatesState(t *testing.T) {
	mockCallback := func(name string) (State, error) {
		return State{Delay: time.Millisecond * 5, Inverted: true}, nil
	}

	sc := NewStateController(WithOnStateNotExist(mockCallback))

	err := sc.SetState("newState", true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	state, exists := sc.states["newState"]
	if !exists {
		t.Fatal("Expected 'newState' to be added to states")
	}

	if state.State.Inverted != true {
		t.Fatal("Expected state to have inverted=true")
	}

	<-time.After(time.Millisecond * 10)

	if !sc.IsActive("newState") {
		t.Fatal("Expected 'newState' to be active")
	}
}

func TestOnStateNotExistErrorHandling(t *testing.T) {
	mockError := errors.New("mock error")
	mockCallback := func(name string) (State, error) {
		return State{}, mockError
	}

	sc := NewStateController(WithOnStateNotExist(mockCallback))

	err := sc.SetState("errorState", true)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !errors.Is(err, mockError) {
		t.Fatalf("Expected error to be '%v', got '%v'", mockError, err)
	}

	_, exists := sc.states["errorState"]
	if exists {
		t.Fatal("Expected 'errorState' not to be added to states due to error")
	}
}
