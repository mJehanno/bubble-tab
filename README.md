# bubble-tab

`bubble-tab` is a tabbed-navigation component for [Bubble Tea v2](https://charm.land/bubbletea/v2) (`charm.land/bubbletea/v2`). It renders a horizontal row of styled, clickable tab headers and delegates rendering and message handling to the body of the currently active tab. Each tab wraps any `tea.Model` as its body, preserving child state across switches and initializing bodies lazily so only tabs the user actually visits are ever allocated or run.

> Requires Go 1.25+ and `charm.land/bubbletea/v2`. Not compatible with the Bubble Tea v1 API.

## Install

```
go get github.com/mJehanno/bubble-tab
```

## Minimal complete example

The following wires three tabs into a `tea.Program`. The root model wraps `bubbletab.TabModel` and delegates `Init`, `Update`, and `View` to it, adding only a quit binding and the alternate-screen flag.

> **Note:** `TabModel` has value-receiver `tea.Model` methods, so its `Update` returns a `TabModel` *value*. Store the wrapped model by value and assert to `bubbletab.TabModel` (not `*bubbletab.TabModel`) — asserting to the pointer type panics at runtime. Dereference the `*TabModel` returned by `New` when building the wrapper.

```go
package main

import (
    "log"

    tea "charm.land/bubbletea/v2"
    bubbletab "github.com/mJehanno/bubble-tab"
    "github.com/mJehanno/bubble-tab/pkg/model"
    "github.com/mJehanno/bubble-tab/pkg/theme"
)

// homeModel, profileModel, settingsModel implement tea.Model.

type app struct{ tab bubbletab.TabModel }

func (a app) Init() tea.Cmd { return a.tab.Init() }

func (a app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "ctrl+c" {
        return a, tea.Quit
    }
    updated, cmd := a.tab.Update(msg)
    a.tab = updated.(bubbletab.TabModel)
    return a, cmd
}

func (a app) View() tea.View {
    v := a.tab.View()
    v.AltScreen = true
    return v
}

func main() {
    tabs := []model.Tab{
        *model.NewTab(model.WithName("Home"),     model.WithBody(homeModel{})),
        *model.NewTab(model.WithName("Profile"),  model.WithBody(profileModel{})),
        *model.NewTab(model.WithName("Settings"), model.WithBody(settingsModel{})),
    }

    root := app{
        tab: *bubbletab.New(
            bubbletab.WithTabs(tabs),
            bubbletab.WithTheme(theme.Catppuccin()),
        ),
    }

    if _, err := tea.NewProgram(root).Run(); err != nil {
        log.Fatal(err)
    }
}
```

A more complete runnable demo — with a counter tab, a notes tab, a lazy-loaded tab, and a disabled tab — lives in [`examples/demo/main.go`](examples/demo/main.go):

```
go run ./examples/demo
```

---

## Tabs and composition

**Each tab wraps a child `tea.Model` as its body.** `TabModel` forwards every non-navigation message to the active tab's body and persists the returned model, so child state accumulates normally across multiple `Update` calls. Non-active tab bodies never receive messages (except `WindowSizeMsg`, described below), which means their state is frozen until the tab is next activated — exactly the right behavior for a tabbed UI.

`WindowSizeMsg` is broadcast to every tab body that has already been constructed. This ensures that off-screen tabs are correctly sized before they are first displayed, preventing layout glitches on first paint.

```go
import "github.com/mJehanno/bubble-tab/pkg/model"

// Eager body — allocated at NewTab time; Init runs on first activation.
counterTab := model.NewTab(
    model.WithName("Counter"),
    model.WithBody(counterModel{}),
)

// Tab visible in the header but body hidden (e.g. permission-gated content).
adminTab := model.NewTab(
    model.WithName("Admin"),
    model.WithBody(adminModel{}),
    model.WithHasPermission(false),
)

// Tab that cannot be selected; skipped by all navigation.
disabledTab := model.NewTab(
    model.WithName("Coming Soon"),
    model.WithState(model.Disabled),
)
```

---

## Lazy loading

Two body-provisioning options exist with different trade-offs.

**`WithBody(tea.Model)`** — eager allocation. The model value is stored immediately in the `Tab`. Its `Init` still runs lazily (only when the tab is first activated), but the model is fully allocated from the start. Use this for lightweight bodies or bodies you have already constructed.

**`WithBodyFunc(func() tea.Model)`** — deferred construction. The factory function is called only when the tab is activated for the first time. A tab that is never visited is never constructed, saving all allocation and initialization costs. Use this for expensive bodies such as those that make network calls or parse large files.

```go
import "github.com/mJehanno/bubble-tab/pkg/model"

reportsTab := model.NewTab(
    model.WithName("Reports"),
    model.WithBodyFunc(func() tea.Model {
        // Called once, on first activation. Never called if tab is never visited.
        return newReportsModel()
    }),
)
```

Regardless of which option you use, a body's `Init` runs **exactly once** — on first activation. On subsequent visits the cached model is reused without re-running `Init`, so accumulated state (scroll position, loaded data, form inputs) is preserved across tab switches.

When a `WindowSizeMsg` arrived before a lazy body was built, `TabModel` automatically replays the cached terminal dimensions to the newly constructed body, so it is correctly sized on its first paint without waiting for the next resize event.

---

## Navigation

### Keyboard

| Key            | Action                                                                                                |
|----------------|-------------------------------------------------------------------------------------------------------|
| `Tab`          | Move to the next non-disabled tab (wraps around).                                                    |
| `Shift+Tab`    | Move to the previous non-disabled tab (wraps around).                                                |
| `1`–`9`        | Jump directly to the tab at that one-based position. Out-of-range or disabled targets are ignored.   |

### Mouse

Left-clicking a tab header activates that tab. The click handler is attached via the Bubble Tea v2 `tea.View.OnMouse` hook and performs offset-aware hit-testing against each header's rendered pixel span (including the borders when the theme draws them). Clicking in the body area below the headers has no effect.

By default, `TabModel` sets `MouseMode` to `tea.MouseModeCellMotion` on the view it returns. If your root model owns mouse configuration, override this with `WithMouseMode`:

```go
tm := bubbletab.New(
    bubbletab.WithTabs(tabs),
    bubbletab.WithMouseMode(tea.MouseModeNone), // parent model controls mouse
)
```

Your application must enable mouse reporting in the program for click events to be delivered. The standard approach is to set `v.MouseMode` on the root view (as `TabModel` does internally), which signals Bubble Tea v2 to request the appropriate terminal mouse mode.

### Custom key bindings

`DefaultKeyMap()` returns the built-in bindings. Override any binding by constructing a `KeyMap` and passing it via `WithKeyMap`. The `key.Binding` fields can also be integrated with a `bubbles/v2` help component to display a key-binding legend:

```go
import "charm.land/bubbles/v2/key"

km := bubbletab.DefaultKeyMap()
km.Next = key.NewBinding(
    key.WithKeys("l", "right"),
    key.WithHelp("l/→", "next tab"),
)
km.Prev = key.NewBinding(
    key.WithKeys("h", "left"),
    key.WithHelp("h/←", "prev tab"),
)

tm := bubbletab.New(
    bubbletab.WithTabs(tabs),
    bubbletab.WithKeyMap(km),
)
```

---

## Theming

`TabModel` uses a semantic `Palette` (six named colors: `Primary`, `Secondary`, `Tertiary`, `Foreground`, `Background`, `Border`) to derive lipgloss styles for each tab state (Active, Inactive, Disabled) and region (header and body). A `Theme` pairs a dark and a light `Palette` with per-state `BorderType` values.

### Built-in presets

All three presets default to the dark variant (`IsDark: true`).

```go
import "github.com/mJehanno/bubble-tab/pkg/theme"

bubbletab.New(bubbletab.WithTheme(theme.Gruvbox()))     // Gruvbox dark / light
bubbletab.New(bubbletab.WithTheme(theme.TokyoNight()))  // Tokyo Night / Tokyo Night Day
bubbletab.New(bubbletab.WithTheme(theme.Catppuccin()))  // Mocha (dark) / Latte (light)
```

#### Catppuccin flavors

Catppuccin ships with four named flavors. Use `CatppuccinPalette` to pick one for the dark or light slot:

```go
t := theme.Theme{
    IsDark:         true,
    ActiveBorder:   theme.Rounded,
    InactiveBorder: theme.Normal,
    DisabledBorder: theme.Hidden,
    Dark:           theme.CatppuccinPalette(theme.Macchiato),
    Light:          theme.CatppuccinPalette(theme.Latte),
}
```

The four flavors are `theme.Latte` (light), `theme.Frappe`, `theme.Macchiato`, and `theme.Mocha` (the darkest).

### Switching dark/light at runtime

`Toggle()` and `WithDark(bool)` return a new `Theme` value — neither modifies the receiver. **Styles are computed once, inside `New`.** To apply the change at runtime, rebuild the `TabModel` with the updated theme. Calling `Toggle` or `WithDark` without rebuilding has no visible effect because the previously derived styles remain in use:

```go
// Keep currentTheme and tabs in your root model.
currentTheme = currentTheme.Toggle()

tabModel = bubbletab.New(
    bubbletab.WithTabs(tabs),
    bubbletab.WithTheme(currentTheme),
)
```

### Padding, margin, and custom styles

Three options control the spacing and style of tab **headers** (Active, Inactive, and Disabled). The tab body is not affected by any of these options.

**`WithPadding(x, y int)`** — padding inside each header's border. `x` is the left/right padding in character columns; `y` is the top/bottom padding in rows.

**`WithMargin(x, y int)`** — margin outside each header's border, creating a gap between adjacent headers. `x` is the left/right margin; `y` is the top/bottom margin.

Both options layer on top of whichever base styles are active — whether those came from the theme or from `WithStyles`. Mouse click-to-activate stays correctly aligned automatically: the hit-test spans are computed from the rendered, already-spaced header widths, so no extra configuration is required.

```go
m := bubbletab.New(
    bubbletab.WithTabs(tabs),
    bubbletab.WithPadding(2, 0), // 2 cols left/right inside each tab header's border
    bubbletab.WithMargin(1, 0),  // 1 col gap between tab headers
)
```

**`WithStyles(theme.Styles)`** — replaces the model's styles wholesale with an explicit `Styles` value. When supplied, `New` uses these styles as-is instead of deriving them from the theme's palette. Build a base with `theme.Theme.Styles()` (or `theme.New(t)`), adjust the per-state `lipgloss.Style` fields you want to change, then pass the result. `WithStyles` does not merge with the theme; it replaces it entirely. Any `WithPadding`/`WithMargin` spacing is still layered on top afterwards.

```go
import (
    "charm.land/lipgloss/v2"
    bubbletab "github.com/mJehanno/bubble-tab"
    "github.com/mJehanno/bubble-tab/pkg/theme"
)

// Start from Catppuccin's derived styles, then override the active header.
styles := theme.Catppuccin().Styles()
styles.ActiveHeader = styles.ActiveHeader.Bold(true).Foreground(lipgloss.Color("#f38ba8"))

m := bubbletab.New(
    bubbletab.WithTabs(tabs),
    bubbletab.WithStyles(styles),
)
```

### Custom theme

Construct a `theme.Theme` directly. The `Palette` fields accept any `image/color.Color`; use `lipgloss.Color` for hex strings:

```go
import (
    "charm.land/lipgloss/v2"
    "github.com/mJehanno/bubble-tab/pkg/theme"
)

myTheme := theme.Theme{
    IsDark:         true,
    ActiveBorder:   theme.Rounded,
    InactiveBorder: theme.Normal,
    DisabledBorder: theme.Hidden,
    Dark: theme.Palette{
        Primary:    lipgloss.Color("#ff79c6"), // pink
        Secondary:  lipgloss.Color("#bd93f9"), // purple
        Tertiary:   lipgloss.Color("#6272a4"), // comment
        Foreground: lipgloss.Color("#f8f8f2"),
        Background: lipgloss.Color("#282a36"),
        Border:     lipgloss.Color("#44475a"),
    },
    Light: theme.Palette{ /* … */ },
}

tm := bubbletab.New(
    bubbletab.WithTabs(tabs),
    bubbletab.WithTheme(myTheme),
)
```

Available `BorderType` constants: `None`, `Normal`, `Rounded`, `Block`, `OuterHalf`, `InnerHalf`, `Thick`, `Double`, `Hidden`, `Markdown`, `Ascii`.

---

## API reference

### Package `bubbletab` (`github.com/mJehanno/bubble-tab`)

| Symbol | Description |
|--------|-------------|
| `TabModel` | Implements `tea.Model` (`Init`/`Update`/`View`). Construct with `New`. |
| `New(...TabModelOption) *TabModel` | Builds a `TabModel`. Defaults: Catppuccin theme, `DefaultKeyMap()`, `tea.MouseModeCellMotion`. Clamps current index; auto-activates the current tab. |
| `WithTabs([]model.Tab)` | Sets the ordered tab list. |
| `WithCurrent(int)` | Sets the initially active tab by zero-based index (clamped to valid range). |
| `WithTheme(theme.Theme)` | Sets the theme; styles are derived immediately in `New`. |
| `WithKeyMap(KeyMap)` | Overrides key bindings. |
| `WithMouseMode(tea.MouseMode)` | Overrides the `MouseMode` set on the returned view. |
| `WithStyles(theme.Styles)` | Replaces the model's styles wholesale; takes precedence over the theme (does not merge). Any `WithPadding`/`WithMargin` spacing is layered on top afterwards. |
| `WithPadding(x, y int)` | Padding inside each tab header's border (`x` = left/right, `y` = top/bottom). Applies to headers only; mouse hit-testing stays aligned automatically. |
| `WithMargin(x, y int)` | Margin outside each tab header's border (`x` = left/right gap between tabs, `y` = top/bottom). Applies to headers only; mouse hit-testing stays aligned automatically. |
| `KeyMap{Next, Prev, Jump key.Binding}` | Navigation bindings. |
| `DefaultKeyMap() KeyMap` | Returns bindings: Tab / Shift+Tab / 1–9. |

### Package `model` (`github.com/mJehanno/bubble-tab/pkg/model`)

| Symbol | Description |
|--------|-------------|
| `Tab` | A single tab entry pairing a name, state, and optional body. |
| `NewTab(...TabOption) *Tab` | Builds a tab. Defaults: empty name, `Inactive`, `hasPermission=true`, no body. |
| `WithName(string)` | Display label rendered in the header. |
| `WithState(TabState)` | Initial state; use `Disabled` to make a tab permanently non-navigable. |
| `WithBody(tea.Model)` | Eager body — stored immediately; `Init` deferred to first activation. |
| `WithBodyFunc(func() tea.Model)` | Lazy body — both allocation and `Init` deferred to first activation. |
| `WithHasPermission(bool)` | When `false`, the header renders but the body area is blank. |
| `TabState` | `Active`, `Inactive`, `Disabled`. |
| Getters | `Name()`, `State()`, `Body()`, `HasPermission()`, `Initialized()`. |
| Mutators | `SetState()`, `SetHasPermission()`, `SetInitialized()`, `SetBody()`, `EnsureBody()`. |

### Package `theme` (`github.com/mJehanno/bubble-tab/pkg/theme`)

| Symbol | Description |
|--------|-------------|
| `Palette` | Six semantic colors: `Primary`, `Secondary`, `Tertiary`, `Foreground`, `Background`, `Border`. |
| `Theme` | Dark + light `Palette` pair with `ActiveBorder`, `InactiveBorder`, `DisabledBorder`, and `IsDark`. |
| `Theme.Active() Palette` | Returns `Dark` when `IsDark` is true, `Light` otherwise. |
| `Theme.WithDark(bool) Theme` | Returns a copy with `IsDark` set; does not modify the receiver. |
| `Theme.Toggle() Theme` | Returns a copy with `IsDark` flipped; does not modify the receiver. |
| `Theme.Styles() Styles` | Derives lipgloss styles from the active palette. Shorthand for `New(t)`. |
| `Styles` | Precomputed per-state lipgloss styles. `Header(TabState)` and `Body(TabState)` dispatch methods. |
| `BorderType` | `None`, `Normal`, `Rounded`, `Block`, `OuterHalf`, `InnerHalf`, `Thick`, `Double`, `Hidden`, `Markdown`, `Ascii`. |
| `BorderType.ToLipgloss()` | Converts to a `lipgloss.Border`. |
| `Gruvbox() Theme` | Gruvbox dark / light preset. |
| `TokyoNight() Theme` | Tokyo Night / Day preset. |
| `Catppuccin() Theme` | Catppuccin Mocha (dark) / Latte (light) preset. |
| `CatppuccinFlavor` | `Latte`, `Frappe`, `Macchiato`, `Mocha`. |
| `CatppuccinPalette(CatppuccinFlavor) Palette` | Returns the palette for a specific Catppuccin flavor; unknown flavors fall back to Mocha. |

---

## Design rationale

The architecture choices — lazy init-once semantics, optional deferred construction via `WithBodyFunc`, correct `WindowSizeMsg` broadcast to all built bodies, offset-aware mouse hit-testing, and the dual dark/light semantic palette model — are documented in [`doc/adrs/ADR-001.md`](doc/adrs/ADR-001.md).
