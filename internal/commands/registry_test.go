package commands

import (
	"strings"
	"testing"
)

// Contract C14 — every command has an enablement predicate and a
// non-conflicting default binding on every platform.

func TestEveryCommandHasAnEnablementPredicate(t *testing.T) {
	for _, c := range Default().All() {
		if strings.TrimSpace(c.Enablement) == "" {
			t.Errorf("%s has no enablement predicate", c.ID)
		}
	}
}

func TestDefaultBindingsDoNotConflictOnAnyPlatform(t *testing.T) {
	r := Default()
	for _, platform := range Platforms {
		conflicts := r.Conflicts(Keymap{Preset: PresetDefault, Platform: platform})
		for _, c := range conflicts {
			t.Errorf("%s: %s", platform, c.Error())
		}
	}
}

func TestNoDefaultBindingClaimsAnOSReservedChord(t *testing.T) {
	// Cmd+Q is reserved on macOS and File › Exit deliberately uses it there:
	// quitting is exactly what the OS chord does, so the platform handles it.
	// Every other reserved chord must be free.
	allowed := map[string]bool{"file.exit": true}

	for _, c := range Default().All() {
		for platform, chord := range c.Bindings {
			if allowed[c.ID] {
				continue
			}
			if reason := ReservedReason(platform, chord); reason != "" {
				t.Errorf("%s binds %s on %s, but %s", c.ID, chord, platform, reason)
			}
		}
	}
}

func TestEveryCommandBelongsToOneOfTheEightMenus(t *testing.T) {
	valid := make(map[Menu]bool, len(Menus))
	for _, m := range Menus {
		valid[m] = true
	}
	for _, c := range Default().All() {
		if c.Menu != "" && !valid[c.Menu] {
			t.Errorf("%s is in menu %q, which is not one of the eight", c.ID, c.Menu)
		}
	}
}

// Deviations D1 and D8 are absences, and an absence is only durable if
// something asserts it.
func TestThereIsNoServerInstanceCommand(t *testing.T) {
	for _, c := range Default().All() {
		if strings.Contains(strings.ToLower(c.Label), "server instance") {
			t.Errorf("%s exists; local server management is out of scope for v1 (D1)", c.ID)
		}
	}
}

func TestUndoIsEditorLocalAndReversalIsSeparate(t *testing.T) {
	r := Default()

	undo, ok := r.Get("edit.undo")
	if !ok {
		t.Fatal("edit.undo is missing")
	}
	if undo.Scope != ScopeTextEditor {
		t.Errorf("edit.undo scope is %s, want %s: entry-level undo is not what this command does (D8)",
			undo.Scope, ScopeTextEditor)
	}

	reverse, ok := r.Get("edit.reverse")
	if !ok {
		t.Fatal("edit.reverse is missing; reversing a dispatched write must be a distinct command (D8)")
	}
	if !reverse.Destructive {
		t.Error("edit.reverse is not marked destructive; it dispatches a write")
	}
}

func TestMenuTreeRendersEightMenusWithSubmenus(t *testing.T) {
	tree := Default().MenuTree(PlatformLinux)
	if len(tree) != 8 {
		t.Fatalf("menu tree has %d menus, want 8", len(tree))
	}

	var found bool
	for _, item := range tree[MenuFile] {
		if item.Label == "New" && len(item.Items) > 0 {
			found = true
			for _, sub := range item.Items {
				if strings.Contains(sub.Label, "Server instance") {
					t.Error("File › New still offers a server instance (D1)")
				}
			}
		}
	}
	if !found {
		t.Error("File › New submenu is missing")
	}
}

func TestChordNormalisationMakesEquivalentChordsEqual(t *testing.T) {
	cases := [][2]string{
		{"Shift+Ctrl+f", "Ctrl+Shift+F"},
		{"cmd+s", "Cmd+S"},
		{"CTRL+ALT+delete", "Ctrl+Alt+Delete"},
		{"F5", "F5"},
	}
	for _, c := range cases {
		if got := NormaliseChord(c[0]); got != c[1] {
			t.Errorf("NormaliseChord(%q) = %q, want %q", c[0], got, c[1])
		}
	}
}

func TestBindRefusesAReservedChordAndReportsWhy(t *testing.T) {
	r := Default()
	km := Keymap{Preset: PresetDefault, Platform: PlatformWindows}

	err := r.Bind(&km, "ldap.refresh", "Alt+F4")
	if err == nil {
		t.Fatal("binding Alt+F4 on Windows succeeded; the window manager owns it")
	}
	if !strings.Contains(err.Error(), "closes the window") {
		t.Errorf("refusal does not say why: %v", err)
	}
	if km.Bindings["ldap.refresh"] != "" {
		t.Error("the refused binding was applied anyway")
	}
}

func TestBindReportsAConflictAndLeavesTheKeymapUnchanged(t *testing.T) {
	r := Default()
	km := Keymap{Preset: PresetDefault, Platform: PlatformLinux}

	// Ctrl+O already belongs to file.open.
	err := r.Bind(&km, "ldap.refresh", "Ctrl+O")
	conflict, ok := err.(Conflict)
	if !ok {
		t.Fatalf("Bind returned %v, want a Conflict", err)
	}
	if len(conflict.CommandIDs) != 2 {
		t.Errorf("conflict names %v, want both commands", conflict.CommandIDs)
	}
	if _, bound := km.Bindings["ldap.refresh"]; bound {
		t.Error("the conflicting binding was applied; a conflict must leave the keymap as it was")
	}
}

func TestTheSameChordInTwoScopesIsNotAConflict(t *testing.T) {
	r := New()
	r.MustRegister(
		Command{
			ID: "a", Label: "A", Scope: ScopeTree, Enablement: Always,
			Bindings: map[Platform]string{PlatformLinux: "Ctrl+E"},
		},
		Command{
			ID: "b", Label: "B", Scope: ScopeTextEditor, Enablement: Always,
			Bindings: map[Platform]string{PlatformLinux: "Ctrl+E"},
		},
	)

	if conflicts := r.Conflicts(Keymap{Platform: PlatformLinux}); len(conflicts) != 0 {
		t.Errorf("reported %v; the same chord in different scopes is how a text editor gets its own meaning", conflicts)
	}
}

func TestRegisterRejectsACommandWithNoPredicate(t *testing.T) {
	r := New()
	err := r.Register(Command{ID: "x", Label: "X"})
	if err == nil {
		t.Fatal("a command with no enablement predicate was accepted")
	}
	if !strings.Contains(err.Error(), "enablement") {
		t.Errorf("error does not explain what is missing: %v", err)
	}
}

func TestRegisterRejectsADuplicateID(t *testing.T) {
	r := New()
	r.MustRegister(Command{ID: "x", Label: "X", Enablement: Always})
	if err := r.Register(Command{ID: "x", Label: "X again", Enablement: Always}); err == nil {
		t.Fatal("a duplicate command id was accepted")
	}
}

func TestSearchFindsCommandsByLabel(t *testing.T) {
	got := Default().Search("subtree")
	if len(got) == 0 {
		t.Fatal("searching for \"subtree\" found nothing")
	}
	for _, c := range got {
		if !strings.Contains(strings.ToLower(c.Label+c.ID), "subtree") {
			t.Errorf("%s does not match the query", c.ID)
		}
	}
}
