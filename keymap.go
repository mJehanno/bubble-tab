package bubbletab

import "charm.land/bubbles/v2/key"

// KeyMap describes the key bindings used to navigate between tabs. Each field
// is a key.Binding so the bindings can be integrated with a bubbles/v2 help
// component or overridden entirely via WithKeyMap.
type KeyMap struct {
	// Next moves focus to the next non-disabled tab, wrapping around.
	Next key.Binding
	// Prev moves focus to the previous non-disabled tab, wrapping around.
	Prev key.Binding
	// Jump documents the 1-9 number-key bindings that jump directly to a tab by
	// its one-based position. The matching logic lives in Update; this binding
	// exists for help text and discoverability rather than for dispatch.
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
