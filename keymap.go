package bubbletab

import "charm.land/bubbles/v2/key"

// KeyMap describes the key bindings used to navigate between tabs. Its fields
// are exposed as key.Binding values so they can be reused for help text or
// reconfigured by callers via WithKeyMap.
type KeyMap struct {
	// Next moves focus to the next non-disabled tab.
	Next key.Binding
	// Prev moves focus to the previous non-disabled tab.
	Prev key.Binding
	// Jump documents the 1-9 number keys that jump directly to a tab. The jump
	// itself is matched on the pressed rune in Update; this binding exists for
	// help and discoverability.
	Jump key.Binding
}

// DefaultKeyMap returns the standard tab navigation bindings: Tab to cycle
// forward, Shift+Tab to cycle backward, and 1-9 to jump to a tab directly.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Next: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		Prev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "previous tab"),
		),
		Jump: key.NewBinding(
			key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8", "9"),
			key.WithHelp("1-9", "jump to tab"),
		),
	}
}
