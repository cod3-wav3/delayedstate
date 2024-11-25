# Description

delayedstate is a Go package for managing the state of multiple states with optional delayed state transitions. It allows you to dynamically control activation and deactivation of states, with configurable delays for smoother handling of state changes. The package is thread-safe, making it suitable for concurrent applications like signal processing, device control, or UI state management.

# Key Features:

- Manage multiple named states with independent states.
- Configure custom delay times for deactivation transitions.
- Reactivate states before the delay expires, canceling the pending transition.
- Thread-safe for concurrent environments.

## Install `delayedstate`

    go get github.com/cod3-wav3/delayedstate
