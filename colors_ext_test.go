package reggol

import (
	"bytes"
	"strings"
	"testing"
)

func TestColorConstructors(t *testing.T) {
	t.Run("256", func(t *testing.T) {
		c := Color256(203)

		got, ok := c.Index()
		if !ok || got != 203 {
			t.Fatalf("Index = %d, %v", got, ok)
		}

		if _, _, _, ok := c.RGB(); ok {
			t.Fatal("a palette color is not an RGB color")
		}
	})

	t.Run("rgb", func(t *testing.T) {
		c := ColorRGB(0x12, 0x34, 0x56)

		r, g, b, ok := c.RGB()
		if !ok || r != 0x12 || g != 0x34 || b != 0x56 {
			t.Fatalf("RGB = %#x %#x %#x, %v", r, g, b, ok)
		}

		if _, ok := c.Index(); ok {
			t.Fatal("an RGB color is not a palette color")
		}
	})

	t.Run("zero is unset", func(t *testing.T) {
		var c Color

		if _, ok := c.Index(); ok {
			t.Fatal("the zero color must not be a palette color")
		}

		if codes, ok := c.sgr(DepthTrueColor, sgrFgBase); ok || codes != nil {
			t.Fatalf("the zero color must render nothing, got %v", codes)
		}
	})
}

func TestStyleColorCodes(t *testing.T) {
	for _, tc := range []struct {
		name        string
		style       Style
		depth       ColorDepth
		start, stop string
	}{
		{
			name:  "256 foreground",
			style: Style{Fg: Color256(203)},
			depth: Depth256,
			start: "\x1b[38;5;203m", stop: "\x1b[39m",
		},
		{
			name:  "256 background",
			style: Style{Bg: Color256(17)},
			depth: Depth256,
			start: "\x1b[48;5;17m", stop: "\x1b[49m",
		},
		{
			name:  "truecolor foreground",
			style: Style{Fg: ColorRGB(0xff, 0x66, 0x00)},
			depth: DepthTrueColor,
			start: "\x1b[38;2;255;102;0m", stop: "\x1b[39m",
		},
		{
			name:  "truecolor both plus attributes",
			style: Style{Fg: ColorRGB(255, 0, 0), Bg: ColorRGB(0, 0, 255), Attrs: ColorBold},
			depth: DepthTrueColor,
			start: "\x1b[1;38;2;255;0;0;48;2;0;0;255m", stop: "\x1b[49;39;22m",
		},
		{
			name:  "named style still works",
			style: StyleOf(ColorFgGreen | ColorBold),
			depth: Depth16,
			start: "\x1b[1;32m", stop: "\x1b[39;22m",
		},
		{
			name:  "attributes only",
			style: Style{Attrs: ColorUnderline},
			depth: DepthTrueColor,
			start: "\x1b[4m", stop: "\x1b[24m",
		},
		{
			name:  "empty",
			style: Style{},
			depth: DepthTrueColor,
			start: "", stop: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start, stop := tc.style.ColorCodes(tc.depth)

			if start != tc.start || stop != tc.stop {
				t.Fatalf("\n  got: %q / %q\n want: %q / %q", start, stop, tc.start, tc.stop)
			}
		})
	}
}

// TestStyleDowngrade covers the reason depth exists at all: a color the
// terminal cannot express must become one it can, not escape codes printed as
// text.
func TestStyleDowngrade(t *testing.T) {
	red := Style{Fg: ColorRGB(255, 0, 0)}

	for _, tc := range []struct {
		depth ColorDepth
		want  string
	}{
		{DepthTrueColor, "\x1b[38;2;255;0;0m"},
		{Depth256, "\x1b[38;5;196m"},
		{Depth16, "\x1b[31m"},
	} {
		t.Run(tc.depth.String(), func(t *testing.T) {
			if start, _ := red.ColorCodes(tc.depth); start != tc.want {
				t.Fatalf("got %q, want %q", start, tc.want)
			}
		})
	}

	// A palette color on a basic terminal degrades too.
	if start, _ := (Style{Fg: Color256(203)}).ColorCodes(Depth16); !strings.HasPrefix(start, "\x1b[") ||
		strings.Contains(start, "38;5") {
		t.Fatalf("palette color was not reduced: %q", start)
	}
}

