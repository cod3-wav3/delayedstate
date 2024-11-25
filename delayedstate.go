// Copyright (c) 2024 Emanuel Sonnek
// Licensed under the MIT License. See LICENSE file for details.
//
// Email: sonnek.emanuel@gmail.com
// Created: 2024-11-24

package delayedstate

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Sentinel errors for type-safe error checking via errors.Is.
var (
	ErrStateNotFound = errors.New("state not found")
	ErrStateExists   = errors.New("state already exists")
)

const (
	stateErrorFormat = "state %s: %w"
)

// StateChangeCallback is called when a state's IsActive value changes.
type StateChangeCallback func(name string, active bool)

// State holds the configuration and current status of a single managed state.
type State struct {
	IsActive          bool
	DelayOnActivation bool          // If true, activation is delayed; otherwise deactivation is delayed.
	Delay             time.Duration // Configurable delay time for the state transition.
}

// StateController manages multiple states and their transitions.
type StateController struct {
	mu     sync.RWMutex
	states map[string]*delayedState

	// Options
	onStateNotExist func(name string) (State, error)
	onStateChange   StateChangeCallback
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
		return fmt.Errorf(stateErrorFormat, name, ErrStateExists)
	}

	sc.states[name] = &delayedState{State: state}

	return nil
}

// UpdateState updates the configuration of an existing state.
// Any pending timer is cancelled. If the IsActive value changes, onStateChange is fired.
// Returns an error if the state does not exist.
func (sc *StateController) UpdateState(name string, state State) error {
	sc.mu.Lock()

	existing, exists := sc.states[name]
	if !exists {
		sc.mu.Unlock()
		return fmt.Errorf(stateErrorFormat, name, ErrStateNotFound)
	}

	if existing.delayedTimer != nil {
		existing.delayedTimer.Stop()
		existing.delayedTimer = nil
	}

	wasActive := existing.IsActive
	existing.State = state
	changed := wasActive != state.IsActive
	cb := sc.onStateChange
	sc.mu.Unlock()

	if changed && cb != nil {
		cb(name, state.IsActive)
	}

	return nil
}

// RemoveState removes a state from the StateController.
// If the state was active, onStateChange is fired with active=false.
func (sc *StateController) RemoveState(name string) {
	sc.mu.Lock()

	state, exists := sc.states[name]
	if !exists {
		sc.mu.Unlock()
		return
	}

	if state.delayedTimer != nil {
		state.delayedTimer.Stop()
		state.delayedTimer = nil
	}

	wasActive := state.IsActive
	delete(sc.states, name)
	cb := sc.onStateChange
	sc.mu.Unlock()

	if wasActive && cb != nil {
		cb(name, false)
	}
}

// SetState sets the state for a given state name.
// SetState will create the state if it does not exist and the onStateNotExist callback is provided.
// Returns an error if the state does not exist and the onStateNotExist callback is not provided.
func (sc *StateController) SetState(name string, active bool) error {
	sc.mu.RLock()
	_, exists := sc.states[name]
	notExistCb := sc.onStateNotExist
	sc.mu.RUnlock()

	if !exists {
		if notExistCb == nil {
			return fmt.Errorf(stateErrorFormat, name, ErrStateNotFound)
		}

		// Call the callback outside of any lock to prevent deadlocks.
		createdState, err := notExistCb(name)
		if err != nil {
			return err
		}

		sc.mu.Lock()
		// Re-check: another goroutine may have added it concurrently.
		if _, exists = sc.states[name]; !exists {
			sc.states[name] = &delayedState{State: createdState}
		}
		sc.mu.Unlock()
	}

	sc.mu.Lock()

	state, exists := sc.states[name]
	if !exists {
		sc.mu.Unlock()
		return fmt.Errorf(stateErrorFormat, name, ErrStateNotFound)
	}

	var changed bool
	if !state.DelayOnActivation {
		changed = sc.handleState(name, state, active)
	} else {
		changed = sc.handleDelayedActivation(name, state, active)
	}

	cb := sc.onStateChange
	sc.mu.Unlock()

	if changed && cb != nil {
		cb(name, active)
	}

	return nil
}

