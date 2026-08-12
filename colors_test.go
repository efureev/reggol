package reggol

import "testing"

// TestColorCodesMatchReference is the proof that dropping gh.tarampamp.am/colors
// changed nothing.
//
// Every expectation below was captured from that package while it was still a
// dependency, so this table is the archived implementation's own output frozen
// in place. It is not a description of what reggol does — it is the contract
// reggol had to reproduce byte for byte.
func TestColorCodesMatchReference(t *testing.T) {
	for _, tc := range []struct {
		name        string
		style       TextStyle
		start, stop string
	}{
		{"0", 0, "", ""},
		{"ColorFgBlack", ColorFgBlack, "\x1b[30m", "\x1b[39m"},
		{"ColorFgRed", ColorFgRed, "\x1b[31m", "\x1b[39m"},
		{"ColorFgGreen", ColorFgGreen, "\x1b[32m", "\x1b[39m"},
		{"ColorFgYellow", ColorFgYellow, "\x1b[33m", "\x1b[39m"},
		{"ColorFgBlue", ColorFgBlue, "\x1b[34m", "\x1b[39m"},
		{"ColorFgMagenta", ColorFgMagenta, "\x1b[35m", "\x1b[39m"},
		{"ColorFgCyan", ColorFgCyan, "\x1b[36m", "\x1b[39m"},
		{"ColorFgWhite", ColorFgWhite, "\x1b[37m", "\x1b[39m"},
		{"ColorFgDefault", ColorFgDefault, "\x1b[39m", ""},
		{"ColorFgBright", ColorFgBright, "", ""},
		{"ColorBgBlack", ColorBgBlack, "\x1b[40m", "\x1b[49m"},
		{"ColorBgRed", ColorBgRed, "\x1b[41m", "\x1b[49m"},
		{"ColorBgGreen", ColorBgGreen, "\x1b[42m", "\x1b[49m"},
		{"ColorBgYellow", ColorBgYellow, "\x1b[43m", "\x1b[49m"},
		{"ColorBgBlue", ColorBgBlue, "\x1b[44m", "\x1b[49m"},
		{"ColorBgMagenta", ColorBgMagenta, "\x1b[45m", "\x1b[49m"},
		{"ColorBgCyan", ColorBgCyan, "\x1b[46m", "\x1b[49m"},
		{"ColorBgWhite", ColorBgWhite, "\x1b[47m", "\x1b[49m"},
		{"ColorBgDefault", ColorBgDefault, "\x1b[49m", ""},
		{"ColorBgBright", ColorBgBright, "", ""},
		{"ColorBold", ColorBold, "\x1b[1m", "\x1b[22m"},
		{"ColorFaint", ColorFaint, "\x1b[2m", "\x1b[22m"},
		{"ColorItalic", ColorItalic, "\x1b[3m", "\x1b[23m"},
		{"ColorUnderline", ColorUnderline, "\x1b[4m", "\x1b[24m"},
		{"ColorBlinking", ColorBlinking, "\x1b[5m", "\x1b[25m"},
		{"ColorReverse", ColorReverse, "\x1b[7m", "\x1b[27m"},
		{"ColorInvisible", ColorInvisible, "\x1b[8m", "\x1b[28m"},
		{"ColorStrike", ColorStrike, "\x1b[9m", "\x1b[29m"},
		{"ColorReset", ColorReset, "\x1b[0m", ""},
		{"ColorFgGreen | ColorBold", ColorFgGreen | ColorBold, "\x1b[1;32m", "\x1b[39;22m"},
		{"ColorFgYellow | ColorBold", ColorFgYellow | ColorBold, "\x1b[1;33m", "\x1b[39;22m"},
		{"ColorFgRed | ColorBold", ColorFgRed | ColorBold, "\x1b[1;31m", "\x1b[39;22m"},
		{"ColorFgBlack | ColorFgBright", ColorFgBlack | ColorFgBright, "\x1b[90m", "\x1b[39m"},
		{"ColorFgRed | ColorFgBright", ColorFgRed | ColorFgBright, "\x1b[91m", "\x1b[39m"},
		{"ColorBgBlue | ColorBgBright", ColorBgBlue | ColorBgBright, "\x1b[104m", "\x1b[49m"},
		{"ColorFgWhite | ColorBgRed", ColorFgWhite | ColorBgRed, "\x1b[37;41m", "\x1b[49;39m"},
		{"ColorFgWhite | ColorBgRed | ColorBold | ColorUnderline", ColorFgWhite | ColorBgRed | ColorBold | ColorUnderline, "\x1b[1;4;37;41m", "\x1b[49;39;24;22m"},
		{"ColorBold | ColorFaint | ColorItalic | ColorUnderline | ColorBlinking | ColorReverse | ColorInvisible | ColorStrike", ColorBold | ColorFaint | ColorItalic | ColorUnderline | ColorBlinking | ColorReverse | ColorInvisible | ColorStrike, "\x1b[1;2;3;4;5;7;8;9m", "\x1b[29;28;27;25;24;23;22;22m"},
		{"ColorReset | ColorFgRed", ColorReset | ColorFgRed, "\x1b[0m", ""},
		{"ColorFgDefault | ColorBgDefault", ColorFgDefault | ColorBgDefault, "\x1b[39;49m", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start, stop := tc.style.ColorCodes()

			if start != tc.start || stop != tc.stop {
				t.Fatalf("ColorCodes()\n  got: %q / %q\n want: %q / %q",
					start, stop, tc.start, tc.stop)
			}
		})
	}
}

