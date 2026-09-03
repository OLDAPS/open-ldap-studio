//go:build linux || freebsd || openbsd || netbsd

package secrets

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/godbus/dbus/v5"
)

// The Secret Service API (org.freedesktop.secrets), spoken in-process over
// D-Bus. This is the whole point of Constitution II's "no subprocess on the
// default path": libsecret's CLI would have been simpler and would have put a
// secret on a command line.
const (
	ssBusName        = "org.freedesktop.secrets"
	ssServicePath    = "/org/freedesktop/secrets"
	ssDefaultAlias   = "/org/freedesktop/secrets/aliases/default"
	ssServiceIface   = "org.freedesktop.Secret.Service"
	ssCollectionIfce = "org.freedesktop.Secret.Collection"
	ssItemIface      = "org.freedesktop.Secret.Item"
	ssPromptIface    = "org.freedesktop.Secret.Prompt"

	// promptTimeout bounds how long we wait for the user to answer the agent.
	// The agent owns the interaction; we only decline to wait forever.
	promptTimeout = 2 * time.Minute
)

// secretService is the Linux/BSD binding.
type secretService struct {
	conn *dbus.Conn
}

// dbusSecret mirrors the Secret Service (o, ay, ay, s) struct.
type dbusSecret struct {
	Session     dbus.ObjectPath
	Parameters  []byte
	Value       []byte
	ContentType string
}

// openPlatform connects to the session bus without ever letting go-dbus
// autolaunch one.
//
// dbus.SessionBus() falls back to running dbus-launch when
// DBUS_SESSION_BUS_ADDRESS is unset. We resolve the address ourselves and
// report the store unavailable instead: Constitution II forbids a subprocess
// on the credential path, and a headless session with no bus is exactly the
// recoverable "no credential store" case the session fallback exists for
// (FR-005). See deviation D10 in docs/deviations.md.
func openPlatform() (Provider, error) {
	address := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	if address == "" || address == "autolaunch:" {
		return nil, fmt.Errorf(
			"%w: no DBUS_SESSION_BUS_ADDRESS is set, and starting a bus would require a subprocess",
			ErrUnavailable)
	}

	// Connect dials, authenticates and says Hello. It never autolaunches.
	conn, err := dbus.Connect(address)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot reach the session bus: %v", ErrUnavailable, err)
	}
	return &secretService{conn: conn}, nil
}

func (s *secretService) Name() string { return "Secret Service" }

func (s *secretService) Available() (bool, string) {
	var owners []string
	err := s.conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&owners)
	if err != nil {
		return false, fmt.Sprintf("cannot query the session bus: %v", err)
	}
	for _, n := range owners {
		if n == ssBusName {
			return true, ""
		}
	}
	// Activatable but not yet running counts as available: the first call
	// starts it.
	var activatable []string
	if err := s.conn.BusObject().Call("org.freedesktop.DBus.ListActivatableNames", 0).Store(&activatable); err == nil {
		for _, n := range activatable {
			if n == ssBusName {
				return true, ""
			}
		}
	}
	return false, "no Secret Service provider is running on the session bus (gnome-keyring, KWallet or equivalent)"
}

// openSession negotiates a plain session. The bus is a local socket owned by
// the user; the DH algorithm protects against nothing extra here and adds a
// dependency on a crypto negotiation we would have to keep correct.
func (s *secretService) openSession() (dbus.BusObject, dbus.ObjectPath, error) {
	service := s.conn.Object(ssBusName, ssServicePath)
	var discard dbus.Variant
	var session dbus.ObjectPath
	err := service.Call(ssServiceIface+".OpenSession", 0, "plain", dbus.MakeVariant("")).
		Store(&discard, &session)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return service, session, nil
}

func (s *secretService) closeSession(session dbus.ObjectPath) {
	_ = s.conn.Object(ssBusName, session).Call("org.freedesktop.Secret.Session.Close", 0).Err
}

// attributes identify one of our items. The reference is the lookup key; the
// service name keeps us out of other applications' items.
func attributes(ref string) map[string]string {
	return map[string]string{"service": Service, "ref": ref}
}

func (s *secretService) Get(ref string) (Secret, error) {
	service, session, err := s.openSession()
	if err != nil {
		return Secret{}, err
	}
	defer s.closeSession(session)

	var unlocked, locked []dbus.ObjectPath
	if err := service.Call(ssServiceIface+".SearchItems", 0, attributes(ref)).
		Store(&unlocked, &locked); err != nil {
		return Secret{}, fmt.Errorf("secrets: search failed: %w", err)
	}

	items := unlocked
	if len(items) == 0 && len(locked) > 0 {
		// The agent owns the unlock interaction and its lifetime. We ask once
		// and never cache the result or re-prompt around it.
		if items, err = s.unlock(service, locked); err != nil {
			return Secret{}, err
		}
	}
	if len(items) == 0 {
		return Secret{}, ErrNotFound
	}

	var out dbusSecret
	if err := s.conn.Object(ssBusName, items[0]).
		Call(ssItemIface+".GetSecret", 0, session).Store(&out); err != nil {
		return Secret{}, fmt.Errorf("secrets: cannot read the stored secret: %w", err)
	}
	return NewSecret(out.Value), nil
}

