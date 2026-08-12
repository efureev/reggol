package reggol

import (
	"slices"
	"strings"
)

// sortFieldsLarge handles the rare event carrying many fields.
func sortFieldsLarge(fields []Field) {
	slices.SortStableFunc(fields, func(a, b Field) int {
		return strings.Compare(a.Key, b.Key)
	})
}