// TestTextStyleValues pins the numeric values.
//
// They are part of the API: reggol now defines TextStyle itself, and anyone who
// stored a style as a number is entitled to have it keep meaning the same thing.
func TestTextStyleValues(t *testing.T) {
	for _, tc := range []struct {
		name  string
		style TextStyle
		want  uint32
	}{
		{"ColorFgBlack", ColorFgBlack, 1},
		{"ColorFgRed", ColorFgRed, 2},
		{"ColorFgGreen", ColorFgGreen, 4},
		{"ColorFgYellow", ColorFgYellow, 8},
		{"ColorFgBlue", ColorFgBlue, 16},
		{"ColorFgMagenta", ColorFgMagenta, 32},
		{"ColorFgCyan", ColorFgCyan, 64},
		{"ColorFgWhite", ColorFgWhite, 128},
		{"ColorFgDefault", ColorFgDefault, 256},
		{"ColorFgBright", ColorFgBright, 512},
		{"ColorBgBlack", ColorBgBlack, 1024},
		{"ColorBgRed", ColorBgRed, 2048},
		{"ColorBgGreen", ColorBgGreen, 4096},
		{"ColorBgYellow", ColorBgYellow, 8192},
		{"ColorBgBlue", ColorBgBlue, 16384},
		{"ColorBgMagenta", ColorBgMagenta, 32768},
		{"ColorBgCyan", ColorBgCyan, 65536},
		{"ColorBgWhite", ColorBgWhite, 131072},
		{"ColorBgDefault", ColorBgDefault, 262144},
		{"ColorBgBright", ColorBgBright, 524288},
		{"ColorBold", ColorBold, 1048576},
		{"ColorFaint", ColorFaint, 2097152},
		{"ColorItalic", ColorItalic, 4194304},
		{"ColorUnderline", ColorUnderline, 8388608},
		{"ColorBlinking", ColorBlinking, 16777216},
		{"ColorReverse", ColorReverse, 33554432},
		{"ColorInvisible", ColorInvisible, 67108864},
		{"ColorStrike", ColorStrike, 134217728},
		{"ColorReset", ColorReset, 268435456},
	} {
		if got := uint32(tc.style); got != tc.want {
			t.Errorf("%s = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestTextStyleHasAddRemove(t *testing.T) {
	s := ColorFgRed

	if !s.Has(ColorFgRed) || s.Has(ColorBold) {
		t.Fatalf("Has is wrong for %d", s)
	}

	s.Add(ColorBold, ColorUnderline)

	if !s.Has(ColorBold) || !s.Has(ColorUnderline) {
		t.Fatalf("Add did not set the styles: %d", s)
	}

	s.Remove(ColorUnderline)

	if s.Has(ColorUnderline) {
		t.Fatalf("Remove did not clear the style: %d", s)
	}

	if !s.Has(ColorFgRed) || !s.Has(ColorBold) {
		t.Fatalf("Remove cleared too much: %d", s)
	}
}
