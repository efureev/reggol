package reggol

import "testing"

func TestParseLevel_KnownStrings_CaseInsensitive(t *testing.T) {
	cases := []struct {
		in   string
		want Level
	}{
		{"trace", TraceLevel},
		{"DEBUG", DebugLevel},
		{"Info", InfoLevel},
		{"wArN", WarnLevel},
		{"error", ErrorLevel},
		{"fatal", FatalLevel},
		{"panic", PanicLevel},
		{"disabled", Disabled},
		{"", NoLevel}, // empty maps to NoLevel
	}

	for _, c := range cases {
		got, err := ParseLevel(c.in)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("%q => %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseLevel_NumericAndErrors(t *testing.T) {
	// numeric valid
	if got, err := ParseLevel("5"); err != nil || got != Level(5) {
		t.Fatalf("numeric parse failed: got %v err %v", got, err)
	}

	// unknown string
	if got, err := ParseLevel("nope"); err == nil || got != NoLevel {
		t.Fatalf("unknown string should error and return NoLevel, got %v err %v", got, err)
	}

	// out-of-bounds
	if got, err := ParseLevel("1000"); err == nil || got != NoLevel {
		t.Fatalf("oob numeric should error and NoLevel, got %v err %v", got, err)
	}
}

func TestLevelMarshalUnmarshalText(t *testing.T) {
	// Marshal
	if b, _ := InfoLevel.MarshalText(); string(b) != "info" {
		t.Fatalf("marshal expected 'info', got %q", string(b))
	}

	// Unmarshal into valid receiver
	var l Level
	if err := l.UnmarshalText([]byte("warn")); err != nil || l != WarnLevel {
		t.Fatalf("unmarshal warn failed: l=%v err=%v", l, err)
	}

	// Unmarshal into nil receiver must error
	var lp *Level
	if err := lp.UnmarshalText([]byte("info")); err == nil {
		t.Fatalf("expected error on nil receiver")
	}
}
