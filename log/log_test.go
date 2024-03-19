package log

import (
	"github.com/efureev/reggol"
)

func setup() {
	trans := reggol.NewTextTransformer(``)
	trans.HideTimestamp()

	// Logger = reggol.New(os.Stdout)
	Logger = reggol.New(reggol.NewConsoleWriter().WithTransformer(trans))
}

func ExamplePrint() {
	setup()

	Print("hello world")
	// Output: level=debug, message=hello world
}

// func ExampleConsoleErr() {
//	trans := reggol.NewConsoleTransformer(false, ``)
//	trans.HideTimestamp()
//
//	Logger = reggol.New(reggol.NewConsoleWriter(func(w *reggol.ConsoleWriter) { w.Trans = trans }))
//	err := errors.New("some error")
//
//	Err(err).BlockText(`block`).Msg("hello world")
// }

// func ExampleCreateInstance() {
//	trans := reggol.NewTextTransformer(``)
//	trans.HideTimestamp()
//
//	logger := reggol.New(reggol.NewConsoleWriter().WithTransformer(trans))
//	_ = logger
// }
