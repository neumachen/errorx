package errorx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"runtime"
	"sync"
)

// DefaultMaxStackDepth is the default cap on captured program counters per
// error. Negative or zero values are clamped to this default at capture time.
const DefaultMaxStackDepth = 50

// MaxStackDepth caps the number of program counters captured per error.
//
// Deprecated: this is a process-wide mutable global with no synchronization.
// New code should rely on the default. The variable is retained only for
// source compatibility; invalid values (<= 0) are clamped to
// DefaultMaxStackDepth at capture time.
var MaxStackDepth = DefaultMaxStackDepth

// Error is the historical exported interface of this package.
//
// Deprecated: new code should accept the standard error interface and use
// type assertions to *TraceError for enriched access. This interface is
// retained for source compatibility.
type Error interface {
	error
	Cause() error
	StackFrames() []StackFrame
	Stack() []uintptr
	Prefix() string
	Type() string
	RuntimeStack() []byte
	Metadata() *json.RawMessage
	SetMetadata(*json.RawMessage) error
	UnmarshalMetadata(target any) error
}

// ErrorSetter is the historical mutation interface.
//
// Deprecated: prefer calling SetMetadata directly on *TraceError. This
// interface is retained for source compatibility and is reduced to metadata
// mutation only; the prefix is no longer mutable after construction.
type ErrorSetter interface {
	SetMetadata(*json.RawMessage) error
}

// Record is the stable structured representation of a TraceError used by
// MarshalJSON and LogValue. Field names are part of the package's public
// surface; new fields may be added but existing ones will not silently change
// meaning.
type Record struct {
	// Message is the full contextual error message, identical to Error().
	Message string `json:"message,omitempty"`
	// Cause is the deepest non-TraceError cause message.
	Cause string `json:"cause,omitempty"`
	// Type is a diagnostic Go type string (e.g. "*errors.errorString",
	// "panic"). It is not stable enough for domain control flow.
	Type string `json:"type,omitempty"`
	// Prefix is this wrapper's own prefix; it does not include prefixes
	// contributed by wrapped TraceErrors.
	Prefix string `json:"prefix,omitempty"`
	// StackFrames contains resolved frame data with no source-code lines.
	StackFrames []StackFrame `json:"stack_frames,omitempty"`
	// Stack contains the raw captured program counters.
	Stack []uintptr `json:"stack,omitempty"`
	// Metadata is caller-supplied raw JSON.
	Metadata *json.RawMessage `json:"metadata,omitempty"`
}

// TraceError is an enriched error value with a captured stack trace, an
// optional contextual prefix, optional structured metadata, and standard
// errors.Unwrap support.
//
// A *TraceError is immutable apart from its metadata slot. All methods are
// safe for concurrent use; reads of mutable state take a read lock, and
// SetMetadata takes a write lock. Wrapping an existing *TraceError produces a
// new *TraceError without mutating the wrapped value.
//
// The zero value is not usable; obtain a *TraceError via NewError, Wrap,
// WrapPrefix, NewErrorf, Errorf, ParsePanic, or FromPanic.
type TraceError struct {
	cause  error
	prefix string

	// stack holds program counters captured at construction. The slice is
	// never mutated after the struct is returned to the caller.
	stack []uintptr

	framesOnce sync.Once
	frames     []StackFrame

	mu       sync.RWMutex
	metadata *json.RawMessage

	// debugStack is the raw runtime.Stack() / debug.Stack() bytes for
	// errors built from a pre-formatted panic stack. It is non-nil only
	// when stack capture was not available (i.e. ParsePanic, FromPanic
	// without runtime.Callers).
	debugStack []byte
	// parsedFrames is the pre-parsed frame list used by ParsePanic; it
	// substitutes for the lazy resolution path when set.
	parsedFrames []StackFrame
}

// captureStack records up to MaxStackDepth program counters, skipping the
// given number of frames. The returned slice is owned by the caller.
func captureStack(skip int) []uintptr {
	depth := MaxStackDepth
	if depth <= 0 {
		depth = DefaultMaxStackDepth
	}
	pcs := make([]uintptr, depth)
	n := runtime.Callers(skip+1, pcs)
	out := make([]uintptr, n)
	copy(out, pcs[:n])
	return out
}

// newTraceError builds a *TraceError around the given cause with a fresh
// stack capture. The caller is responsible for nil-checking cause.
func newTraceError(cause error, skip int) *TraceError {
	return &TraceError{
		cause: cause,
		stack: captureStack(skip + 1),
	}
}

// NewError returns a *TraceError wrapping cause. It returns nil if cause is
// nil, following the Go convention that nil means no error.
func NewError(cause error) Error {
	if cause == nil {
		return nil
	}
	return newTraceError(cause, 1)
}

// NewErrorf creates a *TraceError from a formatted message. The message is
// produced via fmt.Errorf, so %w directives in format participate in
// errors.Is / errors.As walks through the wrapper.
//
// Deprecated: prefer Errorf. NewErrorf is retained for source compatibility.
func NewErrorf(format string, a ...any) Error {
	return newTraceError(fmt.Errorf(format, a...), 1)
}

