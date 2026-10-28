package bridge

import (
	"github.com/open-ldap-studio/open-ldap-studio/internal/connections"
	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
)

// Wails supports a single value plus an error. Keep the LDAP result alongside
// the value in one response; three Go return values are silently lost by IPC.
type IdentityResult struct {
	Identity string       `json:"identity"`
	Result   ldapx.Result `json:"result"`
}

type RootDSEResult struct {
	RootDSE ldapx.RootDSE `json:"rootDse"`
	Result  ldapx.Result  `json:"result"`
}

type EntryResult struct {
	Entry  ldapx.Entry  `json:"entry"`
	Result ldapx.Result `json:"result"`
}

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
