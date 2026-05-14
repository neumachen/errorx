/*
Package errorx is a small, dependency-free extension of Go's standard
errors package. It adds stack-trace capture, contextual prefix wrapping,
structured JSON / log/slog output, optional caller-supplied metadata, and
panic-recovery helpers, while preserving standard errors.Is / errors.As /
errors.Unwrap semantics.

# Quick start

	err := errorx.Errorf("failed to process %s: %w", item, cause)

	if errors.Is(err, cause) {
	    // standard errors.Is sees through errorx wrappers
	}

	// Add context without mutating the wrapped error:
	wrapped := errorx.WrapPrefix(err, "validation failed", 0)

	// Attach structured metadata (validated as JSON at set time):
	md := json.RawMessage(`{"request_id":"abc-123"}`)
	wrapped.(*errorx.TraceError).SetMetadata(&md)

# Core type

The package's canonical type is *TraceError. All constructors return values
backed by *TraceError; the historical Error interface is retained as a
deprecated alias for source compatibility.

A *TraceError carries:

  - the wrapped cause (visible via Unwrap and Cause),
  - a captured runtime stack (Stack, StackFrames),
  - an optional prefix (Prefix),
  - optional caller-supplied JSON metadata (Metadata, SetMetadata).

It implements error, fmt.Formatter, json.Marshaler, and slog.LogValuer.

# Behavior changes from earlier versions

  - Nil in, nil out. NewError(nil), Wrap(nil, ...), and WrapPrefix(nil, ...,
    ...) all return nil.
  - Wrapping never mutates the wrapped error. Each Wrap / WrapPrefix call
    produces a new *TraceError with a fresh stack capture.
  - StackFrames(), Stack(), and Metadata() return copies; callers may freely
    mutate the returned slices.
  - Default stack formatting no longer reads source files from disk. The
    opt-in StackFrame.SourceLine helper remains for explicit callers.
  - errorx.Is is a thin wrapper around errors.Is.

# Structured output

JSON marshaling produces a Record value:

	{
	    "message":      "ctx: root cause",   // == Error()
	    "cause":        "root cause",        // deepest non-TraceError cause
	    "type":         "*errors.errorString",
	    "prefix":       "ctx",
	    "stack_frames": [...],
	    "stack":        [...],
	    "metadata":     {...}
	}

slog.LogValuer emits the same fields as a group attribute, omitting the raw
stack PCs to keep log lines compact.

# Concurrency

All methods on *TraceError are safe for concurrent use. The wrapped cause,
prefix, and captured PCs are immutable after construction; metadata access
is guarded by an internal RWMutex.

# Security note

Stack frames may include absolute file paths and function names, and
metadata may contain caller-controlled data. Do not expose the full
structured representation to untrusted clients without first considering
whether the contents are safe to disclose.
*/
package errorx
