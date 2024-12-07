package errorx_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/neumachen/errorx"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		wantMsg  string
		wantType string
	}{
		{
			name:     "string input",
			input:    "test error",
			wantMsg:  "test error",
			wantType: "*errors.errorString",
		},
		{
			name:     "error input",
			input:    errors.New("wrapped error"),
			wantMsg:  "wrapped error",
			wantType: "*errors.errorString",
		},
		{
			name:     "formatted input",
			input:    fmt.Errorf("formatted %s", "error"),
			wantMsg:  "formatted error",
			wantType: "*errors.errorString",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errorx.New(tt.input)
			require.NotNil(t, err)
			require.Equal(t, tt.wantMsg, err.Error())
			require.Equal(t, tt.wantType, err.Type())
			require.NotEmpty(t, err.StackFrames())
		})
	}
}

func TestWrap(t *testing.T) {
	wrapError := errors.New("wrapped error")
	tests := []struct {
		name      string
		err       any
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
			err:     "string error",
			skip:    0,
			wantMsg: "string error",
		},
		{
			name:    "wrap with skip",
			err:     "skipped error",
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
		err      any
		prefix   string
		skip     int
		wantMsg  string
		wantType string
	}{
		{
			name:     "basic prefix",
			err:      "base error",
			prefix:   "prefix",
			skip:     0,
			wantMsg:  "prefix: base error",
			wantType: "*errors.errorString",
		},
		{
			name:     "multiple prefixes",
			err:      errorx.WrapPrefix("base error", "prefix1", 0),
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
			err := errorx.Errorf(tt.format, tt.args...)
			require.NotNil(t, err)
			require.Equal(t, tt.wantMsg, err.Error())
			require.Equal(t, tt.wantType, err.Type())
			require.NotEmpty(t, err.StackFrames())
		})
	}
}