// Errorf creates a *TraceError from a formatted message. It is the canonical
// alias for the historical NewErrorf and is the form the README recommends.
// %w directives participate in errors.Is / errors.As.
func Errorf(format string, a ...any) Error {
	return newTraceError(fmt.Errorf(format, a...), 1)
}

// Wrap returns a *TraceError around err with a fresh stack capture at the
// call site. It returns nil if err is nil. Unlike historical behavior, Wrap
// never returns the input pointer aliased; wrapping an existing *TraceError
// produces a new wrapper that Unwraps to it.
//
// stackToSkip is added to the number of frames hidden from the captured
// stack; 0 starts the stack at the caller of Wrap.
func Wrap(err error, stackToSkip int) Error {
	if err == nil {
		return nil
	}
	return newTraceError(err, stackToSkip+1)
}

// WrapPrefix returns a new *TraceError that wraps err and prepends prefix to
// the contextual error message. The wrapped error is not mutated. Calling
// Error() on the result walks the wrapper chain, so wrapping a prefixed
// TraceError yields "outer: inner: base".
//
// WrapPrefix returns nil if err is nil.
func WrapPrefix(err error, prefix string, skip int) Error {
	if err == nil {
		return nil
	}
	te := newTraceError(err, skip+1)
	te.prefix = prefix
	return te
}

// Is reports whether any error in err's chain matches target. It is a thin
// wrapper around the standard library's errors.Is.
//
// Deprecated: call errors.Is directly. This function is retained for source
// compatibility.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// FromPanic constructs a *TraceError from a value recovered via recover().
//
// The cause is fmt.Sprint(value) reported with Type "panic". If stack is
// non-nil it is preserved as the error's runtime stack output; otherwise a
// fresh runtime.Callers capture is taken at the call site. The expected
// usage is:
//
//	defer func() {
//	    if r := recover(); r != nil {
//	        err = errorx.FromPanic(r, debug.Stack())
//	    }
//	}()
func FromPanic(value any, stack []byte) *TraceError {
	te := &TraceError{
		cause:      uncaughtPanic{message: fmt.Sprint(value)},
		debugStack: append([]byte(nil), stack...),
	}
	if len(stack) == 0 {
		te.debugStack = nil
		te.stack = captureStack(1)
	}
	return te
}

// Error returns the contextual error message. When the error has a prefix,
// the prefix is prepended with a colon separator. Wrapped errors contribute
// their own prefixes via the Unwrap chain.
func (e *TraceError) Error() string {
	if e == nil {
		return ""
	}
	if e.cause == nil {
		if e.prefix != "" {
			return e.prefix
		}
		return ""
	}
	if e.prefix == "" {
		return e.cause.Error()
	}
	return e.prefix + ": " + e.cause.Error()
}

