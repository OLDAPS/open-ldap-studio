package ldapx

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-ldap/ldap/v3"
)

func TestClosedConnectionCannotActivate(t *testing.T) {
	c := &Conn{}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.activate(); !errors.Is(err, ErrTransport) {
		t.Fatalf("activate closed connection = %v, want transport failure", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestKeepAliveDoesNotReportLocalClose(t *testing.T) {
	c := &Conn{}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	lost := make(chan error, 1)
	c.StartKeepAlive(t.Context(), time.Millisecond, func(_ Result, err error) { lost <- err })
	select {
	case err := <-lost:
		t.Fatalf("starting keep-alive on a locally closed connection reported a loss: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestTranslateErrorPreservesContextAndServerResult(t *testing.T) {
	for _, cause := range []error{context.Canceled, ldap.NewError(ldap.ErrorNetwork, context.Canceled)} {
		err := translateError("connect", cause)
		if !errors.Is(err, ErrCancelled) || !errors.Is(err, context.Canceled) {
			t.Errorf("cancellation = %v, want cancellable category and cause", err)
		}
	}
	server := &ldap.Error{ResultCode: ldap.LDAPResultUnwillingToPerform, MatchedDN: " dc=Odd ", Err: errors.New("  Odd diagnostic\n")} //nolint:revive // deliberately odd server text must survive verbatim
	err := translateError("startTLS", server)
	result := ResultOf(err)
	if result == nil || result.Code != UnwillingToPerform || result.MatchedDN != server.MatchedDN || result.DiagnosticMessage != server.Err.Error() {
		t.Fatalf("server result changed: %#v", result)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	result = ResultOf(contextError(ctx, "startTLS", server))
	if result == nil || result.Code != UnwillingToPerform || result.DiagnosticMessage != server.Err.Error() {
		t.Fatalf("cancellation discarded a received server result: %#v", result)
	}
}

func TestCancellationAfterWriteDispatchIsIndeterminate(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := finishWrite(ctx, "modify", context.Canceled)
	if !errors.Is(err, ErrIndeterminate) {
		t.Fatalf("cancelled in-flight write was reported as definite cancellation: %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("indeterminate write lost its cancellation cause: %v", err)
	}
}
