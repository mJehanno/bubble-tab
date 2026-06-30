package theme

import "image/color"

// Palette is a semantic six-color set used to render tab headers and bodies.
// Colors are stored as the standard image/color.Color interface; build them
// with lipgloss.Color (e.g. lipgloss.Color("#282828")). Each Theme bundles a
// Dark and a Light Palette; the Active method selects which one drives the
// derived Styles.
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

// Theme bundles a dark and a light Palette with per-state border configuration.
// IsDark selects the active palette variant. Use one of the preset constructors
// (Gruvbox, TokyoNight, Catppuccin) or build a custom Theme directly.
//
// Important: styles are derived from a Theme at the time New is called. If you
// toggle the variant at runtime (Toggle/WithDark), pass the updated Theme to a
// new TabModel via WithTheme so the styles are rebuilt. Mutating the Theme
// value after construction has no effect on an existing TabModel.
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

// Active returns the palette that is currently selected by IsDark: the Dark
// palette when IsDark is true, the Light palette otherwise.
func (t Theme) Active() Palette {
	if t.IsDark {
		return t.Dark
	}
	return t.Light
}

// WithDark returns a copy of the theme with IsDark set to dark. The receiver
// is not modified. Call Styles() (or pass the result to WithTheme and rebuild
// the TabModel) after toggling to see the new colors take effect.
func (t Theme) WithDark(dark bool) Theme {
	t.IsDark = dark
	return t
}

// Toggle returns a copy of the theme with IsDark flipped. The receiver is not
// modified. After toggling, call Styles() on the result and pass it to a new
// TabModel via WithTheme to apply the new palette.
func (t Theme) Toggle() Theme {
	t.IsDark = !t.IsDark
	return t
}

// Styles derives and returns the lipgloss styles for the theme's currently
// active palette. It is shorthand for New(t). Call this after Toggle or
// WithDark to obtain a Styles value reflecting the new palette.
func (t Theme) Styles() Styles {
	return New(t)
}
