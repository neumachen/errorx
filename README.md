# errorx

[![test](https://github.com/neumachen/errorx/actions/workflows/test.yml/badge.svg)](https://github.com/neumachen/errorx/actions/workflows/test.yml)
[![lint](https://github.com/neumachen/errorx/actions/workflows/lint.yml/badge.svg)](https://github.com/neumachen/errorx/actions/workflows/lint.yml)

A small, standard-library-only Go error package that extends `errors` with
stack-trace capture, contextual prefix wrapping, structured JSON / `log/slog`
output, optional caller-supplied metadata, and panic-recovery helpers. It
preserves `errors.Is` / `errors.As` / `errors.Unwrap` through every wrapper.

## Install

```bash
go get github.com/neumachen/errorx
```

Minimum Go version: 1.24.

## Why

`errorx` aims to be the smallest useful extension to `errors`:

- enriched error values without a logging-framework dependency,
- standard error semantics (`Unwrap`, `Is`, `As`, `fmt.Errorf %w`),
- safe concurrent reads, immutable wrappers, no surprise filesystem reads.

It is deliberately *not* a logging framework, metrics framework, tracing
SDK, or domain error taxonomy.

## Quick start

```go
import (
    "encoding/json"
    "errors"
    "fmt"

    "github.com/neumachen/errorx"
)

var ErrNotFound = errors.New("not found")

func lookup(id string) error {
    if id == "" {
        return errorx.Errorf("lookup %q: %w", id, ErrNotFound)
    }
    // ...
    return nil
}

func main() {
    err := lookup("")

    fmt.Println(errors.Is(err, ErrNotFound)) // true

    // Add context without mutating the wrapped error.
    wrapped := errorx.WrapPrefix(err, "user lookup", 0).(*errorx.TraceError)

    // Attach JSON metadata. Validation happens at set time.
    md := json.RawMessage(`{"request_id":"abc-123"}`)
    if e := wrapped.SetMetadata(&md); e != nil {
        // invalid JSON, surfaced immediately
    }

    fmt.Printf("%v\n", wrapped)   // user lookup: lookup "": not found
    fmt.Printf("%+v\n", wrapped)  // ...then a runtime stack section
}
```

## Behavior

- **Nil in, nil out.** `NewError(nil)`, `Wrap(nil, ...)`, and
  `WrapPrefix(nil, ..., ...)` all return `nil`. No more accidental
  conversion of a success path into a failure path.
- **Wrapping never mutates the wrapped error.** Each `Wrap` / `WrapPrefix`
  captures a fresh stack and returns a new `*TraceError`. The wrapped value
  is reachable via `Unwrap` and the deepest non-`errorx` cause is available
  via `Cause`.
- **Standard `errors` integration.** `errors.Is`, `errors.As`,
  `errors.Unwrap`, and `fmt.Errorf("…: %w", err)` all work through the
  wrappers. `errorx.Is` is a thin wrapper around `errors.Is`.
- **Immutable from the caller's perspective.** `Stack()`, `StackFrames()`,
  and `Metadata()` return copies; mutating them does not affect the error.
- **Concurrency safe.** All read methods are safe under concurrent use.
  `SetMetadata` is safe to call concurrently with reads; it validates with
  `json.Valid` and stores a clone of the bytes.
- **No filesystem reads in default formatting.** `StackFrame.String()` and
  `RuntimeStack()` never open source files. The opt-in
  `StackFrame.SourceLine` helper remains for explicit callers.

## API surface

Canonical type:

```go
type TraceError struct { /* unexported */ }

func (e *TraceError) Error() string
func (e *TraceError) Unwrap() error
func (e *TraceError) Cause() error
func (e *TraceError) Prefix() string
func (e *TraceError) Type() string
func (e *TraceError) Stack() []uintptr
func (e *TraceError) StackFrames() []StackFrame
func (e *TraceError) RuntimeStack() []byte
func (e *TraceError) Metadata() *json.RawMessage
func (e *TraceError) SetMetadata(*json.RawMessage) error
func (e *TraceError) UnmarshalMetadata(target any) error
func (e *TraceError) Record() Record
func (e *TraceError) MarshalJSON() ([]byte, error)
func (e *TraceError) LogValue() slog.Value
func (e *TraceError) Format(s fmt.State, verb rune)
```

Constructors (return `Error` interface for source compatibility):

```go
func NewError(cause error) Error
func Errorf(format string, a ...any) Error
func Wrap(err error, stackToSkip int) Error
func WrapPrefix(err error, prefix string, skip int) Error
func Is(err, target error) bool                       // == errors.Is
func ParsePanic(s string) (Error, error)
func FromPanic(value any, stack []byte) *TraceError
```

Deprecated but retained:

```go
type Error interface { /* ... */ }      // Deprecated: prefer *TraceError
type ErrorSetter interface { /* ... */ }// Deprecated: metadata-only
func NewErrorf(format string, a ...any) Error // Deprecated: alias of Errorf
var  MaxStackDepth int                  // Deprecated: prefer DefaultMaxStackDepth
```

Constants:

```go
const DefaultMaxStackDepth = 50
```

## Structured output

`*TraceError` marshals to a stable `Record`:

```json
{
  "message":      "user lookup: lookup \"\": not found",
  "cause":        "not found",
  "type":         "*errors.errorString",
  "prefix":       "user lookup",
  "stack_frames": [
    {
      "file": "/path/to/file.go",
      "line_number": 42,
      "name": "FunctionName",
      "package": "github.com/example/pkg",
      "program_counter": 1234567
    }
  ],
  "stack":   [1234567, 2345678],
  "metadata": {"request_id": "abc-123"}
}
```

`message` is the full contextual error (identical to `Error()`); `cause` is
the deepest non-`*TraceError` cause. `type` is diagnostic only — it is not
stable enough to drive control flow; use sentinel errors with `errors.Is`
or typed errors with `errors.As` instead.

`*TraceError` also implements `slog.LogValuer`, producing the same fields
as a slog group attribute (the raw stack PCs are omitted to keep log lines
compact).

## Recovering from panics

```go
import "runtime/debug"

func doWork() (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = errorx.FromPanic(r, debug.Stack())
        }
    }()
    // ...
    return nil
}
```

`FromPanic` returns a `*TraceError` whose `Type()` is `"panic"` and whose
`RuntimeStack()` is the supplied `debug.Stack()` bytes (when non-nil), or a
freshly captured stack (when `nil`).

`ParsePanic` parses pre-formatted panic strings; it remains useful for
post-mortem analysis of crash logs. Use `FromPanic` for in-process recovery.

## Security note

Stack frames may include absolute file paths and function names, and
metadata may contain caller-controlled data. Do not expose the full
structured representation to untrusted clients without first considering
whether the contents are safe to disclose.

## Testing

```bash
go test ./...
go test -race ./...
go test -run='^$' -fuzz=FuzzParsePanic -fuzztime=30s
go test -bench=. -benchmem -run='^$' ./...
```

## License

See [LICENSE.md](LICENSE.md).
