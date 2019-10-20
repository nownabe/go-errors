package errors // import "go.nownabe.dev/errors"

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"

	"golang.org/x/xerrors"
	"go.nownabe.dev/log"
)

const (
	// KindBadRequest is a kind.
	KindBadRequest = http.StatusBadRequest
	// KindUnauthorized is a kind.
	KindUnauthorized = http.StatusUnauthorized
	// KindForbidden is a kind.
	KindForbidden = http.StatusForbidden
	// KindNotFound is a kind.
	KindNotFound = http.StatusNotFound
	// KindUnexpected is a kind.
	KindUnexpected = http.StatusInternalServerError
)

// Op describes packages and functions.
type Op string

type appError struct {
	err    error
	msg    string
	op     Op
	kind   int
	level  log.Level
	frames [3]uintptr
}

// E constructs an error.
func E(op Op, args ...interface{}) error {
	e := &appError{op: op}
	runtime.Callers(1, e.frames[:])

	for _, a := range args {
		switch a := a.(type) {
		case error:
			e.err = a
		case string:
			e.msg = a
		case log.Level:
			e.level = a
		case int:
			e.kind = a
		}
	}

	return e
}

// Ops aggregates the error's operations
// with embedded errors.
func Ops(err error) []string {
	ops := []string{}
	for {
		e, ok := err.(*appError)
		if !ok {
			break
		}
		ops = append(ops, string(e.op))
		err = e.err
	}
	return ops
}

// Kind returns error's kind.
func Kind(err error) int {
	e, ok := err.(*appError)
	if !ok {
		return KindUnexpected
	}
	if e.kind != 0 {
		return e.kind
	}

	return Kind(e.err)
}

// KindText returns a friendly string of
// the Kind type.
func KindText(err error) string {
	return http.StatusText(Kind(err))
}

// Level returns error's level.
func Level(err error) log.Level {
	e, ok := err.(*appError)
	if !ok {
		return log.LevelError
	}

	if e.level != 0 {
		return e.level
	}

	return Level(e.err)
}

// Is checks error's kind.
func Is(err error, kind int) bool {
	if err == nil {
		return false
	}
	return Kind(err) == kind
}

// Msg returns error message for clients.
func Msg(err error) string {
	e, ok := err.(*appError)
	if !ok {
		return err.Error()
	}

	var msg string
	first := true
	for {
		if e.msg != "" {
			if first {
				msg = e.msg
				first = false
			} else {
				msg = msg + ": " + e.msg
			}
		}
		e, ok = e.err.(*appError)
		if !ok {
			break
		}
	}

	if msg == "" {
		msg = http.StatusText(Kind(err))
	}

	return msg
}

func (err *appError) location() (function, file string, line int) {
	frames := runtime.CallersFrames(err.frames[:])
	if _, ok := frames.Next(); !ok {
		return "", "", 0
	}
	fr, ok := frames.Next()
	if !ok {
		return "", "", 0
	}

	return fr.Function, fr.File, fr.Line
}

// Stacktrace returns an array of stacktrace tupples
// that inclues function, file and line.
func Stacktrace(err error) [][3]string {
	frames := [][3]string{}
	for {
		e, ok := err.(*appError)
		if !ok {
			break
		}
		function, file, line := e.location()
		if function != "" && file != "" {
			frames = append(frames, [3]string{function, file, strconv.Itoa(line)})
		}
		err = e.err
	}
	return frames

}

// Error returns the core error message.
func (err *appError) Error() string {
	return err.err.Error()
}

// Unwrap returns a wrapped error.
func (err *appError) Unwrap() error {
	return err.err
}

// Format .
func (err *appError) Format(s fmt.State, v rune) { xerrors.FormatError(err, s, v) }

// FormatError .
func (err *appError) FormatError(p xerrors.Printer) (next error) {
	if err.msg == "" {
		p.Print("(no message)")
	} else {
		p.Print(err.msg)
	}
	if p.Detail() {
		function, file, line := err.location()
		if function != "" {
			p.Printf("%s\n    ", function)
		}
		if file != "" {
			p.Printf("%s:%d\n", file, line)
		}
	}
	return err.err
}
