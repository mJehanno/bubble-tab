package theme

import "charm.land/lipgloss/v2"

// Gruvbox returns the Gruvbox theme with both its dark and light palettes,
// defaulting to the dark variant.
func Gruvbox() Theme {
	return Theme{
		IsDark:         true,
		ActiveBorder:   Rounded,
		InactiveBorder: Normal,
		DisabledBorder: Hidden,
		Dark: Palette{
			Primary:    lipgloss.Color("#fabd2f"), // bright yellow
			Secondary:  lipgloss.Color("#fe8019"), // bright orange
			Tertiary:   lipgloss.Color("#928374"), // gray
			Foreground: lipgloss.Color("#ebdbb2"), // fg
			Background: lipgloss.Color("#282828"), // bg0
			Border:     lipgloss.Color("#504945"), // bg2
		},
		Light: Palette{
			Primary:    lipgloss.Color("#b57614"), // dark yellow
			Secondary:  lipgloss.Color("#af3a03"), // dark orange
			Tertiary:   lipgloss.Color("#7c6f64"), // gray
			Foreground: lipgloss.Color("#3c3836"), // fg
			Background: lipgloss.Color("#fbf1c7"), // bg0
			Border:     lipgloss.Color("#d5c4a1"), // bg2
		},
	}
}

// TokyoNight returns the Tokyo Night theme, with the dark "Night" palette and
// the light "Tokyo Night Day" palette, defaulting to the dark variant.
func TokyoNight() Theme {
	return Theme{
		IsDark:         true,
		ActiveBorder:   Rounded,
		InactiveBorder: Normal,
		DisabledBorder: Hidden,
		Dark: Palette{
			Primary:    lipgloss.Color("#7aa2f7"), // blue
			Secondary:  lipgloss.Color("#bb9af7"), // magenta
			Tertiary:   lipgloss.Color("#565f89"), // comment
			Foreground: lipgloss.Color("#c0caf5"), // fg
			Background: lipgloss.Color("#1a1b26"), // bg
			Border:     lipgloss.Color("#24283b"), // bg highlight
		},
		Light: Palette{
			Primary:    lipgloss.Color("#2e7de9"), // blue
			Secondary:  lipgloss.Color("#9854f1"), // magenta
			Tertiary:   lipgloss.Color("#848cb5"), // comment
			Foreground: lipgloss.Color("#3760bf"), // fg
			Background: lipgloss.Color("#e1e2e7"), // bg
			Border:     lipgloss.Color("#c4c8da"), // bg highlight
		},
	}
}

// Catppuccin returns the Catppuccin theme using Mocha as its dark palette and
// Latte as its light palette, defaulting to the dark variant.
func Catppuccin() Theme {
	return Theme{
		IsDark:         true,
		ActiveBorder:   Rounded,
		InactiveBorder: Normal,
		DisabledBorder: Hidden,
		Dark:           CatppuccinPalette(Mocha),
		Light:          CatppuccinPalette(Latte),
	}
}

// CatppuccinFlavor names one of the four official Catppuccin flavors.
type CatppuccinFlavor string

// The four Catppuccin flavors. Latte is light; the rest are dark.
const (
	Latte     CatppuccinFlavor = "latte"
	Frappe    CatppuccinFlavor = "frappe"
	Macchiato CatppuccinFlavor = "macchiato"
	Mocha     CatppuccinFlavor = "mocha"
)

// CatppuccinPalette returns the Palette for the given Catppuccin flavor.
// Unknown flavors fall back to Mocha.
func CatppuccinPalette(f CatppuccinFlavor) Palette {
	switch f {
	case Latte:
		return Palette{
			Primary:    lipgloss.Color("#1e66f5"), // blue
			Secondary:  lipgloss.Color("#8839ef"), // mauve
			Tertiary:   lipgloss.Color("#6c6f85"), // subtext0
			Foreground: lipgloss.Color("#4c4f69"), // text
			Background: lipgloss.Color("#eff1f5"), // base
			Border:     lipgloss.Color("#ccd0da"), // surface0
		}
	case Frappe:
		return Palette{
			Primary:    lipgloss.Color("#8caaee"), // blue
			Secondary:  lipgloss.Color("#ca9ee6"), // mauve
			Tertiary:   lipgloss.Color("#a5adce"), // subtext0
			Foreground: lipgloss.Color("#c6d0f5"), // text
			Background: lipgloss.Color("#303446"), // base
			Border:     lipgloss.Color("#414559"), // surface0
		}
	case Macchiato:
		return Palette{
			Primary:    lipgloss.Color("#8aadf4"), // blue
			Secondary:  lipgloss.Color("#c6a0f6"), // mauve
			Tertiary:   lipgloss.Color("#a5adcb"), // subtext0
			Foreground: lipgloss.Color("#cad3f5"), // text
			Background: lipgloss.Color("#24273a"), // base
			Border:     lipgloss.Color("#363a4f"), // surface0
		}
	default: // Mocha and unknown.
		return Palette{
			Primary:    lipgloss.Color("#89b4fa"), // blue
			Secondary:  lipgloss.Color("#cba6f7"), // mauve
			Tertiary:   lipgloss.Color("#a6adc8"), // subtext0
			Foreground: lipgloss.Color("#cdd6f4"), // text
			Background: lipgloss.Color("#1e1e2e"), // base
			Border:     lipgloss.Color("#313244"), // surface0
		}
	}
}
