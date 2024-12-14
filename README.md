# errorx

A comprehensive extension of the error handling package for Go that provides rich error context, stack traces, and error wrapping capabilities.

## Features

- Detailed stack traces with source line information
- Error wrapping with cause tracking
- Prefix support for error context
- JSON serialization
- Panic recovery and parsing
- Full compatibility with Go's standard error interface
- Comprehensive testing suite

## Installation

```bash
go get github.com/neumachen/errorx
```

## Key Features

- **Stack Traces**: Capture and format detailed stack traces with source line information
- **Error Wrapping**: Wrap errors while preserving the original error and adding context
- **Error Context**: Add prefixes to errors for better error context
- **Type Information**: Access underlying error types and causes
- **JSON Support**: Serialize errors to JSON for API responses
- **Source Line Info**: Get exact file and line information for debugging
- **Panic Handling**: Parse and convert panics into structured errors

## Usage

### Creating Errors

```go
// Create a new error
err := errorx.New("something went wrong")

// Create a formatted error
err := errorx.Errorf("failed to process %s", item)

// Wrap an existing error
err := errorx.Wrap(existingError, 0)

// Add context with prefix
err := errorx.WrapPrefix(err, "validation failed", 0)
```

### Accessing Error Information

```go
// Get the original cause
cause := err.Cause()

// Get stack frames
frames := err.StackFrames()

// Get error type
errType := err.Type()

// Get formatted stack trace
stack := err.RuntimeStack()

// Get error with context prefix
fmt.Println(err.Error()) // "validation failed: something went wrong"

// Set error metadata
metadata := json.RawMessage(`{"user_id": 123, "request_id": "abc-123"}`)
err.SetMetadata(&metadata)

// Get error metadata
meta := err.Metadata()

// Unmarshal metadata into a struct
type ErrorContext struct {
    UserID    int    `json:"user_id"`
    RequestID string `json:"request_id"`
}
var ctx ErrorContext
err.UnmarshalMetadata(&ctx)
```

### JSON Output Example

```json
{
  "cause": "something went wrong",
  "stack_frames": [
    {
      "file": "/path/to/file.go",
      "line_number": 42,
      "name": "FunctionName",
      "package": "package/path",
      "program_counter": "0x1234567"
    }
  ],
  "prefix": "validation failed",
  "metadata": {
    "user_id": 123,
    "request_id": "abc-123"
  },
  "stack": ["0x1234567", "0x89abcdef"]
}
```

## Documentation

For detailed documentation and examples, see the [Go package documentation](https://pkg.go.dev/github.com/neumachen/errorx).

## Testing

The package includes a comprehensive test suite. Run the tests with:

```bash
go test -v ./...
```

## License

See [LICENSE](LICENSE.md) file for details.