// Reset cancels any pending timer and immediately deactivates the state.
// Returns an error if the state does not exist.
func (sc *StateController) Reset(name string) error {
	sc.mu.Lock()

	state, exists := sc.states[name]
	if !exists {
		sc.mu.Unlock()
		return fmt.Errorf(stateErrorFormat, name, ErrStateNotFound)
	}

	if state.delayedTimer != nil {
		state.delayedTimer.Stop()
		state.delayedTimer = nil
	}

	changed := state.IsActive
	state.IsActive = false
	cb := sc.onStateChange
	sc.mu.Unlock()

	if changed && cb != nil {
		cb(name, false)
	}

	return nil
}

// HasState reports whether a state with the given name exists.
func (sc *StateController) HasState(name string) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	_, exists := sc.states[name]
	return exists
}

// StateNames returns a slice of all registered state names.
func (sc *StateController) StateNames() []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	names := make([]string, 0, len(sc.states))
	for name := range sc.states {
		names = append(names, name)
	}
	return names
}

// Len returns the number of registered states.
func (sc *StateController) Len() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return len(sc.states)
}

// IsActive returns the current active status for a given state name.
func (sc *StateController) IsActive(stateName string) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	state, exists := sc.states[stateName]
	if !exists {
		return false
	}
	return state.IsActive
}

// GetState returns the current state configuration for a given state name.
func (sc *StateController) GetState(stateName string) (State, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	state, exists := sc.states[stateName]
	if !exists {
		return State{}, fmt.Errorf(stateErrorFormat, stateName, ErrStateNotFound)
	}
	return state.State, nil
}

// Clear removes all states, cancelling any pending timers.
// onStateChange is fired for every state that was active at the time of removal.
func (sc *StateController) Clear() {
	sc.mu.Lock()

	var activeNames []string
	for name, state := range sc.states {
		if state.delayedTimer != nil {
			state.delayedTimer.Stop()
			state.delayedTimer = nil
		}
		if state.IsActive {
			activeNames = append(activeNames, name)
		}
	}
	sc.states = make(map[string]*delayedState)
	cb := sc.onStateChange
	sc.mu.Unlock()

	if cb != nil {
		for _, name := range activeNames {
			cb(name, false)
		}
	}
}

// ActiveStates returns a slice of the names of all currently active states.
func (sc *StateController) ActiveStates() []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	names := make([]string, 0, len(sc.states))
	for name, state := range sc.states {
		if state.IsActive {
			names = append(names, name)
		}
	}
	return names
}

// PendingStates returns a slice of the names of all states that have a pending delayed transition.
func (sc *StateController) PendingStates() []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	names := make([]string, 0, len(sc.states))
	for name, state := range sc.states {
		if state.delayedTimer != nil {
			names = append(names, name)
		}
	}
	return names
}

func (sc *StateController) addOptions(opts ...Option) {
	for _, opt := range opts {
		opt(sc)
	}
}

// handleState handles delayed deactivation (default mode).
// Note: If a delayed transition is already pending, repeated calls with the same
// value are ignored (non-retriggerable). The timer is not restarted.
func (sc *StateController) handleState(name string, state *delayedState, active bool) bool {
	if active {
		if state.delayedTimer != nil {
			state.delayedTimer.Stop()
			state.delayedTimer = nil
		}
		if !state.IsActive {
			state.IsActive = true
			return true
		}
	} else {
		if state.IsActive && state.delayedTimer == nil {
			state.delayedTimer = time.AfterFunc(state.Delay, func() {
				sc.mu.Lock()
				if state.delayedTimer == nil {
					sc.mu.Unlock()
					return
				}
				if _, exists := sc.states[name]; !exists {
					sc.mu.Unlock()
					return
				}
				state.IsActive = false
				state.delayedTimer = nil
				cb := sc.onStateChange
				sc.mu.Unlock()
				if cb != nil {
					cb(name, false)
				}
			})
		}
	}
	return false
}

func (sc *StateController) handleDelayedActivation(name string, state *delayedState, active bool) bool {
	if active {
		if !state.IsActive && state.delayedTimer == nil {
			state.delayedTimer = time.AfterFunc(state.Delay, func() {
				sc.mu.Lock()
				if state.delayedTimer == nil {
					sc.mu.Unlock()
					return
				}
				if _, exists := sc.states[name]; !exists {
					sc.mu.Unlock()
					return
				}
				state.IsActive = true
				state.delayedTimer = nil
				cb := sc.onStateChange
				sc.mu.Unlock()
				if cb != nil {
					cb(name, true)
				}
			})
		}
	} else {
		if state.delayedTimer != nil {
			state.delayedTimer.Stop()
			state.delayedTimer = nil
		}
		if state.IsActive {
			state.IsActive = false
			return true
		}
	}
	return false
}
