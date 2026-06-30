// Package theme defines the theming subsystem for bubble-tab.
//
// The main types are:
//   - Palette — a semantic six-color set (primary, secondary, tertiary,
//     foreground, background, border).
//   - Theme — pairs a Dark and a Light Palette with per-state BorderType values
//     and exposes Active/Toggle/WithDark to select the active variant.
//   - Styles — precomputed lipgloss styles (header + body for each of the three
//     tab states) derived by calling Theme.Styles() or New(theme).
//   - BorderType — a string enum mapping to lipgloss border constructors.
//
// Three ready-to-use presets are provided: Gruvbox, TokyoNight, and Catppuccin,
// each with dark and light palette variants (IsDark=true by default).
// Catppuccin additionally exposes CatppuccinPalette(flavor) for per-flavor
// palette access.
package theme

import "charm.land/lipgloss/v2"

// BorderType names a border style independently of lipgloss, so themes can be
// declared without importing rendering details.
type BorderType string

// Supported border types. Each maps to a lipgloss border constructor via
// ToLipgloss; None renders no border at all.
const (
	None      BorderType = "none"
	Normal    BorderType = "normal"
	Rounded   BorderType = "rounded"
	Block     BorderType = "block"
	OuterHalf BorderType = "outerHalf"
	InnerHalf BorderType = "innerHalf"
	Thick     BorderType = "thick"
	Double    BorderType = "double"
	Hidden    BorderType = "hidden"
	Markdown  BorderType = "markdown"
	Ascii     BorderType = "ascii"
)

// ToLipgloss converts the BorderType to its corresponding lipgloss.Border.
// Unknown or None values yield an empty (no-op) border.
func (b BorderType) ToLipgloss() lipgloss.Border {
	switch b {
	case Normal:
		return lipgloss.NormalBorder()
	case Rounded:
		return lipgloss.RoundedBorder()
	case Block:
		return lipgloss.BlockBorder()
	case OuterHalf:
		return lipgloss.OuterHalfBlockBorder()
	case InnerHalf:
		return lipgloss.InnerHalfBlockBorder()
	case Thick:
		return lipgloss.ThickBorder()
	case Double:
		return lipgloss.DoubleBorder()
	case Hidden:
		return lipgloss.HiddenBorder()
	case Markdown:
		return lipgloss.MarkdownBorder()
	case Ascii:
		return lipgloss.ASCIIBorder()
	default: // None and unknown values.
		return lipgloss.Border{}
	}
}
