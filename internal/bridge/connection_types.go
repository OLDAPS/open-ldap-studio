package bridge

import "github.com/open-ldap-studio/open-ldap-studio/internal/connections"

// Transport aliases retain the existing Wails contract while connection
// lifecycle ownership lives outside the bridge.
type (
	State     = connections.State
	ConnState = connections.ConnState
)

const (
	StateDisconnected = connections.StateDisconnected
	StateConnecting   = connections.StateConnecting
	StateConnected    = connections.StateConnected
	StateLost         = connections.StateLost
)

var ErrNotConnected = connections.ErrNotConnected
