// Package theme defines the theming subsystem for bubble-tab: semantic color
// palettes, border styles, prebuilt themes, and the derived lipgloss styles
// used to render tabs.
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
