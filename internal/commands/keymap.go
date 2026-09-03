package commands

import (
	"fmt"
	"sort"
	"strings"
)

// Preset names a keymap the user can start from (screen 5g).
type Preset string

const (
	// PresetDefault is this application's own keymap.
	PresetDefault Preset = "default"
	// PresetEclipse matches Apache Directory Studio's Eclipse heritage, so
	// somebody migrating keeps their muscle memory.
	PresetEclipse Preset = "eclipse"
	// PresetCustom is whatever the user has made.
	PresetCustom Preset = "custom"
)

// Keymap is a set of bindings for one platform.
type Keymap struct {
	Preset   Preset   `json:"preset"`
	Platform Platform `json:"platform"`
	// Bindings maps command id to chord. A command absent here uses its
	// registered default; a command bound to "" is deliberately unbound.
	Bindings map[string]string `json:"bindings"`
}

// Conflict is two commands claiming one chord in the same scope.
type Conflict struct {
	Chord      string   `json:"chord"`
	Scope      Scope    `json:"scope"`
	CommandIDs []string `json:"commandIds"`
}

func (c Conflict) Error() string {
	return fmt.Sprintf("%s is bound to %s in the %s scope", c.Chord, strings.Join(c.CommandIDs, " and "), c.Scope)
}

// reservedChords are owned by the operating system or its window manager.
// Binding one produces a shortcut that silently never fires, which reads as a
// broken application rather than a rejected binding.
var reservedChords = map[Platform]map[string]string{
	PlatformDarwin: {
		"Cmd+Q":     "macOS quits the application",
		"Cmd+H":     "macOS hides the application",
		"Cmd+M":     "macOS minimises the window",
		"Cmd+Tab":   "macOS switches applications",
		"Cmd+Space": "macOS opens Spotlight",
	},
	PlatformWindows: {
		"Alt+F4":            "Windows closes the window",
		"Ctrl+Alt+Delete":   "Windows opens the security screen",
		"Ctrl+Shift+Escape": "Windows opens Task Manager",
		"Win+L":             "Windows locks the session",
	},
	PlatformLinux: {
		"Alt+F4":          "the window manager closes the window",
		"Ctrl+Alt+F1":     "the system switches virtual terminal",
		"Ctrl+Alt+Delete": "the desktop environment opens the logout dialog",
	},
}

// NormaliseChord puts a chord into the canonical form bindings are compared
// in: modifiers in a fixed order, each part capitalised.
//
// Without this, Ctrl+Shift+F and Shift+Ctrl+f are two bindings that both fire.
func NormaliseChord(chord string) string {
	if strings.TrimSpace(chord) == "" {
		return ""
	}
	parts := strings.Split(strings.ReplaceAll(chord, " ", ""), "+")

	modifierOrder := map[string]int{"Ctrl": 0, "Cmd": 1, "Alt": 2, "Shift": 3}
	var modifiers []string
	var key string

	for _, part := range parts {
		canonical := canonicalPart(part)
		if _, ok := modifierOrder[canonical]; ok {
			modifiers = append(modifiers, canonical)
			continue
		}
		key = canonical
	}
	sort.SliceStable(modifiers, func(i, j int) bool {
		return modifierOrder[modifiers[i]] < modifierOrder[modifiers[j]]
	})
	if key == "" {
		return strings.Join(modifiers, "+")
	}
	return strings.Join(append(modifiers, key), "+")
}

func canonicalPart(part string) string {
	switch strings.ToLower(part) {
	case "ctrl", "control":
		return "Ctrl"
	case "cmd", "command", "meta", "super":
		return "Cmd"
	case "alt", "option":
		return "Alt"
	case "shift":
		return "Shift"
	case "del", "delete":
		return "Delete"
	case "esc", "escape":
		return "Escape"
	case "enter", "return":
		return "Enter"
	}
	if len(part) == 1 {
		return strings.ToUpper(part)
	}
	return strings.ToUpper(part[:1]) + part[1:]
}

// ReservedReason returns why a chord cannot be bound on a platform, or "".
func ReservedReason(p Platform, chord string) string {
	normalised := NormaliseChord(chord)
	for reserved, reason := range reservedChords[p] {
		if NormaliseChord(reserved) == normalised {
			return reason
		}
	}
	return ""
}

// Resolve produces the effective chord for every command under a keymap.
func (r *Registry) Resolve(km Keymap) map[string]string {
	out := make(map[string]string)
	for _, c := range r.All() {
		chord := c.Bindings[km.Platform]
		if override, ok := km.Bindings[c.ID]; ok {
			chord = override
		}
		if chord = NormaliseChord(chord); chord != "" {
			out[c.ID] = chord
		}
	}
	return out
}

// Conflicts reports every chord claimed twice within one scope.
//
// The same chord in two different scopes is not a conflict: that is how a text
// editor gets its own meaning for a key without losing the global one.
func (r *Registry) Conflicts(km Keymap) []Conflict {
	effective := r.Resolve(km)

	type key struct {
		chord string
		scope Scope
	}
	claimed := make(map[key][]string)

	for id, chord := range effective {
		c, ok := r.Get(id)
		if !ok {
			continue
		}
		k := key{chord: chord, scope: c.Scope}
		claimed[k] = append(claimed[k], id)
	}

	var conflicts []Conflict
	for k, ids := range claimed {
		if len(ids) < 2 {
			continue
		}
		sort.Strings(ids)
		conflicts = append(conflicts, Conflict{Chord: k.chord, Scope: k.scope, CommandIDs: ids})
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Chord < conflicts[j].Chord })
	return conflicts
}

// Bind sets one binding, refusing a reserved chord and reporting any conflict
// it would create.
//
// A conflict is reported, not silently resolved: which of two commands the
// user meant is not ours to guess.
func (r *Registry) Bind(km *Keymap, commandID, chord string) error {
	if _, ok := r.Get(commandID); !ok {
		return fmt.Errorf("commands: no command %s", commandID)
	}
	normalised := NormaliseChord(chord)

	if reason := ReservedReason(km.Platform, normalised); reason != "" {
		return fmt.Errorf("commands: %s cannot be bound: %s", normalised, reason)
	}
	if km.Bindings == nil {
		km.Bindings = make(map[string]string)
	}

	previous, had := km.Bindings[commandID]
	km.Bindings[commandID] = normalised
	km.Preset = PresetCustom

	for _, conflict := range r.Conflicts(*km) {
		if conflict.Chord != normalised {
			continue
		}
		if had {
			km.Bindings[commandID] = previous
		} else {
			delete(km.Bindings, commandID)
		}
		return conflict
	}
	return nil
}
