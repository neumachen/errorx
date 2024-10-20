// Package errorx provides errors that have stack-traces.
//
// This is particularly useful when you want to understand the
// state of execution when an error was returned unexpectedly.
//
// It provides the type *Error which implements the standard
// golang error interface, so you can use this library interchangably
// with code that is expecting a normal error return.
package errorx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
)

// MaxStackDepth is the maximum number of stackframes on any error.
var MaxStackDepth = 50

// Error is a more feature rich implementation of error interface inspired by
// PostgreSQL error style guide.  It can be used wherever the builtin error
// interface is expected.
type Error struct {
	Cause       error `json:"cause"`
	stackFrames []StackFrame
	prefix      string
	stack       []uintptr
}

// MarshalJSON ...
func (e *Error) MarshalJSON() ([]byte, error) {
	type Alias Error

	return json.Marshal(&struct {
		StackFrames []StackFrame `json:"stack_frames,omitempty"`
		Stack       []uintptr    `json:"stack,omitempty"`
		Prefix      string       `json:"prefix,omitempty"`
		Cause       string       `json:"cause,omitempty"`
		*Alias
	}{
		Alias:       (*Alias)(e),
		StackFrames: e.StackFrames(),
		Stack:       e.stack,
		Prefix:      e.prefix,
		Cause:       e.Cause.Error(),
	})
}

// New makes an Error from the given value. If that value is already an
// error then it will be used directly, if not, it will be passed to
// fmt.Errorf("%v"). The stacktrace will point to the line of code that
// called New.
func New(e any) *Error {
	var err error

	switch e := e.(type) {
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}

	stack := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2, stack[:])
	return &Error{
		Cause: err,
		stack: stack[:length],
	}
}

// Wrap makes an Error from the given value. If that value is already an
// error then it will be used directly, if not, it will be passed to
// fmt.Errorf("%v"). The skip parameter indicates how far up the stack
// to start the stacktrace. 0 is from the current call, 1 from its caller, etc.
func Wrap(e any, skip int) *Error {
	var err error

	switch e := e.(type) {
	case *Error:
		return e
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}

	stack := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2+skip, stack[:])
	return &Error{
		Cause: err,
		stack: stack[:length],
	}
}

// WrapPrefix makes an Error from the given value. If that value is already an
// error then it will be used directly, if not, it will be passed to
// fmt.Errorf("%v"). The prefix parameter is used to add a prefix to the
// error message when calling Error(). The skip parameter indicates how far
// up the stack to start the stacktrace. 0 is from the current call,
// 1 from its caller, etc.
func WrapPrefix(e any, prefix string, skip int) *Error {
	err := Wrap(e, skip)

	if err.prefix != "" {
		err.prefix = fmt.Sprintf("%s: %s", prefix, err.prefix)
	} else {
		err.prefix = prefix
	}

	return err
}

// Is detects whether the error is equal to a given error. Errors
// are considered equal by this function if they are the same object,
// or if they both contain the same error inside an errors.Error.
func Is(e error, original error) bool {
	if e == original {
		return true
	}

	if e, ok := e.(*Error); ok {
		return Is(e.Cause, original)
	}

	if original, ok := original.(*Error); ok {
		return Is(e, original.Cause)
	}

	return false
}

// Errorf creates a new error with the given message. You can use it
// as a drop-in replacement for fmt.Errorf() to provide descriptive
// errors in return values.
func Errorf(format string, a ...any) *Error {
	return Wrap(fmt.Errorf(format, a...), 2)
}

// Error returns the underlying error's message.
func (e *Error) Error() string {
	msg := e.Cause.Error()
	if e.prefix != "" {
		msg = fmt.Sprintf("%s: %s", e.prefix, msg)
	}

	return msg
}

// Stack returns the callstack formatted the same way that go does
// in runtime/debug.Stack()
func (e *Error) Stack() []byte {
	var buf bytes.Buffer
	for _, frame := range e.StackFrames() {
		buf.WriteString(frame.String())
	}

	return buf.Bytes()
}

// StackFrames returns an array of frames containing information about the
// stack.
func (e *Error) StackFrames() []StackFrame {
	if e.stackFrames == nil {
		e.stackFrames = make([]StackFrame, len(e.stack))

		for i, pc := range e.stack {
			e.stackFrames[i] = NewStackFrame(pc)
		}
	}

	return e.stackFrames
}

// ErrorStack returns a string that contains both the
// error message and the callstack.
func (e *Error) ErrorStack() string {
	return e.TypeName() + " " + e.Error() + "\n" + string(e.Stack())
}

// TypeName returns the type this error. e.g. *errors.stringError.
func (e *Error) TypeName() string {
	if _, ok := e.Cause.(uncaughtPanic); ok {
		return "panic"
	}
	return reflect.TypeOf(e.Cause).String()
}
