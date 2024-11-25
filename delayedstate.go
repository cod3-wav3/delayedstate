// Copyright (c) 2024 Emanuel Sonnek
// Licensed under the MIT License. See LICENSE file for details.
//
// Email: sonnek.emanuel@gmail.com
// Created: 2024-11-24

package delayedstate

import (
	"fmt"
	"sync"
	"time"
)

type State struct {
	IsActive bool
	Inverted bool          // Set to true if the state transition IsActive should be set delayed to true.
	Delay    time.Duration // Configurable delay time for the state transition.
}

// StateController manages multiple states and their state.
type StateController struct {
	mu     sync.Mutex
	states map[string]*delayedState

	// Options
	onStateNotExist func(name string) (State, error)
}

// delayedState handles the state, timer, and delay for an individual state.
type delayedState struct {
	State
	delayedTimer *time.Timer
}

// NewStateController initializes a new StateController.
func NewStateController(opts ...Option) *StateController {
	sc := StateController{
		states: make(map[string]*delayedState),
	}

	sc.addOptions(opts...)

	return &sc
}

// AddState adds a new state to the StateController.
// Returns an error if the state already exists.
func (sc *StateController) AddState(name string, state State) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	_, exists := sc.states[name]
	if exists {
		return fmt.Errorf("state %s already exist", name)
	}

	sc.states[name] = &delayedState{State: state}

	return nil
}

// RemoveState removes an state from the StateController.
func (sc *StateController) RemoveState(name string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	state, exists := sc.states[name]
	if !exists {
		return
	}

	if state.delayedTimer != nil {
		state.delayedTimer.Stop()
		state.delayedTimer = nil
	}

	delete(sc.states, name)
}

// SetState sets the state for a given state name.
// SetState will create the state if it does not exist and the onStateNotExist callback is provided.
// Returns an error if the state does not exist and the onStateNotExist callback is not provided.
func (sc *StateController) SetState(name string, active bool) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Get or create the delayedState for this state.
	state, exists := sc.states[name]
	if !exists {
		if sc.onStateNotExist == nil {
			return fmt.Errorf("state %s does not exist", name)
		}

		createdState, err := sc.onStateNotExist(name)
		if err != nil {
			return err
		}

		sc.states[name] = &delayedState{State: createdState}
		state = sc.states[name]
	}

	if !state.Inverted {
		return sc.handleState(state, active)
	}

	return sc.handleInvertedState(state, active)
}

// IsActive returns the current state for a given state name.
func (sc *StateController) IsActive(stateName string) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	state, exists := sc.states[stateName]
	if !exists {
		return false
	}
	return state.IsActive
}

// State returns the current state for a given state name.
func (sc *StateController) State(stateName string) (State, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	state, exists := sc.states[stateName]
	if !exists {
		return State{}, fmt.Errorf("state %s does not exist", stateName)
	}
	return state.State, nil
}

func (sc *StateController) addOptions(opts ...Option) {
	for _, opt := range opts {
		opt(sc)
	}
}

func (sc *StateController) handleState(state *delayedState, active bool) error {
	if active {
		// Immediate activation.
		state.IsActive = active
		if state.delayedTimer != nil {
			state.delayedTimer.Stop()
			state.delayedTimer = nil
		}
	} else {
		// Delayed deactivation.
		if state.delayedTimer == nil {
			state.delayedTimer = time.AfterFunc(state.Delay, func() {
				sc.mu.Lock()
				defer sc.mu.Unlock()
				state.IsActive = false
				state.delayedTimer = nil
			})
		}
	}

	return nil
}

func (sc *StateController) handleInvertedState(state *delayedState, active bool) error {
	if active {
		// Delayed activation.
		if state.delayedTimer == nil {
			state.delayedTimer = time.AfterFunc(state.Delay, func() {
				sc.mu.Lock()
				defer sc.mu.Unlock()
				state.IsActive = true
				state.delayedTimer = nil
			})
		}
	} else {
		// Immediate deactivation.
		state.IsActive = false
		if state.delayedTimer != nil {
			state.delayedTimer.Stop()
			state.delayedTimer = nil
		}
	}

	return nil
}
