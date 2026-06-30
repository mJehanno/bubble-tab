// Command demo is a runnable showcase of the bubbletab library. It wires up
// four tabs to exercise every feature:
//
//   - Counter   — an eager body (WithBody) that keeps its count across tab
//     switches, proving child state is preserved.
//   - Notes     — an eager body that appends typed runes, proving non-navigation
//     keys are forwarded to the active tab only.
//   - Lazy      — a deferred body (WithBodyFunc) that records when it was first
//     constructed, proving bodies are built lazily on first activation.
//   - Disabled  — a disabled tab that Tab/Shift+Tab cycling skips over.
//
// Navigation: Tab / Shift+Tab cycle, 1-9 jump directly, and clicking a header
// activates that tab. Ctrl+C or q quits.
//
// Styling: the tabs use the Gruvbox theme with WithPadding/WithMargin for
// breathing room and a gap between headers; mouse clicks stay aligned with the
// spacing applied.
package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	bubbletab "github.com/mJehanno/bubble-tab"
	"github.com/mJehanno/bubble-tab/pkg/model"
	"github.com/mJehanno/bubble-tab/pkg/theme"
)

// start marks program startup so the lazy tab can show how long after launch it
// was first built.
var start = time.Now()

// ---- Tab bodies -----------------------------------------------------------

// counter is an eager body whose state survives tab switches.
type counter struct{ n int }

func (c counter) Init() tea.Cmd { return nil }

func (c counter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "up", "+", "k":
			c.n++
		case "down", "-", "j":
			c.n--
		}
	}
	return c, nil
}

func (c counter) View() tea.View {
	return tea.NewView(fmt.Sprintf(
		"Count: %d\n\nPress ↑/+ to increment, ↓/- to decrement.\nSwitch tabs and come back — the count persists.",
		c.n,
	))
}

// notes is an eager body that accumulates typed characters.
type notes struct{ text strings.Builder }

func (n *notes) Init() tea.Cmd { return nil }

func (n *notes) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.Code {
		case tea.KeyBackspace:
			s := n.text.String()
			if len(s) > 0 {
				n.text.Reset()
				n.text.WriteString(s[:len(s)-1])
			}
		case tea.KeyEnter:
			n.text.WriteByte('\n')
		default:
			if r := key.String(); len(r) == 1 {
				n.text.WriteString(r)
			}
		}
	}
	return n, nil
}

func (n *notes) View() tea.View {
	body := n.text.String()
	if body == "" {
		body = "(start typing…)"
	}
	return tea.NewView("Notes — type freely; only this tab receives the keys:\n\n" + body)
}

// lazyBody records when it was constructed, demonstrating deferred creation.
type lazyBody struct{ builtAfter time.Duration }

func newLazyBody() tea.Model {
	return lazyBody{builtAfter: time.Since(start)}
}

func (l lazyBody) Init() tea.Cmd { return nil }

func (l lazyBody) Update(tea.Msg) (tea.Model, tea.Cmd) { return l, nil }

func (l lazyBody) View() tea.View {
	return tea.NewView(fmt.Sprintf(
		"This body was constructed lazily, %s after the program started —\n"+
			"not at launch. It is built once, on first activation, then cached.",
		l.builtAfter.Round(time.Millisecond),
	))
}

// ---- Root model -----------------------------------------------------------

// app wraps the TabModel so it can own program-level concerns: quitting and the
// alternate screen. (Mouse mode is enabled by the TabModel itself, so header
// clicks work without any extra setup here.)
//
// TabModel has value-receiver tea.Model methods, so its Update returns a
// TabModel value — store it back by value, not as a pointer.
type app struct {
	tab bubbletab.TabModel
}

func (a app) Init() tea.Cmd { return a.tab.Init() }

func (a app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "ctrl+c", "q":
			return a, tea.Quit
		}
	}
	updated, cmd := a.tab.Update(msg)
	a.tab = updated.(bubbletab.TabModel)
	return a, cmd
}

func (a app) View() tea.View {
	v := a.tab.View()
	v.AltScreen = true
	help := lipgloss.NewStyle().Faint(true).Render(
		"\n\ntab/shift+tab: cycle • 1-9: jump • click a header • q/ctrl+c: quit",
	)
	v.Content += help
	return v
}

// newApp builds the demo's root model with its four tabs.
func newApp() app {
	tabs := []model.Tab{
		*model.NewTab(
			model.WithName("Counter"),
			model.WithBody(counter{}),
		),
		*model.NewTab(
			model.WithName("Notes"),
			model.WithBody(&notes{}),
		),
		*model.NewTab(
			model.WithName("Lazy"),
			model.WithBodyFunc(newLazyBody),
		),
		*model.NewTab(
			model.WithName("Disabled"),
			model.WithState(model.Disabled),
			model.WithBody(counter{}),
		),
	}

	return app{
		tab: *bubbletab.New(
			bubbletab.WithTabs(tabs),
			bubbletab.WithTheme(theme.Gruvbox()),
			bubbletab.WithPadding(1, 0), // 1 col left/right inside each tab header
			bubbletab.WithMargin(1, 0),  // 1 col gap between tabs
		),
	}
}

func main() {
	if _, err := tea.NewProgram(newApp()).Run(); err != nil {
		log.Fatal(err)
	}
}
