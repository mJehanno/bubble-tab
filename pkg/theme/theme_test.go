package theme

// White-box tests for the theme package. Same package for full access.

import (
	"fmt"
	"image/color"
	"testing"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/mJehanno/bubble-tab/pkg/model"
)

// ---- helpers ----

// colorEqual compares two color.Color values by their RGBA components.
func colorEqual(a, b color.Color) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

func paletteEqual(a, b Palette) bool {
	return colorEqual(a.Primary, b.Primary) &&
		colorEqual(a.Secondary, b.Secondary) &&
		colorEqual(a.Tertiary, b.Tertiary) &&
		colorEqual(a.Foreground, b.Foreground) &&
		colorEqual(a.Background, b.Background) &&
		colorEqual(a.Border, b.Border)
}

// ---- BorderType.ToLipgloss ----

func TestToLipgloss_KnownBorderTypes(t *testing.T) {
	tests := []struct {
		bt       BorderType
		wantSome bool // whether Top should be non-empty
	}{
		{Normal, true},
		{Rounded, true},
		{Block, true},
		{OuterHalf, true},
		{InnerHalf, true},
		{Thick, true},
		{Double, true},
		{Hidden, true},
		{Markdown, true},
		{Ascii, true},
	}
	for _, tt := range tests {
		t.Run(string(tt.bt), func(t *testing.T) {
			b := tt.bt.ToLipgloss()
			hasContent := b.Top != "" || b.Bottom != "" || b.Left != "" || b.Right != ""
			if hasContent != tt.wantSome {
				t.Errorf("BorderType(%q).ToLipgloss() hasContent=%v, want %v", tt.bt, hasContent, tt.wantSome)
			}
		})
	}
}

func TestToLipgloss_None_ReturnsEmptyBorder(t *testing.T) {
	b := None.ToLipgloss()
	empty := lipgloss.Border{}
	if b != empty {
		t.Errorf("None.ToLipgloss() = %+v, want empty border", b)
	}
}

func TestToLipgloss_UnknownValue_ReturnsEmptyBorder(t *testing.T) {
	for _, unknown := range []BorderType{"", "invalid", "ROUNDED", "nOne"} {
		t.Run(string(unknown), func(t *testing.T) {
			b := unknown.ToLipgloss()
			empty := lipgloss.Border{}
			if b != empty {
				t.Errorf("BorderType(%q).ToLipgloss() = %+v, want empty", unknown, b)
			}
		})
	}
}

// ---- Theme.Active ----

func TestTheme_Active_DarkMode(t *testing.T) {
	g := Gruvbox()
	g.IsDark = true
	p := g.Active()
	if !paletteEqual(p, g.Dark) {
		t.Error("Active() should return Dark palette when IsDark=true")
	}
}

func TestTheme_Active_LightMode(t *testing.T) {
	g := Gruvbox()
	g.IsDark = false
	p := g.Active()
	if !paletteEqual(p, g.Light) {
		t.Error("Active() should return Light palette when IsDark=false")
	}
}

// ---- Theme.WithDark — non-mutating ----

func TestTheme_WithDark_NonMutating(t *testing.T) {
	original := Gruvbox() // IsDark=true
	if !original.IsDark {
		t.Fatal("precondition: Gruvbox starts as dark")
	}

	lightCopy := original.WithDark(false)

	// Original unchanged.
	if !original.IsDark {
		t.Error("WithDark mutated the receiver: original.IsDark should still be true")
	}
	// Returned copy reflects the change.
	if lightCopy.IsDark {
		t.Error("WithDark(false) returned copy should have IsDark=false")
	}
	// Dark/Light palettes are preserved in the copy.
	if !paletteEqual(lightCopy.Dark, original.Dark) {
		t.Error("WithDark(false) should not change the Dark palette in the copy")
	}
}

func TestTheme_WithDark_True_NonMutating(t *testing.T) {
	original := Gruvbox().WithDark(false) // start as light
	darkCopy := original.WithDark(true)

	if original.IsDark {
		t.Error("WithDark(true) mutated the receiver")
	}
	if !darkCopy.IsDark {
		t.Error("WithDark(true) returned copy should have IsDark=true")
	}
}

// ---- Theme.Toggle — non-mutating ----

func TestTheme_Toggle_NonMutating(t *testing.T) {
	original := Gruvbox() // IsDark=true
	toggled := original.Toggle()

	if !original.IsDark {
		t.Error("Toggle mutated the receiver: original.IsDark should still be true")
	}
	if toggled.IsDark {
		t.Error("Toggle() should flip IsDark: toggled.IsDark should be false")
	}
}

func TestTheme_Toggle_Twice_ReturnsToDark(t *testing.T) {
	g := Gruvbox()
	twice := g.Toggle().Toggle()
	if !twice.IsDark {
		t.Error("two Toggles should return IsDark=true")
	}
}

