package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/mJehanno/bubble-tab/pkg/model"
)

// Styles holds the precomputed lipgloss styles for every tab state, split into
// the header (tab title) and body (tab content) parts. Build it with New or
// Theme.Styles.
type Styles struct {
	// ActiveHeader styles the title of the active tab.
	ActiveHeader lipgloss.Style
	// ActiveBody styles the content area of the active tab.
	ActiveBody lipgloss.Style
	// InactiveHeader styles the titles of inactive tabs.
	InactiveHeader lipgloss.Style
	// InactiveBody styles the content area of inactive tabs.
	InactiveBody lipgloss.Style
	// DisabledHeader styles the titles of disabled tabs.
	DisabledHeader lipgloss.Style
	// DisabledBody styles the content area of disabled tabs.
	DisabledBody lipgloss.Style
}

// New builds the derived Styles for a theme from its active palette and border
// configuration. Header styles carry the state's border with palette-driven
// border and text colors; body styles carry a matching border plus foreground
// and background, so the content is always visibly themed.
func New(t Theme) Styles {
	p := t.Active()

	return Styles{
		ActiveHeader: headerStyle(t.ActiveBorder, p.Secondary, p.Primary),
		ActiveBody:   bodyStyle(t.ActiveBorder, p.Secondary, p.Primary, p.Background),

		InactiveHeader: headerStyle(t.InactiveBorder, p.Tertiary, p.Secondary),
		InactiveBody:   bodyStyle(t.InactiveBorder, p.Tertiary, p.Foreground, p.Background),

		DisabledHeader: headerStyle(t.DisabledBorder, p.Border, p.Tertiary),
		DisabledBody:   bodyStyle(t.DisabledBorder, p.Border, p.Tertiary, p.Background),
	}
}

// headerStyle builds a tab-title style with the given border and colors.
func headerStyle(b BorderType, border, text color.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(b.ToLipgloss()).
		BorderForeground(border).
		Foreground(text)
}

// bodyStyle builds a tab-content style. Unlike the old inert body branch, this
// applies a visible border plus foreground and background colors.
func bodyStyle(b BorderType, border, fg, bg color.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(b.ToLipgloss()).
		BorderForeground(border).
		Foreground(fg).
		Background(bg)
}

// Header returns the header style for the given tab state.
func (s Styles) Header(state model.TabState) lipgloss.Style {
	switch state {
	case model.Active:
		return s.ActiveHeader
	case model.Inactive:
		return s.InactiveHeader
	default:
		return s.DisabledHeader
	}
}

// Body returns the body style for the given tab state.
func (s Styles) Body(state model.TabState) lipgloss.Style {
	switch state {
	case model.Active:
		return s.ActiveBody
	case model.Inactive:
		return s.InactiveBody
	default:
		return s.DisabledBody
	}
}
