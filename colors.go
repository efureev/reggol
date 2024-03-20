package reggol

import (
	"fmt"

	"gh.tarampamp.am/colors"
)

const (
	ColorFgBlack   colors.TextStyle = 1 << iota // Black text color
	ColorFgRed                                  // Red text color
	ColorFgGreen                                // Green text color
	ColorFgYellow                               // Yellow text color
	ColorFgBlue                                 // Blue text color
	ColorFgMagenta                              // Magenta text color
	ColorFgCyan                                 // Cyan text color
	ColorFgWhite                                // White text color
	ColorFgDefault                              // Default text color

	ColorFgBright // Bright text color, usage example: (FgRed | FgBright).Wrap("hello world")

	ColorBgBlack   // Black background color
	ColorBgRed     // Red background color
	ColorBgGreen   // Green background color
	ColorBgYellow  // Yellow background color
	ColorBgBlue    // Blue background color
	ColorBgMagenta // Magenta background color
	ColorBgCyan    // Cyan background color
	ColorBgWhite   // White background color
	ColorBgDefault // Default background color

	ColorBgBright // Bright background color, usage example: (BgRed | BgBright).Wrap("hello world")

	ColorBold      // Bold text
	ColorFaint     // Faint text
	ColorItalic    // Italic text
	ColorUnderline // Underline text
	ColorBlinking  // Blinking text
	ColorReverse   // Reverse text
	ColorInvisible // Invisible text
	ColorStrike    // Strike text

	ColorReset // Reset text style
)

func colorize(s interface{}, c colors.TextStyle, disabled bool) string {
	if c == 0 {
		disabled = true
	}

	str := fmt.Sprintf("%s", s)

	if disabled {
		return str
	}

	return c.Wrap(str)
}

func SetColor(s interface{}, c colors.TextStyle, disabled bool) string {
	return colorize(s, c, disabled)
}
