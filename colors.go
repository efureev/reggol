package reggol

import (
	"fmt"

	"gh.tarampamp.am/colors"
)

const (
	colorFgBlack   colors.TextStyle = 1 << iota // Black text color
	colorFgRed                                  // Red text color
	colorFgGreen                                // Green text color
	colorFgYellow                               // Yellow text color
	colorFgBlue                                 // Blue text color
	colorFgMagenta                              // Magenta text color
	colorFgCyan                                 // Cyan text color
	colorFgWhite                                // White text color
	colorFgDefault                              // Default text color

	colorFgBright // Bright text color, usage example: (FgRed | FgBright).Wrap("hello world")

	colorBgBlack   // Black background color
	colorBgRed     // Red background color
	colorBgGreen   // Green background color
	colorBgYellow  // Yellow background color
	colorBgBlue    // Blue background color
	colorBgMagenta // Magenta background color
	colorBgCyan    // Cyan background color
	colorBgWhite   // White background color
	colorBgDefault // Default background color

	colorBgBright // Bright background color, usage example: (BgRed | BgBright).Wrap("hello world")

	colorBold      // Bold text
	colorFaint     // Faint text
	colorItalic    // Italic text
	colorUnderline // Underline text
	colorBlinking  // Blinking text
	colorReverse   // Reverse text
	colorInvisible // Invisible text
	colorStrike    // Strike text

	colorReset // Reset text style
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
