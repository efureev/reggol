package reggol

import (
	"runtime"
	"strconv"
)

// Frame counts for runtime.Callers.
//
// runtime.Callers counts frame 0 as itself and frame 1 as its caller, and it
// counts inlined frames as logical frames — so these numbers hold regardless of
// what the compiler decides to inline.
//
// callerSkipEvent only holds because every public entry point calls
// Logger.event directly, exactly one hop. TestCallerReportsCallSite enforces
// that invariant across every entry point; changing the call chains without
// running it will silently report reggol's own internals as the call site.
// capturePC occupies a frame of its own, and it is counted here: frame 1 is
// capturePC itself, not its caller.
const (
	// callerSkipEvent: Callers → capturePC → Logger.event → entry point → user.
	callerSkipEvent = 4
	// callerSkipDirect: Callers → capturePC → Event.Caller → user.
	callerSkipDirect = 3
)

// capturePC records the program counter of the call site, or 0.
//
// The program counter is stored rather than the resolved position: capturing is
// the expensive half, and a caller that is filtered out later should not pay for
// formatting it.
func capturePC(skip int) uintptr {
	var pcs [1]uintptr

	if runtime.Callers(skip, pcs[:]) == 0 {
		return 0
	}

	return pcs[0]
}

// resolvePC turns a program counter into a source position.
//
// runtime.FuncForPC is used rather than runtime.CallersFrames, which the runtime
// documentation points you towards: FuncForPC costs no allocation where
// CallersFrames costs two, and keeping this path allocation-free is the whole
// point of the library.
//
// The subtraction is what makes the shortcut correct. runtime.Callers records
// return addresses — the instruction *after* the call — and resolving one
// directly can land in a completely different function: an event started
// through Logger.Err resolved to Event.Msg before this was fixed.
// CallersFrames performs the same adjustment internally.
//
// The equivalence is guarded rather than assumed: TestCallerMatchesRuntimeFrames
// compares this against CallersFrames for every entry point.
func resolvePC(pc uintptr) (file string, line int, ok bool) {
	if pc == 0 {
		return "", 0, false
	}

	fn := runtime.FuncForPC(pc - 1)
	if fn == nil {
		return "", 0, false
	}

	file, line = fn.FileLine(pc - 1)

	return file, line, file != ""
}

// funcForPC returns the fully qualified function name for a program counter.
func funcForPC(pc uintptr) string {
	if pc == 0 {
		return ""
	}

	fn := runtime.FuncForPC(pc - 1)
	if fn == nil {
		return ""
	}

	return fn.Name()
}

// shortCallerPath keeps the last two segments of a source path.
//
// `api/handler.go:42` tells you which package a file belongs to, which the bare
// base name does not — handler.go exists in a dozen places in any real project —
// while staying short and not leaking the build machine's directory layout into
// production logs.
//
// The result is a slice of the input, so this allocates nothing.
func shortCallerPath(file string) string {
	i := lastSeparator(file)
	if i < 0 {
		return file
	}

	j := lastSeparator(file[:i])
	if j < 0 {
		return file
	}

	return file[j+1:]
}

// lastSeparator reports the last path separator, accepting both conventions:
// the compiler records forward slashes, but a path may arrive from elsewhere.
func lastSeparator(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' || s[i] == '\\' {
			return i
		}
	}

	return -1
}

// appendCallerPosition appends `pkg/file.go:line` for a program counter.
//
// Nothing is appended when the counter cannot be resolved, so a caller-enabled
// logger degrades to its normal output rather than printing a placeholder.
func appendCallerPosition(dst []byte, pc uintptr) []byte {
	file, line, ok := resolvePC(pc)
	if !ok {
		return dst
	}

	dst = append(dst, shortCallerPath(file)...)
	dst = append(dst, ':')

	return strconv.AppendInt(dst, int64(line), base10)
}
