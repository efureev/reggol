package reggol

import (
	"testing"
)

func TestFields(t *testing.T) {
	t.Run(`Add Fields`, func(t *testing.T) {
		fields := make(Fields)

		if l := len(fields); l > 0 {
			t.Errorf("Fields should have no items, but have %d", len(fields))
		}

		fields.
			Add(`key1`, `value1`).
			Add(`key2`, `value2`)

		if l := len(fields); l != 2 {
			t.Errorf("Fields should have 2 item, but have %d", len(fields))
		}

		if val1, ok := fields[`key1`]; !ok || val1 != `value1` {
			t.Errorf("Fields should contain key `key1` with value `value1`")
		}
	})
}
