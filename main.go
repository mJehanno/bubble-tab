package bubbletab

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mJehanno/bubble-tab/internal/model"
)

type theme struct {
	primary        string
	secundary      string
	tercary        string
	activeBorder   borderType
	inactiveBorder borderType
	disabledBorder borderType
}

type borderType string

func (b borderType) toLipglossBorder() lipgloss.Border {
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
	None      borderType = "none"
	Normal    borderType = "normal"
	Rounded   borderType = "rounded"
	Block     borderType = "block"
	OuterHalf borderType = "outerHalf"
	InnerHalf borderType = "innerHalf"
	Thick     borderType = "thick"
	Double    borderType = "double"
	Hidden    borderType = "hidden"
	Markdown  borderType = "markdown"
	Ascii     borderType = "ascii"
)

type (
	TabModel struct {
		tabs    []model.Tab
		current uint64
		theme   theme
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

func New(options ...TabModelOption) *TabModel {
	tabModel := new(TabModel)
	for _, o := range options {
		o(tabModel)
	}
	return tabModel
}

func (t TabModel) Init() tea.Cmd {
	return nil
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
		}
	}
	return t, nil
}

type tabPart string

const (
	head tabPart = "head"
	body tabPart = "body"
)

func (t theme) apply(state model.TabState, part tabPart, content string) string {
	newStyle := lipgloss.NewStyle()
	switch state {
	case model.Active:
		if part == head {
			return newStyle.
				BorderStyle(t.activeBorder.toLipglossBorder()).
				BorderForeground(lipgloss.Color(t.secundary)).
				Foreground(lipgloss.Color(t.primary)).
				Render(content)
		} else {
			return newStyle.Render(content)
		}
	case model.Inactive:
		if part == head {
			return newStyle.
				BorderStyle(t.inactiveBorder.toLipglossBorder()).
				BorderForeground(lipgloss.Color(t.tercary)).
				Foreground(lipgloss.Color(t.secundary)).
				Render(content)
		} else {
			return newStyle.Render(content)
		}
	default:
		if part == head {
			return newStyle.
				BorderStyle(t.disabledBorder.toLipglossBorder()).
				BorderForeground(lipgloss.White).
				Foreground(lipgloss.Color(t.tercary)).
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
	view.WriteString(t.theme.apply(t.tabs[t.current].State(), body, t.tabs[t.current].Body().View().Content))
	return tea.NewView(view.String())
}
