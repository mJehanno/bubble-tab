package theme

import "image/color"

// Palette is a semantic set of colors used to render tabs. Colors are stored as
// the standard image/color.Color interface; build them with lipgloss.Color
// (e.g. lipgloss.Color("#282828")).
type Palette struct {
	// Primary is the most prominent accent color, used for active text.
	Primary color.Color
	// Secondary is a supporting accent, used for active borders and inactive text.
	Secondary color.Color
	// Tertiary is a muted accent, used for inactive borders and disabled text.
	Tertiary color.Color
	// Foreground is the default text color.
	Foreground color.Color
	// Background is the default surface color.
	Background color.Color
	// Border is the default border color.
	Border color.Color
}

// Theme bundles a dark and a light Palette together with the border styles for
// each tab state. IsDark selects which palette Active returns.
type Theme struct {
	// Dark is the palette used when IsDark is true.
	Dark Palette
	// Light is the palette used when IsDark is false.
	Light Palette
	// IsDark selects the active palette variant.
	IsDark bool
	// ActiveBorder is the border drawn around the active tab's header.
	ActiveBorder BorderType
	// InactiveBorder is the border drawn around inactive tab headers.
	InactiveBorder BorderType
	// DisabledBorder is the border drawn around disabled tab headers.
	DisabledBorder BorderType
}

// Active returns the Dark palette when IsDark is true, otherwise the Light one.
func (t Theme) Active() Palette {
	if t.IsDark {
		return t.Dark
	}
	return t.Light
}

// WithDark returns a copy of the theme with IsDark set to the given value,
// selecting the dark or light palette variant.
func (t Theme) WithDark(dark bool) Theme {
	t.IsDark = dark
	return t
}

// Toggle returns a copy of the theme with the active palette variant flipped
// between dark and light.
func (t Theme) Toggle() Theme {
	t.IsDark = !t.IsDark
	return t
}

// Styles returns the derived lipgloss styles for the theme's active palette.
// It is shorthand for New(t).
func (t Theme) Styles() Styles {
	return New(t)
}