func TestTheme_Toggle_FromLight(t *testing.T) {
	light := Gruvbox().WithDark(false)
	toggled := light.Toggle()
	if !toggled.IsDark {
		t.Error("Toggle from light should produce dark")
	}
	if light.IsDark {
		t.Error("Toggle mutated the light copy receiver")
	}
}

// ---- Theme.Styles shorthand ----

func TestTheme_Styles_EqualToNew(t *testing.T) {
	g := Gruvbox()
	// Styles() calls New(t) — just verify it doesn't panic and returns something.
	s := g.Styles()
	// If Active header has a non-nil underlying type, we're good.
	// We verify by rendering — panic would indicate failure.
	_ = s.ActiveHeader.Render("test")
	_ = s.InactiveHeader.Render("test")
	_ = s.DisabledHeader.Render("test")
}

// ---- Styles.Header / Styles.Body dispatch ----

func TestStyles_Header_DispatchToCorrectStyle(t *testing.T) {
	s := Catppuccin().Styles()

	// Active and Inactive should not equal each other (different colors/borders).
	active := fmt.Sprintf("%v", s.Header(model.Active))
	inactive := fmt.Sprintf("%v", s.Header(model.Inactive))
	disabled := fmt.Sprintf("%v", s.Header(model.Disabled))
	unknown := fmt.Sprintf("%v", s.Header(model.TabState("unknown-state")))

	if active == inactive {
		t.Error("Active header style should differ from Inactive")
	}
	// Default branch (unknown) maps to DisabledHeader.
	if unknown != disabled {
		t.Errorf("Unknown state Header() = %q, want disabled style %q", unknown, disabled)
	}
}

func TestStyles_Body_DispatchToCorrectStyle(t *testing.T) {
	s := Catppuccin().Styles()

	activeBody := fmt.Sprintf("%v", s.Body(model.Active))
	inactiveBody := fmt.Sprintf("%v", s.Body(model.Inactive))
	disabledBody := fmt.Sprintf("%v", s.Body(model.Disabled))
	unknownBody := fmt.Sprintf("%v", s.Body(model.TabState("bogus")))

	if activeBody == inactiveBody {
		t.Error("Active body style should differ from Inactive")
	}
	if unknownBody != disabledBody {
		t.Errorf("Unknown state Body() = %q, want disabled body style %q", unknownBody, disabledBody)
	}
}

// ---- Preset constructors: IsDark=true, non-zero palettes ----

func TestPresets_DefaultToDark(t *testing.T) {
	presets := []struct {
		name  string
		theme Theme
	}{
		{"Gruvbox", Gruvbox()},
		{"TokyoNight", TokyoNight()},
		{"Catppuccin", Catppuccin()},
	}
	for _, tt := range presets {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.theme.IsDark {
				t.Errorf("%s: IsDark should be true by default", tt.name)
			}
		})
	}
}

func TestPresets_NonZeroPalettes(t *testing.T) {
	themes := map[string]Theme{
		"Gruvbox":    Gruvbox(),
		"TokyoNight": TokyoNight(),
		"Catppuccin": Catppuccin(),
	}
	for name, theme := range themes {
		t.Run(name+"/Dark", func(t *testing.T) {
			p := theme.Dark
			if p.Primary == nil {
				t.Errorf("%s Dark palette Primary is nil", name)
			}
			if p.Background == nil {
				t.Errorf("%s Dark palette Background is nil", name)
			}
		})
		t.Run(name+"/Light", func(t *testing.T) {
			p := theme.Light
			if p.Primary == nil {
				t.Errorf("%s Light palette Primary is nil", name)
			}
			if p.Background == nil {
				t.Errorf("%s Light palette Background is nil", name)
			}
		})
	}
}

func TestPresets_HaveBorderTypes(t *testing.T) {
	for _, theme := range []Theme{Gruvbox(), TokyoNight(), Catppuccin()} {
		if theme.ActiveBorder == "" {
			t.Error("preset missing ActiveBorder")
		}
		if theme.InactiveBorder == "" {
			t.Error("preset missing InactiveBorder")
		}
		// DisabledBorder may be Hidden (non-empty string).
		if theme.DisabledBorder == "" {
			t.Error("preset missing DisabledBorder")
		}
	}
}

// ---- CatppuccinPalette flavors ----

func TestCatppuccinPalette_AllFlavorsNonZero(t *testing.T) {
	flavors := []CatppuccinFlavor{Latte, Frappe, Macchiato, Mocha}
	for _, f := range flavors {
		t.Run(string(f), func(t *testing.T) {
			p := CatppuccinPalette(f)
			if p.Primary == nil {
				t.Errorf("flavor %q: Primary is nil", f)
			}
			if p.Background == nil {
				t.Errorf("flavor %q: Background is nil", f)
			}
		})
	}
}

