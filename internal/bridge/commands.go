package bridge

import (
	"runtime"

	"github.com/open-ldap-studio/open-ldap-studio/internal/commands"
)

func currentPlatform() commands.Platform {
	switch runtime.GOOS {
	case "darwin":
		return commands.PlatformDarwin
	case "windows":
		return commands.PlatformWindows
	default:
		return commands.PlatformLinux
	}
}

// CommandSet is everything the frontend needs to render menus, context menus
// and the shortcut pane from one registry.
type CommandSet struct {
	Platform commands.Platform                     `json:"platform"`
	Menus    []commands.Menu                       `json:"menus"`
	Tree     map[commands.Menu][]commands.MenuItem `json:"tree"`
	Commands []commands.Command                    `json:"commands"`
	Bindings map[string]string                     `json:"bindings"`
}

// GetCommands returns the command set for this platform.
func (b *Bridge) GetCommands() CommandSet {
	platform := currentPlatform()
	keymap := commands.Keymap{Preset: commands.PresetDefault, Platform: platform}
	return CommandSet{
		Platform: platform,
		Menus:    commands.Menus,
		Tree:     b.commands.MenuTree(platform),
		Commands: b.commands.All(),
		Bindings: b.commands.Resolve(keymap),
	}
}

// SearchCommands backs the command palette and the shortcut preference pane.
func (b *Bridge) SearchCommands(query string) []commands.Command {
	return b.commands.Search(query)
}

// BindKey rebinds a command, refusing an OS-reserved chord and reporting any
// conflict rather than silently resolving it.
func (b *Bridge) BindKey(keymap commands.Keymap, commandID, chord string) (commands.Keymap, error) {
	if keymap.Platform == "" {
		keymap.Platform = currentPlatform()
	}
	if err := b.commands.Bind(&keymap, commandID, chord); err != nil {
		return keymap, err
	}
	return keymap, nil
}

// KeymapConflicts lists every chord claimed twice in one scope.
func (b *Bridge) KeymapConflicts(keymap commands.Keymap) []commands.Conflict {
	if keymap.Platform == "" {
		keymap.Platform = currentPlatform()
	}
	return b.commands.Conflicts(keymap)
}
