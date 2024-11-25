# delayedstate

[![Go Reference](https://pkg.go.dev/badge/github.com/cod3-wav3/delayedstate.svg)](https://pkg.go.dev/github.com/cod3-wav3/delayedstate)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A Go package for managing multiple named states with configurable delayed transitions. Activate or deactivate states with optional delays, cancel pending transitions, and react to state changes via callbacks — all fully thread-safe.

## Why delayedstate?

Many systems deal with noisy or flickering signals — a sensor toggling rapidly, a network connection dropping briefly, or a button being pressed intermittently. Reacting to every raw transition creates instability.

`delayedstate` solves this by introducing configurable delays between state transitions:

- **Delayed deactivation** — keep a state active for a grace period after the signal disappears, preventing flicker (e.g. "user is still present" even if the sensor briefly loses contact).
- **Delayed activation** — require a signal to persist for a minimum duration before accepting it as valid (e.g. debouncing a button press).

Without this package you'd write custom timers, manage goroutines, handle cancellation, and synchronize access manually — for every state. `delayedstate` encapsulates all of that in a simple, tested, thread-safe API.

## Key Features

- Manage multiple named states with independent configurations
- Delayed deactivation — state stays active for a configurable duration before turning off
- Delayed activation — state activates only after a configurable duration of sustained input
- Cancel pending transitions by toggling the state before the delay expires
- React to state changes via a callback (`WithOnStateChange`)
- Auto-create states on first access via a callback (`WithOnStateNotExist`)
- Thread-safe for concurrent environments

## Installation

```sh
go get github.com/cod3-wav3/delayedstate
```

## Quick Start

```go
package main

import (
	"fmt"
	"time"

	"github.com/cod3-wav3/delayedstate"
)

func main() {
	sc := delayedstate.NewStateController(
		delayedstate.WithOnStateChange(func(name string, active bool) {
			fmt.Printf("state %q changed: active=%v\n", name, active)
		}),
	)

	// Add a state with a 2-second delayed deactivation.
	sc.AddState("sensor", delayedstate.State{
		Delay: 2 * time.Second,
	})

	// Activate immediately.
	sc.SetState("sensor", true)  // -> callback: active=true

	// Deactivation is delayed — the state remains active for 2 seconds.
	sc.SetState("sensor", false)

	// Re-activating before the delay expires cancels the pending deactivation.
	sc.SetState("sensor", true)
}
```

## Delayed Activation

Set `DelayOnActivation: true` to delay the activation instead of the deactivation. This is useful when a signal must be sustained for a minimum duration before it is considered active.

```go
sc.AddState("button", delayedstate.State{
	Delay:             500 * time.Millisecond,
	DelayOnActivation: true,
})

// Activation is delayed — the state becomes active only after 500ms.
sc.SetState("button", true)

// Deactivating before the delay expires cancels the pending activation.
sc.SetState("button", false)
```

## Options

| Option                      | Description                                                                                                   |
| --------------------------- | ------------------------------------------------------------------------------------------------------------- |
| `WithOnStateChange(cb)`     | Called whenever a state's active value changes.                                                               |
| `WithOnStateNotExist(cb)`   | Called when `SetState` targets a state that does not exist. The callback returns a `State` to auto-create it. |
| `WithInitializeStates(map)` | Pre-populates the controller with a set of states. `OnStateChange` is not fired for these.                    |

## API Overview

| Method                        | Description                                                             |
| ----------------------------- | ----------------------------------------------------------------------- |
| `NewStateController(opts...)` | Create a new controller with functional options.                        |
| `AddState(name, state)`       | Register a new state. Returns `ErrStateExists` if it already exists.    |
| `SetState(name, active)`      | Activate or deactivate a state, respecting the configured delay.        |
| `UpdateState(name, state)`    | Replace configuration of an existing state. Cancels any pending timer.  |
| `RemoveState(name)`           | Remove a state and cancel its pending timer.                            |
| `Reset(name)`                 | Cancel any pending timer and immediately deactivate the state.          |
| `GetState(name)`              | Return the current `State` configuration.                               |
| `IsActive(name)`              | Return whether the state is currently active.                           |
| `HasState(name)`              | Return whether a state with the given name exists.                      |
| `ActiveStates()`              | Return the names of all currently active states.                        |
| `PendingStates()`             | Return the names of all states with a pending delayed transition.       |
| `StateNames()`                | Return all registered state names.                                      |
| `Len()`                       | Return the number of registered states.                                 |
| `Clear()`                     | Remove all states, cancel all timers, fire callbacks for active states. |

## Errors

Sentinel errors are provided for type-safe checking via `errors.Is`:

```go
if errors.Is(err, delayedstate.ErrStateNotFound) { ... }
if errors.Is(err, delayedstate.ErrStateExists)   { ... }
```

## License

[MIT](LICENSE)
