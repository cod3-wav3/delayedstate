// Copyright (c) 2024 Emanuel Sonnek
// Licensed under the MIT License. See LICENSE file for details.
//
// Email: sonnek.emanuel@gmail.com
// Created: 2024-11-24

package delayedstate

type Option func(*StateController)

// WithOnStateNotExist sets the callback function to be github.com/fsnotify/fsnotifycalled when a state does not exist.
func WithOnStateNotExist(cb func(name string) (State, error)) Option {
	return func(sc *StateController) {
		sc.onStateNotExist = cb
	}
}

// WithOnStateChange sets the callback function to be called when a state's active value changes.
func WithOnStateChange(cb StateChangeCallback) Option {
	return func(sc *StateController) {
		sc.onStateChange = cb
	}
}

// WithInitializeStates initializes the StateController with the provided states.
// Note: onStateChange is not called for the initial states.
func WithInitializeStates(states map[string]State) Option {
	if states == nil {
		return func(sc *StateController) {}
	}

	return func(sc *StateController) {
		for name, state := range states {
			sc.states[name] = &delayedState{State: state}
		}
	}
}
