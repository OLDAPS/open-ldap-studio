package commands

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Menu is one of the eight top-level menus.
//
// There are eight, not nine: the Window menu is deliberately absent, and
// File › New › Server instance… went with the LDAP Servers perspective
// (deviation D1, screens.md § Shell).
type Menu string

const (
	MenuFile        Menu = "File"
	MenuEdit        Menu = "Edit"
	MenuSearch      Menu = "Search"
	MenuLDAP        Menu = "LDAP"
	MenuSchema      Menu = "Schema"
	MenuCredentials Menu = "Credentials"
	MenuPreferences Menu = "Preferences"
	MenuHelp        Menu = "Help"
)

// Menus is the fixed, ordered set.
var Menus = []Menu{
	MenuFile, MenuEdit, MenuSearch, MenuLDAP,
	MenuSchema, MenuCredentials, MenuPreferences, MenuHelp,
}

// Scope limits where a binding applies, so the same chord can mean different
// things in the tree and in a text editor without being a conflict.
type Scope string

const (
	ScopeGlobal     Scope = "global"
	ScopeTree       Scope = "tree"
	ScopeGrid       Scope = "grid"
	ScopeTextEditor Scope = "textEditor"
)

// Platform names a keymap's target.
type Platform string

const (
	PlatformLinux   Platform = "linux"
	PlatformDarwin  Platform = "darwin"
	PlatformWindows Platform = "windows"
)

// Platforms is every platform a default binding must be valid on.
var Platforms = []Platform{PlatformLinux, PlatformDarwin, PlatformWindows}

// Command is one invocable action. Menus, keymaps and context menus are three
// views of this one registry, which is why a command that is invocable one way
// is invocable every way (FR-102, design gap G4).
type Command struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Menu  Menu   `json:"menu,omitempty"`
	// Path places the command inside a submenu, e.g. ["New"].
	Path []string `json:"path,omitempty"`
	// Group separates blocks within a menu; a separator is drawn between them.
	Group string `json:"group,omitempty"`
	Order int    `json:"order"`
	Scope Scope  `json:"scope"`
	// Enablement is the predicate the frontend evaluates against the current
	// context. Every command has one — "always" is a valid answer, an empty
	// string is not (contract C14).
	Enablement string `json:"enablement"`
	// Bindings holds the default chord per platform. A command may have none,
	// but where it has one it must be valid on every platform it names.
	Bindings map[Platform]string `json:"bindings,omitempty"`
	// Destructive marks a command that reaches the confirm-and-preview path.
	// It is what the UI uses to style the item, not what enforces the preview:
	// that is the changeset boundary.
	Destructive bool `json:"destructive,omitempty"`
}

// Registry holds every command exactly once.
type Registry struct {
	mu       sync.RWMutex
	commands map[string]Command
	order    []string
}

// New returns an empty registry.
func New() *Registry {
	return &Registry{commands: make(map[string]Command)}
}

// Register adds a command, rejecting a duplicate id or a missing predicate.
func (r *Registry) Register(c Command) error {
	if c.ID == "" {
		return fmt.Errorf("commands: a command needs an id (%q)", c.Label)
	}
	if c.Label == "" {
		return fmt.Errorf("commands: %s needs a label", c.ID)
	}
	if c.Enablement == "" {
		return fmt.Errorf("commands: %s needs an enablement predicate; use \"always\" if it is always available", c.ID)
	}
	if c.Scope == "" {
		c.Scope = ScopeGlobal
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.commands[c.ID]; exists {
		return fmt.Errorf("commands: %s is already registered", c.ID)
	}
	r.commands[c.ID] = c
	r.order = append(r.order, c.ID)
	return nil
}

// MustRegister panics on a registration error. It is used only for the static
// command table, where a failure is a programming error caught at start-up.
func (r *Registry) MustRegister(commands ...Command) {
	for _, c := range commands {
		if err := r.Register(c); err != nil {
			panic(err)
		}
	}
}

// Get returns one command.
func (r *Registry) Get(id string) (Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.commands[id]
	return c, ok
}

// All returns every command in registration order.
func (r *Registry) All() []Command {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Command, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.commands[id])
	}
	return out
}

// MenuItem is one rendered entry: a command, or a submenu of them.
type MenuItem struct {
	CommandID   string     `json:"commandId,omitempty"`
	Label       string     `json:"label"`
	Chord       string     `json:"chord,omitempty"`
	Group       string     `json:"group,omitempty"`
	Enablement  string     `json:"enablement,omitempty"`
	Destructive bool       `json:"destructive,omitempty"`
	Items       []MenuItem `json:"items,omitempty"`
}

// MenuTree renders the menu bar for one platform, in order, with submenus.
func (r *Registry) MenuTree(p Platform) map[Menu][]MenuItem {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tree := make(map[Menu][]MenuItem, len(Menus))
	for _, menu := range Menus {
		var top []Command
		submenus := make(map[string][]Command)

		for _, id := range r.order {
			c := r.commands[id]
			if c.Menu != menu {
				continue
			}
			if len(c.Path) == 0 {
				top = append(top, c)
				continue
			}
			submenus[c.Path[0]] = append(submenus[c.Path[0]], c)
		}

		sort.SliceStable(top, func(i, j int) bool { return top[i].Order < top[j].Order })

		items := make([]MenuItem, 0, len(top)+len(submenus))
		for _, c := range top {
			items = append(items, item(c, p))
		}

		names := make([]string, 0, len(submenus))
		for name := range submenus {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			children := submenus[name]
			sort.SliceStable(children, func(i, j int) bool { return children[i].Order < children[j].Order })
			sub := MenuItem{Label: name}
			for _, c := range children {
				sub.Items = append(sub.Items, item(c, p))
			}
			items = append(items, sub)
		}

		tree[menu] = items
	}
	return tree
}

func item(c Command, p Platform) MenuItem {
	return MenuItem{
		CommandID:   c.ID,
		Label:       c.Label,
		Chord:       c.Bindings[p],
		Group:       c.Group,
		Enablement:  c.Enablement,
		Destructive: c.Destructive,
	}
}

// Search finds commands by label or id, for the command palette and the
// keyboard-shortcut preference pane (5g).
func (r *Registry) Search(query string) []Command {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return r.All()
	}
	var out []Command
	for _, c := range r.All() {
		if strings.Contains(strings.ToLower(c.Label), needle) ||
			strings.Contains(strings.ToLower(c.ID), needle) {
			out = append(out, c)
		}
	}
	return out
}
