package commands

import "strings"

// The application's command table.
//
// It is the single source for the menu bar, the context menus, the command
// palette and the keyboard-shortcut preference pane. A command that is not
// here is not invocable anywhere, which is the property FR-102's "fully
// keyboard operable" depends on.
//
// Two absences are deliberate, not oversights:
//   - There is no File › New › Server instance…: local server management is
//     out of scope for v1 (deviation D1).
//   - Edit › Undo and Edit › Redo are editor-local only. Reversing a change
//     that reached a server is Edit › Reverse committed change…, which goes
//     through history and is previewed like any other write (deviation D8).

// Enablement predicates, evaluated by the frontend against the current
// context. They are named constants so a typo is a compile error rather than a
// command that is silently never enabled.
const (
	Always             = "always"
	ConnectionOpen     = "connection.open"
	ConnectionClosed   = "connection.closed"
	ConnectionWritable = "connection.open && !connection.readOnly"
	ProfileSelected    = "profile.selected"
	EntrySelected      = "entry.selected"
	EntryWritable      = "entry.selected && !connection.readOnly"
	EditorOpen         = "editor.open"
	EditorDirty        = "editor.dirty"
	EditorUndoable     = "editor.canUndo"
	EditorRedoable     = "editor.canRedo"
	ResultsPresent     = "search.hasResults"
	HistoryReversible  = "history.selectedIsReversible"
	SchemaProjectOpen  = "schemaProject.open"
)

// bindings expands Mod to the platform's primary modifier — Cmd on macOS,
// Ctrl elsewhere — because writing every binding twice is how the two drift
// apart.
//
// darwinChord overrides the expansion where the obvious chord collides with
// one macOS owns: Ctrl+H is the familiar search shortcut, but Cmd+H hides the
// application, so macOS gets a different chord rather than a dead one.
func bindings(chord, darwinChord string) map[Platform]string {
	if chord == "" && darwinChord == "" {
		return nil
	}
	primary := NormaliseChord(strings.ReplaceAll(chord, "Mod", "Ctrl"))
	mac := NormaliseChord(strings.ReplaceAll(chord, "Mod", "Cmd"))
	if darwinChord != "" {
		mac = NormaliseChord(darwinChord)
	}

	out := make(map[Platform]string, 3)
	if primary != "" {
		out[PlatformLinux] = primary
		out[PlatformWindows] = primary
	}
	if mac != "" {
		out[PlatformDarwin] = mac
	}
	return out
}

type spec struct {
	id, label   string
	menu        Menu
	path        []string
	group       string
	order       int
	enablement  string
	chord       string
	darwinChord string
	scope       Scope
	destructive bool
}

