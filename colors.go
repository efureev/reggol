package reggol

import (
	"fmt"

	"gh.tarampamp.am/colors"
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
