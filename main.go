package bubbletab

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mJehanno/bubble-tab/pkg/model"
)

type Theme struct {
	Primary        string
	Secundary      string
	Tercary        string
	ActiveBorder   BorderType
	InactiveBorder BorderType
	DisabledBorder BorderType
}

type BorderType string

func (b BorderType) toLipglossBorder() lipgloss.Border {
	switch b {
	case Normal:
		return lipgloss.NormalBorder()
	case Rounded:
		return lipgloss.RoundedBorder()
	case OuterHalf:
		return lipgloss.OuterHalfBlockBorder()
	case InnerHalf:
		return lipgloss.InnerHalfBlockBorder()
	case Double:
		return lipgloss.DoubleBorder()
	case Block:
		return lipgloss.BlockBorder()
	case Hidden:
		return lipgloss.HiddenBorder()
	case Markdown:
		return lipgloss.MarkdownBorder()
	case Ascii:
		return lipgloss.ASCIIBorder()
	case Thick:
		return lipgloss.ThickBorder()
	default:
		return lipgloss.Border{}
	}
}

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

type (
	TabModel struct {
		tabs    []model.Tab
		current uint64
		theme   Theme
	}
	TabModelOption func(*TabModel)
)

func WithTabs(tabs []model.Tab) TabModelOption {
	return func(tm *TabModel) {
		tm.tabs = tabs
	}
}

func WithCurrent(current uint64) TabModelOption {
	return func(tm *TabModel) {
		tm.current = current
	}
}

func WithTheme(theme Theme) TabModelOption {
	return func(tm *TabModel) {
		tm.theme = theme
	}
}

func New(options ...TabModelOption) *TabModel {
	tabModel := new(TabModel)
	for _, o := range options {
		o(tabModel)
	}
	return tabModel
}

func (t TabModel) Init() tea.Cmd {
	return t.tabs[t.current].Body().Init()
}

func (t *TabModel) moveForward() {
	nextIndex := t.current + 1
	for {
		if nextIndex == uint64(len(t.tabs)) {
			nextIndex = 0
		}

		if t.tabs[nextIndex].State() != model.Disabled {
			break
		}
		nextIndex++
	}
	t.current = nextIndex
}

func (t *TabModel) moveBackward() {
	nextIndex := t.current - 1
	for {
		if nextIndex == 0 {
			nextIndex = uint64(len(t.tabs) - 1)
		} else {
			nextIndex -= 1
		}

		if t.tabs[nextIndex].State() != model.Disabled {
			break
		}
		nextIndex--
	}
	t.current = nextIndex
}

func (t TabModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.Code == tea.KeyTab {
			t.tabs[t.current].SetState(model.Inactive)
			if msg.Mod == tea.ModAlt {
				t.moveBackward()
			} else {
				t.moveForward()
			}
			t.tabs[t.current].SetState(model.Active)
			return t, t.tabs[t.current].Body().Init()
		}
	}
	return t, nil
}

type tabPart string

const (
	head tabPart = "head"
	body tabPart = "body"
)

func (t Theme) apply(state model.TabState, part tabPart, content string) string {
	newStyle := lipgloss.NewStyle()
	switch state {
	case model.Active:
		if part == head {
			return newStyle.
				BorderStyle(t.ActiveBorder.toLipglossBorder()).
				BorderForeground(lipgloss.Color(t.Secundary)).
				Foreground(lipgloss.Color(t.Primary)).
				Render(content)
		} else {
			return newStyle.Render(content)
		}
	case model.Inactive:
		if part == head {
			return newStyle.
				BorderStyle(t.InactiveBorder.toLipglossBorder()).
				BorderForeground(lipgloss.Color(t.Tercary)).
				Foreground(lipgloss.Color(t.Secundary)).
				Render(content)
		} else {
			return newStyle.Render(content)
		}
	default:
		if part == head {
			return newStyle.
				BorderStyle(t.DisabledBorder.toLipglossBorder()).
				BorderForeground(lipgloss.White).
				Foreground(lipgloss.Color(t.Tercary)).
				Render(content)
		} else {
			return newStyle.Render(content)
		}
	}
}

func (t TabModel) View() tea.View {
	view := new(strings.Builder)
	for tab := range slices.Values(t.tabs) {
		view.WriteString(t.theme.apply(tab.State(), head, tab.Name()))
	}
	view.WriteString("\n")
	if t.tabs[t.current].HasPermission() {
		view.WriteString(t.theme.apply(t.tabs[t.current].State(), body, t.tabs[t.current].Body().View().Content))
	}
	return tea.NewView(view.String())
}
