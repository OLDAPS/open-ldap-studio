package ldapx

import (
	"crypto/hmac"
	"crypto/md5" //nolint:gosec // the mechanism is defined over MD5 (RFC 2195)
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/open-ldap-studio/open-ldap-studio/internal/secrets"
)

// SASL CRAM-MD5, implemented here because go-ldap ships neither the mechanism
// nor a way to add one: its request plumbing is unexported (FR-022, plan.md
// § Complexity Tracking).
//
// The exchange is RFC 2195 carried in an LDAP SASL bind: the server sends a
// challenge with resultCode saslBindInProgress, and the client answers with
// its username and the HMAC-MD5 of the challenge keyed by the secret. The
// secret itself never crosses the wire.
const (
	mechanismCramMD5   = "CRAM-MD5"
	saslBindInProgress = 14
)

// bindCramMD5 performs the two-step exchange on the raw socket, before the
// go-ldap client is started.
//
// It returns the server's verbatim result in every case, including failure:
// an authentication failure is answered with the server's own words, and never
// with a retry loop or an anonymous fallback (deviation D7, contract X8).
func bindCramMD5(raw net.Conn, username string, secret secrets.Secret, timeout time.Duration) (Result, error) {
	if raw == nil {
		return Result{}, TransportError("bind", errors.New("the connection has no socket"))
	}
	if timeout > 0 {
		_ = raw.SetDeadline(time.Now().Add(timeout))
		defer func() { _ = raw.SetDeadline(time.Time{}) }()
	}

	// Step one: name the mechanism and ask for a challenge.
	if err := writeSASLBindRequest(raw, 1, mechanismCramMD5, nil); err != nil {
		return Result{}, TransportError("bind", err)
	}
	result, challenge, err := readBindResponse(raw)
	if err != nil {
		return Result{}, TransportError("bind", err)
	}
	if result.Code != saslBindInProgress {
		if result.Code == Success {
			// A server that accepts CRAM-MD5 without a challenge has not
			// authenticated anyone. Treat it as a refusal rather than a bind.
			return result, ServerError("bind", NewResult(Other, result.MatchedDN,
				"the server completed a CRAM-MD5 bind without issuing a challenge"))
		}
		return result, ServerError("bind", result)
	}
	if len(challenge) == 0 {
		return result, ServerError("bind", NewResult(Other, result.MatchedDN,
			"the server issued an empty CRAM-MD5 challenge"))
	}

	// Step two: answer it. RFC 2195 defines the response as the username, a
	// space, and the lowercase hex HMAC-MD5 of the challenge keyed by the
	// secret.
	mac := hmac.New(md5.New, secret.Bytes())
	mac.Write(challenge)
	response := fmt.Sprintf("%s %s", username, hex.EncodeToString(mac.Sum(nil)))

	if err := writeSASLBindRequest(raw, 2, mechanismCramMD5, []byte(response)); err != nil {
		return Result{}, TransportError("bind", err)
	}
	result, _, err = readBindResponse(raw)
	if err != nil {
		return Result{}, TransportError("bind", err)
	}
	if result.Code != Success {
		return result, ServerError("bind", result)
	}
	return result, nil
}
