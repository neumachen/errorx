package errorx_test

import (
	"errors"
	"fmt"
	"encoding/json"
	"testing"

	"github.com/neumachen/errorx"
	"github.com/stretchr/testify/require"
)

func TestNewError(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		wantMsg  string
		wantType string
	}{
		{
			name:     "basic error",
			input:    errors.New("test error"),
			wantMsg:  "test error",
			wantType: "*errors.errorString",
		},
		{
			name:     "formatted error",
			input:    fmt.Errorf("formatted %s", "error"),
			wantMsg:  "formatted error",
			wantType: "*errors.errorString",
		},
		{
			name:     "nil error",
			input:    nil,
			wantMsg:  "",
			wantType: "*errors.errorString",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errorx.NewError(tt.input)
			require.NotNil(t, err)
			require.Equal(t, tt.wantMsg, err.Error())
			require.Equal(t, tt.wantType, err.Type())
			require.NotEmpty(t, err.StackFrames())
		})
	}
}

func TestErrorMetadata(t *testing.T) {
	t.Run("SetMetadata with valid JSON", func(t *testing.T) {
		err := errorx.NewError(fmt.Errorf("test error"))
		metadata := json.RawMessage(`{"key": "value"}`)

		setErr := err.SetMetadata(&metadata)
		require.NoError(t, setErr)

		require.Equal(t, &metadata, err.Metadata())
	})

	t.Run("SetMetadata with nil", func(t *testing.T) {
		err := errorx.NewError(fmt.Errorf("test error"))

		setErr := err.SetMetadata(nil)
		require.NoError(t, setErr)

		require.Nil(t, err.Metadata())
	})

	t.Run("SetMetadata can be updated", func(t *testing.T) {
		err := errorx.NewError(fmt.Errorf("test error"))
		metadata1 := json.RawMessage(`{"first": true}`)
		metadata2 := json.RawMessage(`{"second": true}`)

		setErr := err.SetMetadata(&metadata1)
		require.NoError(t, setErr)
		require.Equal(t, &metadata1, err.Metadata())

		setErr = err.SetMetadata(&metadata2)
		require.NoError(t, setErr)
		require.Equal(t, &metadata2, err.Metadata())
	})
}

func TestErrorMetadataUnmarshal(t *testing.T) {
	t.Run("UnmarshalMetadata with valid data", func(t *testing.T) {
		type TestMetadata struct {
			Key   string `json:"key"`
			Value int    `json:"value"`
		}
		err := errorx.NewError(fmt.Errorf("test error"))
		metadata := json.RawMessage(`{"key": "test", "value": 123}`)

		setErr := err.SetMetadata(&metadata)
		require.NoError(t, setErr)

		var result TestMetadata
		unmarshalErr := err.UnmarshalMetadata(&result)
		require.NoError(t, unmarshalErr)
		require.Equal(t, "test", result.Key)
		require.Equal(t, 123, result.Value)
	})

	t.Run("UnmarshalMetadata with nil metadata", func(t *testing.T) {
		type TestMetadata struct {
			Key string `json:"key"`
		}
		err := errorx.NewError(fmt.Errorf("test error"))

		var result TestMetadata
		unmarshalErr := err.UnmarshalMetadata(&result)
		require.NoError(t, unmarshalErr)
		require.Empty(t, result.Key)
	})
}

func TestErrorJSON(t *testing.T) {
	t.Run("Marshal/Unmarshal error", func(t *testing.T) {
		// Create error with metadata
		originalErr := errorx.NewError(fmt.Errorf("test error"))
		metadata := json.RawMessage(`{"key": "value"}`)
		err := originalErr.SetMetadata(&metadata)
		require.NoError(t, err)

		// Marshal to JSON
		jsonBytes, err := json.Marshal(originalErr)
		require.NoError(t, err)
		require.NotEmpty(t, jsonBytes)

		// Unmarshal and verify fields
		var unmarshaled map[string]interface{}
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		require.NoError(t, err)

		require.Equal(t, "test error", unmarshaled["cause"])
		require.NotEmpty(t, unmarshaled["stack_frames"])
		require.NotEmpty(t, unmarshaled["stack"])
		require.Contains(t, unmarshaled, "metadata")
	})
}

func TestWrap(t *testing.T) {
	wrapError := errors.New("wrapped error")
	tests := []struct {
		name      string
		err       error
		skip      int
		wantMsg   string
		wantCause error
	}{
		{
			name:      "wrap error",
			err:       wrapError,
			skip:      0,
			wantMsg:   "wrapped error",
			wantCause: wrapError,
		},
		{
			name:    "wrap string",
			err:     fmt.Errorf("string error"),
			skip:    0,
			wantMsg: "string error",
		},
		{
			name:    "wrap with skip",
			err:     fmt.Errorf("skipped error"),
			skip:    1,
			wantMsg: "skipped error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errorx.Wrap(tt.err, tt.skip)
			require.NotNil(t, err)
			require.Equal(t, tt.wantMsg, err.Error())
			if tt.wantCause != nil {
				require.True(t, errorx.Is(err, tt.wantCause))
			}
			require.NotEmpty(t, err.StackFrames())
		})
	}
}

func TestWrapPrefix(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		prefix   string
		skip     int
		wantMsg  string
		wantType string
	}{
		{
			name:     "basic prefix",
			err:      fmt.Errorf("base error"),
			prefix:   "prefix",
			skip:     0,
			wantMsg:  "prefix: base error",
			wantType: "*errors.errorString",
		},
		{
			name:     "multiple prefixes",
			err:      errorx.WrapPrefix(fmt.Errorf("base error"), "prefix1", 0),
			prefix:   "prefix2",
			skip:     0,
			wantMsg:  "prefix2: prefix1: base error",
			wantType: "*errors.errorString",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errorx.WrapPrefix(tt.err, tt.prefix, tt.skip)
			require.NotNil(t, err)
			require.Equal(t, tt.wantMsg, err.Error())
			require.Equal(t, tt.wantType, err.Type())
			require.NotEmpty(t, err.StackFrames())
		})
	}
}

func TestIs(t *testing.T) {
	wrappedError := errors.New("wrapped error")
	tests := []struct {
		name        string
		err         error
		target      error
		wantIsEqual bool
	}{
		{
			name:        "same error",
			err:         errors.New("error"),
			target:      errors.New("error"),
			wantIsEqual: false, // Different error instances
		},
		{
			name:        "wrapped same error",
			err:         errorx.Wrap(wrappedError, 0),
			target:      wrappedError,
			wantIsEqual: true,
		},
		{
			name:        "different errors",
			err:         errors.New("error1"),
			target:      errors.New("error2"),
			wantIsEqual: false,
		},
		{
			name:        "nil errors",
			err:         nil,
			target:      nil,
			wantIsEqual: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isEqual := errorx.Is(tt.err, tt.target)
			require.Equal(t, tt.wantIsEqual, isEqual)
		})
	}
}

func TestErrorf(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []any
		wantMsg  string
		wantType string
	}{
		{
			name:     "simple format",
			format:   "test %s",
			args:     []any{"error"},
			wantMsg:  "test error",
			wantType: "*errors.errorString",
		},
		{
			name:     "multiple args",
			format:   "%s: %d",
			args:     []any{"count", 42},
			wantMsg:  "count: 42",
			wantType: "*errors.errorString",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errorx.NewErrorf(tt.format, tt.args...)
			require.NotNil(t, err)
			require.Equal(t, tt.wantMsg, err.Error())
			require.Equal(t, tt.wantType, err.Type())
			require.NotEmpty(t, err.StackFrames())
		})
	}
}