// Unwrap returns the immediate wrapped cause for use with errors.Is and
// errors.As.
func (e *TraceError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Cause returns the deepest non-TraceError cause in the wrapper chain. It is
// retained for source compatibility with callers that want the "original"
// error; new code should use errors.Is / errors.As / errors.Unwrap.
func (e *TraceError) Cause() error {
	if e == nil {
		return nil
	}
	cur := e.cause
	for {
		next, ok := cur.(*TraceError)
		if !ok || next == nil {
			return cur
		}
		cur = next.cause
	}
}

// Prefix returns this wrapper's own prefix string. It does not include
// prefixes contributed by wrapped TraceErrors.
func (e *TraceError) Prefix() string {
	if e == nil {
		return ""
	}
	return e.prefix
}

// Type returns a Go type string describing the underlying cause. For errors
// produced by ParsePanic or FromPanic it returns "panic". The empty string is
// returned when no cause is present. The result is diagnostic only and is
// not stable enough for domain control flow.
func (e *TraceError) Type() string {
	if e == nil || e.cause == nil {
		return ""
	}
	if _, ok := e.cause.(uncaughtPanic); ok {
		return "panic"
	}
	return reflect.TypeOf(e.cause).String()
}

// Stack returns a copy of the captured program counters. Callers may
// freely mutate the returned slice.
func (e *TraceError) Stack() []uintptr {
	if e == nil {
		return nil
	}
	out := make([]uintptr, len(e.stack))
	copy(out, e.stack)
	return out
}

// StackFrames returns a copy of the resolved stack frame data. Frames are
// resolved lazily on first call. Subsequent calls reuse the cached frames
// and return a fresh copy each time.
func (e *TraceError) StackFrames() []StackFrame {
	if e == nil {
		return nil
	}
	e.framesOnce.Do(func() {
		switch {
		case e.parsedFrames != nil:
			e.frames = e.parsedFrames
		case len(e.stack) > 0:
			frames := make([]StackFrame, 0, len(e.stack))
			it := runtime.CallersFrames(e.stack)
			i := 0
			for {
				rf, more := it.Next()
				pc := uintptr(0)
				if i < len(e.stack) {
					pc = e.stack[i]
				}
				pkg, name := splitPackageAndName(rf.Function)
				frames = append(frames, StackFrame{
					File:           rf.File,
					LineNumber:     rf.Line,
					Name:           name,
					Package:        pkg,
					ProgramCounter: pc,
				})
				i++
				if !more {
					break
				}
			}
			e.frames = frames
		}
	})
	out := make([]StackFrame, len(e.frames))
	copy(out, e.frames)
	return out
}

// RuntimeStack returns a formatted byte slice describing the captured stack.
// The format is the package's own representation; it does not read source
// files from disk and is not guaranteed to match runtime/debug.Stack().
//
// For TraceErrors built from a pre-formatted panic stack (FromPanic with a
// non-nil stack argument), RuntimeStack returns those raw bytes.
func (e *TraceError) RuntimeStack() []byte {
	if e == nil {
		return nil
	}
	if len(e.debugStack) > 0 {
		out := make([]byte, len(e.debugStack))
		copy(out, e.debugStack)
		return out
	}
	frames := e.StackFrames()
	var buf bytes.Buffer
	for _, f := range frames {
		buf.WriteString(f.String())
	}
	return buf.Bytes()
}

// Metadata returns a deep copy of the caller-supplied metadata, or nil if
// none was set.
func (e *TraceError) Metadata() *json.RawMessage {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.metadata == nil {
		return nil
	}
	clone := make(json.RawMessage, len(*e.metadata))
	copy(clone, *e.metadata)
	return &clone
}

// SetMetadata stores metadata on the error. Passing nil clears any previously
// stored metadata. Non-nil metadata is validated with json.Valid and cloned;
// invalid JSON is reported immediately and the previous value is left
// unchanged. SetMetadata is safe for concurrent use.
func (e *TraceError) SetMetadata(metadata *json.RawMessage) error {
	if e == nil {
		return errors.New("errorx: SetMetadata on nil *TraceError")
	}
	if metadata == nil {
		e.mu.Lock()
		e.metadata = nil
		e.mu.Unlock()
		return nil
	}
	if !json.Valid(*metadata) {
		return errors.New("errorx: invalid metadata: not valid JSON")
	}
	clone := make(json.RawMessage, len(*metadata))
	copy(clone, *metadata)
	e.mu.Lock()
	e.metadata = &clone
	e.mu.Unlock()
	return nil
}

// UnmarshalMetadata decodes the stored metadata into target. It returns nil
// without touching target when no metadata is present.
func (e *TraceError) UnmarshalMetadata(target any) error {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	md := e.metadata
	e.mu.RUnlock()
	if md == nil {
		return nil
	}
	return json.Unmarshal(*md, target)
}

// Record returns a snapshot of the error suitable for structured output.
// The returned StackFrames and Stack slices are copies owned by the caller.
func (e *TraceError) Record() Record {
	if e == nil {
		return Record{}
	}
	var causeMsg string
	if c := e.Cause(); c != nil {
		causeMsg = c.Error()
	}
	return Record{
		Message:     e.Error(),
		Cause:       causeMsg,
		Type:        e.Type(),
		Prefix:      e.Prefix(),
		StackFrames: e.StackFrames(),
		Stack:       e.Stack(),
		Metadata:    e.Metadata(),
	}
}

// MarshalJSON encodes the error as a Record.
func (e *TraceError) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}
	return json.Marshal(e.Record())
}

// LogValue returns a slog.Value with the structured fields from Record. The
// raw stack PCs are omitted from the slog output to keep log lines compact;
// they remain available via JSON marshaling and the Stack method.
func (e *TraceError) LogValue() slog.Value {
	if e == nil {
		return slog.Value{}
	}
	r := e.Record()
	attrs := make([]slog.Attr, 0, 6)
	if r.Message != "" {
		attrs = append(attrs, slog.String("message", r.Message))
	}
	if r.Cause != "" && r.Cause != r.Message {
		attrs = append(attrs, slog.String("cause", r.Cause))
	}
	if r.Type != "" {
		attrs = append(attrs, slog.String("type", r.Type))
	}
	if r.Prefix != "" {
		attrs = append(attrs, slog.String("prefix", r.Prefix))
	}
	if len(r.StackFrames) > 0 {
		attrs = append(attrs, slog.Any("stack_frames", r.StackFrames))
	}
	if r.Metadata != nil {
		attrs = append(attrs, slog.Any("metadata", r.Metadata))
	}
	return slog.GroupValue(attrs...)
}

// Format implements fmt.Formatter.
//
//	%s, %v  → Error()
//	%q      → quoted Error()
//	%+v     → Error() followed by RuntimeStack()
func (e *TraceError) Format(s fmt.State, verb rune) {
	if e == nil {
		_, _ = io.WriteString(s, "<nil>")
		return
	}
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = io.WriteString(s, e.Error())
			_, _ = s.Write([]byte{'\n'})
			_, _ = s.Write(e.RuntimeStack())
			return
		}
		_, _ = io.WriteString(s, e.Error())
	case 's':
		_, _ = io.WriteString(s, e.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.Error())
	default:
		_, _ = fmt.Fprintf(s, "%%!%c(errorx.TraceError=%s)", verb, e.Error())
	}
}
