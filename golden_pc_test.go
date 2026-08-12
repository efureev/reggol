package reggol

// goldenPC is the call site the golden matrix renders.
//
// It lives in a file of its own, and goldenCallSite stays on the last line, so
// that editing the matrix cannot shift the line number baked into every golden
// file. Without that, unrelated edits to golden_test.go would show up as a diff
// in testdata and train reviewers to regenerate without reading.
var goldenPC = goldenCallSite()

func goldenCallSite() uintptr { return capturePC(2) }