func TestCatppuccinPalette_UnknownFlavor_FallsBackToMocha(t *testing.T) {
	mocha := CatppuccinPalette(Mocha)
	for _, bad := range []CatppuccinFlavor{"", "invalid", "LATTE"} {
		t.Run(string(bad), func(t *testing.T) {
			got := CatppuccinPalette(bad)
			if !paletteEqual(got, mocha) {
				t.Errorf("unknown flavor %q: got %+v, want Mocha %+v", bad, got, mocha)
			}
		})
	}
}

func TestCatppuccinPalette_FlavorsAreDifferent(t *testing.T) {
	latte := CatppuccinPalette(Latte)
	frappe := CatppuccinPalette(Frappe)
	macchiato := CatppuccinPalette(Macchiato)
	mocha := CatppuccinPalette(Mocha)

	if paletteEqual(latte, mocha) {
		t.Error("Latte and Mocha palettes should differ")
	}
	if paletteEqual(frappe, macchiato) {
		t.Error("Frappe and Macchiato palettes should differ")
	}
}

// ---- Catppuccin() preset uses correct flavors ----

func TestCatppuccin_UsesMochaForDarkAndLatteForLight(t *testing.T) {
	cat := Catppuccin()
	mocha := CatppuccinPalette(Mocha)
	latte := CatppuccinPalette(Latte)

	if !paletteEqual(cat.Dark, mocha) {
		t.Error("Catppuccin().Dark should equal Mocha palette")
	}
	if !paletteEqual(cat.Light, latte) {
		t.Error("Catppuccin().Light should equal Latte palette")
	}
}

// ---- Styles built at New-time use Active palette ----

func TestNew_StylesUseDarkPaletteWhenIsDark(t *testing.T) {
	dark := Gruvbox() // IsDark=true
	light := dark.WithDark(false)

	darkStyles := New(dark)
	lightStyles := New(light)

	// Active header styles differ because palettes differ.
	ds := fmt.Sprintf("%v", darkStyles.ActiveHeader)
	ls := fmt.Sprintf("%v", lightStyles.ActiveHeader)
	if ds == ls {
		t.Error("Styles built from dark and light themes should differ for ActiveHeader")
	}
}

// ---- Toggle + Styles rebuild ----

func TestTheme_ToggleThenStyles_ReflectsNewPalette(t *testing.T) {
	dark := Gruvbox()
	light := dark.Toggle() // IsDark=false
	// Must rebuild styles from toggled theme to see light palette.
	darkStyles := dark.Styles()
	lightStyles := light.Styles()

	if fmt.Sprintf("%v", darkStyles.ActiveHeader) == fmt.Sprintf("%v", lightStyles.ActiveHeader) {
		t.Error("dark and light Styles() should produce different ActiveHeader styles")
	}
}

// ---- BorderType constant values ----

func TestBorderTypeConstants(t *testing.T) {
	tests := []struct {
		bt   BorderType
		want string
	}{
		{None, "none"},
		{Normal, "normal"},
		{Rounded, "rounded"},
		{Block, "block"},
		{OuterHalf, "outerHalf"},
		{InnerHalf, "innerHalf"},
		{Thick, "thick"},
		{Double, "double"},
		{Hidden, "hidden"},
		{Markdown, "markdown"},
		{Ascii, "ascii"},
	}
	for _, tt := range tests {
		if string(tt.bt) != tt.want {
			t.Errorf("BorderType constant: got %q, want %q", tt.bt, tt.want)
		}
	}
}

// ---- CatppuccinFlavor constants ----

func TestCatppuccinFlavorConstants(t *testing.T) {
	if Latte != "latte" {
		t.Errorf("Latte = %q, want 'latte'", Latte)
	}
	if Frappe != "frappe" {
		t.Errorf("Frappe = %q, want 'frappe'", Frappe)
	}
	if Macchiato != "macchiato" {
		t.Errorf("Macchiato = %q, want 'macchiato'", Macchiato)
	}
	if Mocha != "mocha" {
		t.Errorf("Mocha = %q, want 'mocha'", Mocha)
	}
}

// ---- Gruvbox dark vs light palettes differ ----

func TestGruvbox_DarkAndLightPalettesAreDifferent(t *testing.T) {
	g := Gruvbox()
	if paletteEqual(g.Dark, g.Light) {
		t.Error("Gruvbox Dark and Light palettes should differ")
	}
}

func TestTokyoNight_DarkAndLightPalettesAreDifferent(t *testing.T) {
	tn := TokyoNight()
	if paletteEqual(tn.Dark, tn.Light) {
		t.Error("TokyoNight Dark and Light palettes should differ")
	}
}