func table() []spec {
	return []spec{
		// File
		{id: "file.new.connection", label: "LDAP Connection…", menu: MenuFile, path: []string{"New"}, enablement: Always, chord: "Mod+N"},
		{id: "file.new.folder", label: "Connection folder…", menu: MenuFile, path: []string{"New"}, enablement: Always},
		{id: "file.new.search", label: "Search…", menu: MenuFile, path: []string{"New"}, enablement: ConnectionOpen, chord: "Mod+Shift+N"},
		{id: "file.new.ldif", label: "LDIF file", menu: MenuFile, path: []string{"New"}, enablement: Always},
		{id: "file.new.schemaProject", label: "Schema project…", menu: MenuFile, path: []string{"New"}, enablement: Always},
		{id: "file.open", label: "Open file…", menu: MenuFile, group: "open", order: 10, enablement: Always, chord: "Mod+O"},
		{id: "file.save", label: "Save", menu: MenuFile, group: "save", order: 20, enablement: EditorDirty, chord: "Mod+S"},
		{id: "file.saveAs", label: "Save as…", menu: MenuFile, group: "save", order: 21, enablement: EditorOpen, chord: "Mod+Shift+S"},
		{id: "file.import", label: "Import…", menu: MenuFile, group: "transfer", order: 30, enablement: Always},
		{id: "file.export", label: "Export…", menu: MenuFile, group: "transfer", order: 31, enablement: Always},
		{id: "file.closeTab", label: "Close tab", menu: MenuFile, group: "close", order: 40, enablement: EditorOpen, chord: "Mod+W"},
		{id: "file.exit", label: "Exit", menu: MenuFile, group: "close", order: 41, enablement: Always, chord: "Mod+Q"},

		// Edit — undo and redo are editor-local (deviation D8).
		{id: "edit.undo", label: "Undo", menu: MenuEdit, group: "undo", order: 1, enablement: EditorUndoable, chord: "Mod+Z", scope: ScopeTextEditor},
		{id: "edit.redo", label: "Redo", menu: MenuEdit, group: "undo", order: 2, enablement: EditorRedoable, chord: "Mod+Shift+Z", scope: ScopeTextEditor},
		{id: "edit.reverse", label: "Reverse committed change…", menu: MenuEdit, group: "undo", order: 3, enablement: HistoryReversible, destructive: true},
		{id: "edit.cut", label: "Cut", menu: MenuEdit, group: "clipboard", order: 10, enablement: EditorOpen, chord: "Mod+X"},
		{id: "edit.copy", label: "Copy", menu: MenuEdit, group: "clipboard", order: 11, enablement: Always, chord: "Mod+C"},
		{id: "edit.paste", label: "Paste", menu: MenuEdit, group: "clipboard", order: 12, enablement: Always, chord: "Mod+V"},
		{id: "edit.copyDN", label: "Copy DN", menu: MenuEdit, group: "clipboard", order: 13, enablement: EntrySelected, chord: "Mod+Shift+C"},
		{id: "edit.rename", label: "Rename…", menu: MenuEdit, group: "entry", order: 20, enablement: EntryWritable, destructive: true},
		{id: "edit.move", label: "Move…", menu: MenuEdit, group: "entry", order: 21, enablement: EntryWritable, destructive: true},
		{id: "edit.delete", label: "Delete…", menu: MenuEdit, group: "entry", order: 22, enablement: EntryWritable, chord: "Delete", scope: ScopeTree, destructive: true},
		{id: "edit.properties", label: "Properties", menu: MenuEdit, group: "entry", order: 23, enablement: EntrySelected, chord: "Alt+Enter"},
		{id: "edit.find", label: "Find…", menu: MenuEdit, group: "find", order: 30, enablement: EditorOpen, chord: "Mod+F", scope: ScopeTextEditor},

		// Search
		{id: "search.new", label: "New search…", menu: MenuSearch, order: 1, enablement: ConnectionOpen, chord: "Mod+H", darwinChord: "Cmd+Shift+F"},
		{id: "search.filterEditor", label: "Filter editor…", menu: MenuSearch, order: 2, enablement: ConnectionOpen},
		{id: "search.goToDN", label: "Go to DN…", menu: MenuSearch, order: 3, enablement: ConnectionOpen, chord: "Mod+G"},
		{id: "search.history", label: "Search history", menu: MenuSearch, group: "saved", order: 10, enablement: ConnectionOpen},
		{id: "search.saved", label: "Saved searches", menu: MenuSearch, group: "saved", order: 11, enablement: ConnectionOpen},
		{id: "search.bookmarks", label: "Bookmarks", menu: MenuSearch, group: "saved", order: 12, enablement: ConnectionOpen},
		{id: "search.exportResults", label: "Export results…", menu: MenuSearch, group: "results", order: 20, enablement: ResultsPresent},

		// LDAP
		{id: "ldap.connect", label: "Connect", menu: MenuLDAP, group: "connection", order: 1, enablement: ProfileSelected},
		{id: "ldap.disconnect", label: "Disconnect", menu: MenuLDAP, group: "connection", order: 2, enablement: ConnectionOpen},
		{id: "ldap.reconnect", label: "Reconnect", menu: MenuLDAP, group: "connection", order: 3, enablement: ProfileSelected},
		{id: "ldap.refresh", label: "Refresh", menu: MenuLDAP, group: "connection", order: 4, enablement: ConnectionOpen, chord: "F5"},
		{id: "ldap.readOnly", label: "Open read-only", menu: MenuLDAP, group: "connection", order: 5, enablement: ProfileSelected},
		{id: "ldap.newEntry", label: "New entry…", menu: MenuLDAP, group: "entry", order: 10, enablement: ConnectionWritable},
		{id: "ldap.newFromExisting", label: "New entry from existing…", menu: MenuLDAP, group: "entry", order: 11, enablement: EntryWritable},
		{id: "ldap.deleteSubtree", label: "Delete subtree…", menu: MenuLDAP, group: "entry", order: 12, enablement: EntryWritable, destructive: true},
		{id: "ldap.copyMove", label: "Copy or move entries…", menu: MenuLDAP, group: "entry", order: 13, enablement: EntryWritable, destructive: true},
		{id: "ldap.batch", label: "Batch operation…", menu: MenuLDAP, group: "bulk", order: 20, enablement: ConnectionWritable, destructive: true},
		{id: "ldap.compare", label: "Compare…", menu: MenuLDAP, group: "bulk", order: 21, enablement: ConnectionOpen},
		{id: "ldap.passwordModify", label: "Modify password…", menu: MenuLDAP, group: "operations", order: 30, enablement: EntryWritable, destructive: true},
		{id: "ldap.whoAmI", label: "Who am I", menu: MenuLDAP, group: "operations", order: 31, enablement: ConnectionOpen},
		{id: "ldap.rootDSE", label: "Root DSE", menu: MenuLDAP, group: "operations", order: 32, enablement: ConnectionOpen},

		// Schema
		{id: "schema.browse", label: "Browse server schema", menu: MenuSchema, order: 1, enablement: ConnectionOpen},
		{id: "schema.typeHierarchy", label: "Open in type hierarchy", menu: MenuSchema, order: 2, enablement: SchemaProjectOpen, chord: "Mod+T"},
		{id: "schema.problems", label: "Problems", menu: MenuSchema, order: 3, enablement: SchemaProjectOpen},
		{id: "schema.mergeProject", label: "Merge from project…", menu: MenuSchema, group: "project", order: 10, enablement: SchemaProjectOpen},
		{id: "schema.export", label: "Export schema…", menu: MenuSchema, group: "project", order: 11, enablement: SchemaProjectOpen},
		{id: "schema.diff", label: "Compare schemas…", menu: MenuSchema, group: "project", order: 12, enablement: ConnectionOpen},

		// Credentials — a list of named references and their assignments. There
		// is no vault, no lock, no reveal (deviation D2).
		{id: "credentials.manage", label: "Manage credentials…", menu: MenuCredentials, order: 1, enablement: Always},
		{id: "credentials.new", label: "New credential…", menu: MenuCredentials, order: 2, enablement: Always},
		{id: "credentials.testBind", label: "Test bind", menu: MenuCredentials, order: 3, enablement: ProfileSelected},
		{id: "credentials.assignments", label: "Show assignments", menu: MenuCredentials, order: 4, enablement: Always},
		{id: "credentials.trustStore", label: "Certificate trust store…", menu: MenuCredentials, group: "trust", order: 10, enablement: Always},

		// Preferences — one command per pane (3a, 5a–5h).
		{id: "prefs.appearance", label: "Appearance", menu: MenuPreferences, order: 1, enablement: Always, chord: "Mod+,"},
		{id: "prefs.browser", label: "Browser & tree", menu: MenuPreferences, order: 2, enablement: Always},
		{id: "prefs.entryEditor", label: "Entry editor", menu: MenuPreferences, order: 3, enablement: Always},
		{id: "prefs.valueEditors", label: "Value editors", menu: MenuPreferences, order: 4, enablement: Always},
		{id: "prefs.textEditors", label: "LDIF & text editors", menu: MenuPreferences, order: 5, enablement: Always},
		{id: "prefs.connections", label: "Connections & timeouts", menu: MenuPreferences, order: 6, enablement: Always},
		{id: "prefs.security", label: "Credentials & security", menu: MenuPreferences, order: 7, enablement: Always},
		{id: "prefs.keyboard", label: "Keyboard shortcuts", menu: MenuPreferences, order: 8, enablement: Always},
		{id: "prefs.updates", label: "Updates & about", menu: MenuPreferences, order: 9, enablement: Always},

		// Help
		{id: "help.documentation", label: "Documentation", menu: MenuHelp, order: 1, enablement: Always, chord: "F1"},
		{id: "help.shortcuts", label: "Keyboard shortcut sheet", menu: MenuHelp, order: 2, enablement: Always},
		{id: "help.releaseNotes", label: "Release notes", menu: MenuHelp, order: 3, enablement: Always},
		{id: "help.about", label: "About Open LDAP Studio", menu: MenuHelp, order: 4, enablement: Always},
	}
}

// Default builds the registry every surface reads from.
func Default() *Registry {
	r := New()
	for _, s := range table() {
		scope := s.scope
		if scope == "" {
			scope = ScopeGlobal
		}
		r.MustRegister(Command{
			ID:          s.id,
			Label:       s.label,
			Menu:        s.menu,
			Path:        s.path,
			Group:       s.group,
			Order:       s.order,
			Scope:       scope,
			Enablement:  s.enablement,
			Bindings:    bindings(s.chord, s.darwinChord),
			Destructive: s.destructive,
		})
	}
	return r
}
