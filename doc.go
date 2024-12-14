/*
Package errorx provides a rich error handling implementation with stack traces, error wrapping capabilities,
and metadata support.

Key Features:
  - Stack traces for errors
  - Error wrapping with optional prefixes
  - JSON serialization support
  - Panic parsing and recovery
  - Source line information in stack frames
  - Structured metadata attachment
  - Type-safe metadata unmarshaling

Core Types and Interfaces:

Error interface extends the standard error interface with additional capabilities:
  - Cause() error: Returns the underlying error
  - StackFrames() []StackFrame: Returns the call stack frames
  - Stack() []uintptr: Returns the raw program counters
  - Prefix() string: Returns any prefix added to the error
  - Type() string: Returns the error type
  - RuntimeStack() []byte: Returns a formatted stack trace
  - Metadata() *json.RawMessage: Returns associated metadata
  - UnmarshalMetadata(any) error: Unmarshals metadata into a target struct

StackFrame type provides detailed information about a single stack frame:
  - File: Source file path
  - LineNumber: Line number in the source file
  - Name: Function name
  - Package: Package path
  - ProgramCounter: Raw program counter value

Key Functions:

Creating Errors:

	err := errorx.New("something went wrong")
	err := errorx.Errorf("failed to process %s", item)
	err := errorx.NewError(existingError)

Wrapping Errors:

	err := errorx.Wrap(existingError, 0)
	err := errorx.WrapPrefix(existingError, "validation", 0)

Error Comparison:

	if errorx.Is(err, target) {
	    // Handle specific error
	}

Working with Metadata:

	// Attach metadata
	metadata := json.RawMessage(`{"user_id": 123, "request_id": "abc-123"}`)
	err.SetMetadata(&metadata)

	// Retrieve metadata
	type ErrorContext struct {
	    UserID    int    `json:"user_id"`
	    RequestID string `json:"request_id"`
	}
	var ctx ErrorContext
	err.UnmarshalMetadata(&ctx)

Example Usage:

	func ProcessItem(item string) error {
	    if err := validate(item); err != nil {
	        metadata := json.RawMessage(`{"item": "` + item + `"}`)
	        wrappedErr := errorx.WrapPrefix(err, "validation failed", 0)
	        wrappedErr.SetMetadata(&metadata)
	        return wrappedErr
	    }

	    result, err := process(item)
	    if err != nil {
	        return errorx.Errorf("processing failed: %v", err)
	    }

	    return nil
	}

The package is particularly useful in applications that need:
  - Detailed error tracking and debugging
  - Error cause chain analysis
  - Stack trace information
  - Structured error handling
  - Context-rich error reporting
  - Type-safe error metadata
*/
package errorx