func TestIndexFromRGB(t *testing.T) {
	for _, tc := range []struct {
		name    string
		r, g, b uint8
		want    uint8
	}{
		{"pure red", 255, 0, 0, 196},
		{"pure green", 0, 255, 0, 46},
		{"pure blue", 0, 0, 255, 21},
		{"white", 255, 255, 255, 231},
		{"black", 0, 0, 0, 16},
		{"mid grey", 128, 128, 128, 244},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := indexFromRGB(tc.r, tc.g, tc.b); got != tc.want {
				t.Fatalf("indexFromRGB(%d,%d,%d) = %d, want %d", tc.r, tc.g, tc.b, got, tc.want)
			}
		})
	}
}

func TestNamedFromIndex(t *testing.T) {
	for _, tc := range []struct {
		name  string
		index uint8
		want  int
	}{
		{"basic red", 1, sgrFgBase + 1},
		{"bright red", 9, sgrFgBase + sgrBrightOffset + 1},
		{"cube red", 196, sgrFgBase + 1},
		{"cube white", 231, sgrFgBase + 7},
		{"dark grey", 234, sgrFgBase},
		{"light grey", 250, sgrFgBase + sgrBrightOffset + 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := namedFromIndex(tc.index, sgrFgBase); got != tc.want {
				t.Fatalf("namedFromIndex(%d) = %d, want %d", tc.index, got, tc.want)
			}
		})
	}
}

func TestDetectColorDepth(t *testing.T) {
	for _, tc := range []struct {
		name      string
		colorterm string
		term      string
		want      ColorDepth
	}{
		{"truecolor", "truecolor", "xterm", DepthTrueColor},
		{"24bit", "24bit", "xterm", DepthTrueColor},
		{"mixed case", "TrueColor", "xterm", DepthTrueColor},
		{"256 via TERM", "", "xterm-256color", Depth256},
		{"plain", "", "xterm", Depth16},
		{"nothing", "", "", Depth16},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("COLORTERM", tc.colorterm)
			t.Setenv("TERM", tc.term)

			if got := detectColorDepth(); got != tc.want {
				t.Fatalf("detectColorDepth = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSetLevelStyleOnEncoder(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")

	enc := NewConsoleEncoder(
		WithColorMode(ColorAlways, nil),
		WithColorDepth(DepthTrueColor),
		WithConsoleOptions(WithoutTimestamp()),
	)
	enc.SetLevelStyle(InfoLevel, Style{Fg: ColorRGB(255, 102, 0), Attrs: ColorBold})

	var buf bytes.Buffer

	New(&buf, WithEncoder(enc), WithLevel(TraceLevel)).Info().Msg("m")

	if got := buf.String(); !strings.Contains(got, "\x1b[1;38;2;255;102;0m") {
		t.Fatalf("truecolor level style not applied: %q", got)
	}
}

// TestSetLevelStyleDowngradesOnEncoder pins that the encoder's depth, not the
// style, decides what actually reaches the terminal.
func TestSetLevelStyleDowngradesOnEncoder(t *testing.T) {
	enc := NewConsoleEncoder(
		WithColorMode(ColorAlways, nil),
		WithColorDepth(Depth16),
		WithConsoleOptions(WithoutTimestamp()),
	)
	enc.SetLevelStyle(InfoLevel, Style{Fg: ColorRGB(255, 0, 0)})

	var buf bytes.Buffer

	New(&buf, WithEncoder(enc), WithLevel(TraceLevel)).Info().Msg("m")

	got := buf.String()
	if strings.Contains(got, "38;2") {
		t.Fatalf("truecolor leaked to a 16-color terminal: %q", got)
	}

	if !strings.Contains(got, "\x1b[31m") {
		t.Fatalf("expected the reduced basic color: %q", got)
	}
}

func TestColorDepthString(t *testing.T) {
	for _, tc := range []struct {
		depth ColorDepth
		want  string
	}{
		{DepthAuto, "auto"},
		{Depth16, "16"},
		{Depth256, "256"},
		{DepthTrueColor, "truecolor"},
		{ColorDepth(9), "ColorDepth(9)"},
	} {
		if got := tc.depth.String(); got != tc.want {
			t.Errorf("%v = %q, want %q", tc.depth, got, tc.want)
		}
	}
}
