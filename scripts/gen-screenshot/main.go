// Command gen-screenshot renders the console encoder's real output into an SVG
// for the README.
//
// Generating the image from the library itself, rather than photographing a
// terminal, means the picture cannot drift from the code: regenerate it with
// `go run ./scripts/gen-screenshot` whenever the format changes. The v0
// repository carried a hand-taken PNG that outlived the format it depicted.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/efureev/reggol"
)

const outPath = ".assets/console.svg"

// Terminal geometry and palette, close to a dark macOS terminal.
const (
	charW      = 8.4
	lineH      = 22.0
	padX       = 18.0
	padY       = 34.0
	fontSize   = 14.0
	titleBarH  = 28.0
	background = "#1d1f21"
	titleBar   = "#2b2d30"
	defaultFg  = "#c5c8c6"
)

// ansiPalette maps the SGR codes the console encoder emits onto hex colors.
var ansiPalette = map[string]string{
	"30": "#3b3f42", "31": "#cc6666", "32": "#b5bd68", "33": "#f0c674",
	"34": "#81a2be", "35": "#b294bb", "36": "#8abeb7", "37": "#c5c8c6",
	"90": "#6b7075", "91": "#d54e53", "92": "#b9ca4a", "93": "#e7c547",
	"94": "#7aa6da", "95": "#c397d8", "96": "#70c0b1", "97": "#eaeaea",
	"39": defaultFg,
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gen-screenshot:", err)
		os.Exit(1)
	}
}

func run() error {
	raw, err := sample()
	if err != nil {
		return err
	}

	svg := render(strings.Split(strings.TrimRight(raw, "\n"), "\n"))

	if err := os.WriteFile(outPath, []byte(svg), 0o644); err != nil { //nolint:gosec // a public asset
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	fmt.Printf("wrote %s\n", outPath)

	return nil
}

// sample produces a realistic set of records through the actual encoder.
func sample() (string, error) {
	var buf bytes.Buffer

	enc := reggol.NewConsoleEncoder(
		reggol.WithColorMode(reggol.ColorAlways, nil),
		reggol.WithConsoleOptions(reggol.WithTimeFormat("15:04:05")),
	)

	base := reggol.New(&buf, reggol.WithEncoder(enc), reggol.WithLevel(reggol.TraceLevel))

	prev := reggol.GlobalLevel()
	reggol.SetGlobalLevel(reggol.TraceLevel)

	defer reggol.SetGlobalLevel(prev)

	api := base.With().Str("service", "api").Logger()

	base.Info().Msg("server started")
	base.Debug().Str("addr", ":8080").Int("workers", 8).Msg("listening")
	api.Info().Blocks("GET", "/users").Int("status", 200).Dur("took", 12*time.Millisecond).Msg("handled")
	api.Warn().Blocks("GET", "/orders").Int("status", 429).Msg("rate limited")
	api.Error().
		Err(errors.New("connection refused")).
		Str("host", "db-1").
		Msg("query failed")
	base.Trace().Str("pool", "pg").Int("idle", 3).Msg("stats")

	return buf.String(), nil
}

// segment is a run of characters sharing one color.
type segment struct {
	text  string
	color string
}

// render turns ANSI-colored lines into an SVG document.
func render(lines []string) string {
	parsed := make([][]segment, 0, len(lines))
	width := 0

	for _, line := range lines {
		segs := parseANSI(line)
		parsed = append(parsed, segs)

		n := 0
		for _, s := range segs {
			n += len([]rune(s.text))
		}

		width = max(width, n)
	}

	w := padX*2 + float64(width)*charW
	h := titleBarH + padY + float64(len(parsed))*lineH

	var b strings.Builder

	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" `+
		`viewBox="0 0 %.0f %.0f" font-family="SFMono-Regular,Menlo,Consolas,monospace" font-size="%.0f">`,
		w, h, w, h, fontSize)

	fmt.Fprintf(&b, `<rect width="%.0f" height="%.0f" rx="10" fill="%s"/>`, w, h, background)
	fmt.Fprintf(&b, `<path d="M0 10a10 10 0 0 1 10-10h%.0fa10 10 0 0 1 10 10v%.0fH0z" fill="%s"/>`,
		w-20, titleBarH-10, titleBar)

	for i, c := range []string{"#ff5f57", "#febc2e", "#28c840"} {
		fmt.Fprintf(&b, `<circle cx="%.0f" cy="14" r="5.5" fill="%s"/>`, 18+float64(i)*18, c)
	}

	for i, segs := range parsed {
		y := titleBarH + padY + float64(i)*lineH - 6
		x := padX

		for _, s := range segs {
			if strings.TrimSpace(s.text) != "" {
				fmt.Fprintf(&b, `<text x="%.2f" y="%.2f" fill="%s" xml:space="preserve">%s</text>`,
					x, y, s.color, escapeXML(s.text))
			}

			x += float64(len([]rune(s.text))) * charW
		}
	}

	b.WriteString(`</svg>`)

	return b.String()
}

// parseANSI splits a line into colored segments, understanding the subset of
// SGR sequences the console encoder produces.
func parseANSI(line string) []segment {
	var (
		segs  []segment
		cur   strings.Builder
		color = defaultFg
	)

	flush := func() {
		if cur.Len() > 0 {
			segs = append(segs, segment{text: cur.String(), color: color})
			cur.Reset()
		}
	}

	for i := 0; i < len(line); {
		if line[i] != 0x1b || i+1 >= len(line) || line[i+1] != '[' {
			cur.WriteByte(line[i])
			i++

			continue
		}

		end := strings.IndexByte(line[i:], 'm')
		if end < 0 {
			cur.WriteByte(line[i])
			i++

			continue
		}

		flush()

		color = applySGR(color, line[i+2:i+end])
		i += end + 1
	}

	flush()

	return segs
}

// applySGR resolves an SGR parameter list to a color.
func applySGR(current, params string) string {
	for _, p := range strings.Split(params, ";") {
		switch p {
		case "", "0":
			current = defaultFg
		case "1", "22", "2", "3", "4":
			// Weight and style carry no color information here.
		default:
			if c, ok := ansiPalette[p]; ok {
				current = c
			} else if n, err := strconv.Atoi(p); err == nil && n >= 30 && n <= 97 {
				current = defaultFg
			}
		}
	}

	return current
}

func escapeXML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

	return r.Replace(s)
}