func (s *secretService) Set(ref string, sec Secret) error {
	service, session, err := s.openSession()
	if err != nil {
		return err
	}
	defer s.closeSession(session)

	collection := s.conn.Object(ssBusName, ssDefaultAlias)
	if _, err := s.unlock(service, []dbus.ObjectPath{ssDefaultAlias}); err != nil {
		return err
	}

	props := map[string]dbus.Variant{
		ssItemIface + ".Label":      dbus.MakeVariant(Service + ": " + ref),
		ssItemIface + ".Attributes": dbus.MakeVariant(attributes(ref)),
	}
	value := dbusSecret{
		Session:     session,
		Value:       sec.Bytes(),
		ContentType: "application/octet-stream",
	}

	var item, prompt dbus.ObjectPath
	if err := collection.Call(ssCollectionIfce+".CreateItem", 0, props, value, true).
		Store(&item, &prompt); err != nil {
		return fmt.Errorf("secrets: cannot store the secret: %w", err)
	}
	if prompt != "/" {
		if _, err := s.prompt(prompt); err != nil {
			return err
		}
	}
	return nil
}

func (s *secretService) Delete(ref string) error {
	service, session, err := s.openSession()
	if err != nil {
		return err
	}
	defer s.closeSession(session)

	var unlocked, locked []dbus.ObjectPath
	if err := service.Call(ssServiceIface+".SearchItems", 0, attributes(ref)).
		Store(&unlocked, &locked); err != nil {
		return fmt.Errorf("secrets: search failed: %w", err)
	}
	for _, item := range append(unlocked, locked...) {
		var prompt dbus.ObjectPath
		if err := s.conn.Object(ssBusName, item).Call(ssItemIface+".Delete", 0).Store(&prompt); err != nil {
			return fmt.Errorf("secrets: cannot delete the secret: %w", err)
		}
		if prompt != "/" {
			if _, err := s.prompt(prompt); err != nil {
				return err
			}
		}
	}
	return nil
}

// unlock asks the agent to unlock paths, following a prompt if one is raised.
func (s *secretService) unlock(service dbus.BusObject, paths []dbus.ObjectPath) ([]dbus.ObjectPath, error) {
	var unlocked []dbus.ObjectPath
	var prompt dbus.ObjectPath
	if err := service.Call(ssServiceIface+".Unlock", 0, paths).Store(&unlocked, &prompt); err != nil {
		return nil, fmt.Errorf("secrets: unlock failed: %w", err)
	}
	if prompt == "/" {
		return unlocked, nil
	}
	result, err := s.prompt(prompt)
	if err != nil {
		return nil, err
	}
	if promptedPaths, ok := result.Value().([]dbus.ObjectPath); ok {
		return promptedPaths, nil
	}
	return unlocked, nil
}

// prompt drives one agent interaction to completion. A dismissed prompt is a
// recoverable outcome — the user declined, and the caller falls through to the
// session-only path.
func (s *secretService) prompt(path dbus.ObjectPath) (dbus.Variant, error) {
	if err := s.conn.AddMatchSignal(
		dbus.WithMatchObjectPath(path),
		dbus.WithMatchInterface(ssPromptIface),
		dbus.WithMatchMember("Completed"),
	); err != nil {
		return dbus.Variant{}, fmt.Errorf("secrets: cannot watch the agent prompt: %w", err)
	}
	signals := make(chan *dbus.Signal, 1)
	s.conn.Signal(signals)
	defer s.conn.RemoveSignal(signals)

	if err := s.conn.Object(ssBusName, path).Call(ssPromptIface+".Prompt", 0, "").Err; err != nil {
		return dbus.Variant{}, fmt.Errorf("secrets: cannot raise the agent prompt: %w", err)
	}

	timeout := time.NewTimer(promptTimeout)
	defer timeout.Stop()

	for {
		select {
		case sig := <-signals:
			if sig.Path != path || sig.Name != ssPromptIface+".Completed" {
				continue
			}
			if len(sig.Body) < 2 {
				return dbus.Variant{}, errors.New("secrets: malformed prompt response")
			}
			if dismissed, _ := sig.Body[0].(bool); dismissed {
				return dbus.Variant{}, fmt.Errorf("%w: the unlock prompt was dismissed", ErrUnavailable)
			}
			v, _ := sig.Body[1].(dbus.Variant)
			return v, nil
		case <-timeout.C:
			return dbus.Variant{}, fmt.Errorf("%w: the unlock prompt was not answered", ErrUnavailable)
		}
	}
}
