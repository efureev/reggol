//go:build !go1.21

package reggol

func clearMap(m Fields) {
	for k := range m {
		delete(m, k)
	}
}
