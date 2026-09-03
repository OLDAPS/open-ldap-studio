package changeset

import (
	"fmt"

	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
)

// Guard decides whether a profile may be written to at all.
//
// This runs at the token-issue boundary, below every UI check, because it
// protects against the one accident previews and dry runs cannot catch: the
// right command in the wrong window. Both assume the user meant *this* server
// (research R20, design gap G3).
type Guard struct {
	// RequireProductionConfirmation is the preference behind the extra
	// confirmation a production-tagged profile demands.
	RequireProductionConfirmation bool
}

// ErrReadOnly is returned when a read-only profile is asked for a preview
// token. It is not an error the UI can talk its way past: no token means no
// commit, because Commit accepts nothing else.
type ErrReadOnly struct {
	ProfileID string
	Name      string
}

func (e *ErrReadOnly) Error() string {
	return fmt.Sprintf("connection %q is open read-only; no change can be committed through it", e.Name)
}

// ErrProductionUnconfirmed is returned when a production-tagged profile is
// asked for a token without the extra confirmation.
type ErrProductionUnconfirmed struct {
	ProfileID string
	Name      string
}

func (e *ErrProductionUnconfirmed) Error() string {
	return fmt.Sprintf("connection %q is tagged production and needs its extra confirmation before a change can be previewed", e.Name)
}

// Check refuses a token where the profile forbids one.
//
// confirmedProduction is the user's answer to the extra confirmation, passed
// in rather than remembered: a session-wide "yes" would defeat the point.
func (g Guard) Check(p profiles.Profile, confirmedProduction bool) error {
	if p.ReadOnly {
		return &ErrReadOnly{ProfileID: p.ID, Name: p.Name}
	}
	if g.RequireProductionConfirmation && p.IsProduction() && !confirmedProduction {
		return &ErrProductionUnconfirmed{ProfileID: p.ID, Name: p.Name}
	}
	return nil
}
