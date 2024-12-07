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

type errorSetter interface {
	setPrefix(string)
}

type Error interface {
	errorSetter
	Cause() error
	Error() string
	StackFrames() []StackFrame
	Stack() []uintptr
	Prefix() string
	Type() string
	RuntimeStack() []byte
}

type errorJSONObject struct {
	Cause       string       `json:"cause,omitempty"`
	StackFrames []StackFrame `json:"stack_frames,omitempty"`
	Stack       []uintptr    `json:"stack,omitempty"`
	Prefix      string       `json:"prefix,omitempty"`
}

// errorData is a more feature rich implementation of error interface inspired by
// PostgreSQL error style guide. It can be used wherever the builtin error
// interface is expected.
type errorData struct {
	cause       error
	stackFrames []StackFrame
	prefix      string
	stack       []uintptr
}

func (e errorData) jsonObject() errorJSONObject {
	return errorJSONObject{
		Cause:       e.Error(),
		StackFrames: e.StackFrames(),
		Stack:       e.Stack(),
		Prefix:      e.Prefix(),
	}
}

// MarshalJSON ...
func (e errorData) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.jsonObject())
}

func (e *errorData) Cause() error {
	return e.cause
}

func (e errorData) Prefix() string {
	return e.prefix
}

func (e errorData) Stack() []uintptr {
	return e.stack
}

// Error returns the underlying error's message.
func (e errorData) Error() string {
	if e.cause == nil {
		return ""
	}

	msg := e.cause.Error()
	if e.Prefix() != "" {
		msg = fmt.Sprintf("%s: %s", e.Prefix(), msg)
	}

	return msg
}

func (e *errorData) setPrefix(prefix string) {
	e.prefix = prefix
}

// Stack returns the callstack formatted the same way that go does
// in runtime/debug.Stack()
func (e errorData) RuntimeStack() []byte {
	var buf bytes.Buffer
	defer buf.Reset()
	for _, frame := range e.StackFrames() {
		buf.WriteString(frame.String())
	}

	return buf.Bytes()
}

// StackFrames returns an array of frames containing information about the
// stack.
func (e errorData) StackFrames() []StackFrame {
	if e.stackFrames == nil {
		e.stackFrames = make([]StackFrame, len(e.stack))

		for i, pc := range e.stack {
			e.stackFrames[i] = NewStackFrame(pc)
		}
	}

	return e.stackFrames
}

// ErrorRuntimeStack returns a string that contains both the
// error message and the callstack.
func (e errorData) ErrorRuntimeStack() string {
	return e.Type() + " " + e.Error() + "\n" + string(e.RuntimeStack())
}

// Type returns the type this error. e.g. *errors.stringError.
func (e errorData) Type() string {
	if e.cause == nil {
		return ""
	}
	if _, ok := e.cause.(uncaughtPanic); ok {
		return fmt.Sprintf("panic")
	}
	return reflect.TypeOf(e.cause).String()
}

// New makes an Error from the given value. If that value is already an
// error then it will be used directly, if not, it will be passed to
// fmt.Errorf("%v"). The stacktrace will point to the line of code that
// called New.
func New(newError any) Error {
	var cause error

	switch e := newError.(type) {
	case error:
		cause = e
	default:
		cause = fmt.Errorf("%v", e)
	}

	stack := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2, stack[:])
	return &errorData{
		cause: cause,
		stack: stack[:length],
	}
}

// Wrap makes an Error from the given value. If that value is already an
// error then it will be used directly, if not, it will be passed to
// fmt.Errorf("%v"). The stackToSkip parameter indicates how far up the stack
// to start the stacktrace. 0 is from the current call, 1 from its caller, etc.
func Wrap(errToWrap any, stackToSkip int) Error {
	var err error

	switch e := errToWrap.(type) {
	case *errorData:
		return e
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}

	stack := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2+stackToSkip, stack[:])
	return &errorData{
		cause: err,
		stack: stack[:length],
	}
}

// WrapPrefix makes an Error from the given value. If that value is already an
// error then it will be used directly, if not, it will be passed to
// fmt.Errorf("%v"). The prefix parameter is used to add a prefix to the
// error message when calling Error(). The skip parameter indicates how far
// up the stack to start the stacktrace. 0 is from the current call,
// 1 from its caller, etc.
func WrapPrefix(e any, prefix string, skip int) Error {
	err := Wrap(e, skip)

	if err.Prefix() != "" {
		err.setPrefix(fmt.Sprintf("%s: %s", prefix, err.Prefix()))
	} else {
		err.setPrefix(prefix)
	}

	return err
}

// Is detects whether the error is equal to a given error. Errors
// are considered equal by this function if they are the same object,
// or if they both contain the same error inside an errors.Error.
func Is(comparedTo error, target error) bool {
	if comparedTo == target {
		return true
	}

	if errx, ok := comparedTo.(Error); ok {
		return Is(errx.Cause(), target)
	}

	if original, ok := target.(*errorData); ok {
		return Is(comparedTo, original.cause)
	}

	return false
}

// Errorf creates a new error with the given message. You can use it
// as a drop-in replacement for fmt.Errorf() to provide descriptive
// errors in return values.
func Errorf(format string, a ...any) Error {
	return Wrap(fmt.Errorf(format, a...), 2)
}
