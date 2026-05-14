package errorx_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/neumachen/errorx"
)

func TestNewError(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		wantNil  bool
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
			name:    "nil error returns nil",
			input:   nil,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errorx.NewError(tt.input)
			if tt.wantNil {
				if err != nil {
					t.Fatalf("NewError(nil) = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("NewError returned nil unexpectedly")
			}
			if got := err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
			if got := err.Type(); got != tt.wantType {
				t.Errorf("Type() = %q, want %q", got, tt.wantType)
			}
			if len(err.StackFrames()) == 0 {
				t.Errorf("StackFrames empty")
			}
		})
	}
}

func TestNilHandling(t *testing.T) {
	if got := errorx.NewError(nil); got != nil {
		t.Errorf("NewError(nil) = %v, want nil", got)
	}
	if got := errorx.Wrap(nil, 0); got != nil {
		t.Errorf("Wrap(nil, 0) = %v, want nil", got)
	}
	if got := errorx.WrapPrefix(nil, "p", 0); got != nil {
		t.Errorf("WrapPrefix(nil, ...) = %v, want nil", got)
	}
}

func TestErrorsIsAndAs(t *testing.T) {
	sentinel := errors.New("sentinel")

	t.Run("Is through Wrap", func(t *testing.T) {
		wrapped := errorx.Wrap(sentinel, 0)
		if !errors.Is(wrapped, sentinel) {
			t.Errorf("errors.Is(Wrap(sentinel), sentinel) = false")
		}
		if !errorx.Is(wrapped, sentinel) {
			t.Errorf("errorx.Is(Wrap(sentinel), sentinel) = false")
		}
	})

	t.Run("Is through WrapPrefix", func(t *testing.T) {
		wrapped := errorx.WrapPrefix(sentinel, "ctx", 0)
		if !errors.Is(wrapped, sentinel) {
			t.Errorf("errors.Is(WrapPrefix(sentinel), sentinel) = false")
		}
	})

	t.Run("As through Wrap", func(t *testing.T) {
		typed := &customErr{code: 42}
		wrapped := errorx.Wrap(typed, 0)
		var target *customErr
		if !errors.As(wrapped, &target) {
			t.Fatalf("errors.As did not find *customErr through Wrap")
		}
		if target.code != 42 {
			t.Errorf("target.code = %d, want 42", target.code)
		}
	})

	t.Run("Is through nested wrappers", func(t *testing.T) {
		inner := fmt.Errorf("layer1: %w", sentinel)
		outer := errorx.Wrap(inner, 0)
		if !errors.Is(outer, sentinel) {
			t.Errorf("errors.Is did not traverse fmt.Errorf %%w through Wrap")
		}
	})

	t.Run("Is nil-nil", func(t *testing.T) {
		if !errorx.Is(nil, nil) {
			t.Errorf("Is(nil, nil) = false, want true")
		}
	})
}

type customErr struct{ code int }

func (c *customErr) Error() string { return fmt.Sprintf("custom err: %d", c.code) }

func TestWrapDoesNotMutateExisting(t *testing.T) {
	base := errors.New("base error")
	inner := errorx.WrapPrefix(base, "inner", 0)
	originalMsg := inner.Error()
	originalPrefix := inner.Prefix()
	originalStack := inner.Stack()

	outer := errorx.WrapPrefix(inner, "outer", 0)

	if inner.Error() != originalMsg {
		t.Errorf("inner.Error() changed: got %q, want %q", inner.Error(), originalMsg)
	}
	if inner.Prefix() != originalPrefix {
		t.Errorf("inner.Prefix() changed: got %q, want %q", inner.Prefix(), originalPrefix)
	}
	if want, got := "outer: inner: base error", outer.Error(); want != got {
		t.Errorf("outer.Error() = %q, want %q", got, want)
	}
	if outer.Prefix() != "outer" {
		t.Errorf("outer.Prefix() = %q, want %q", outer.Prefix(), "outer")
	}
	if len(outer.Stack()) == 0 {
		t.Errorf("outer.Stack() empty; expected fresh capture")
	}
	if len(originalStack) == 0 {
		t.Errorf("inner.Stack() unexpectedly empty")
	}
}

func TestWrapPrefix(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		prefix   string
		wantMsg  string
		wantType string
	}{
		{
			name:     "basic prefix",
			err:      fmt.Errorf("base error"),
			prefix:   "prefix",
			wantMsg:  "prefix: base error",
			wantType: "*errors.errorString",
		},
		{
			name:     "multiple prefixes",
			err:      errorx.WrapPrefix(fmt.Errorf("base error"), "prefix1", 0),
			prefix:   "prefix2",
			wantMsg:  "prefix2: prefix1: base error",
			wantType: "*errorx.TraceError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errorx.WrapPrefix(tt.err, tt.prefix, 0)
			if err == nil {
				t.Fatal("WrapPrefix returned nil")
			}
			if got := err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
			if got := err.Type(); got != tt.wantType {
				t.Errorf("Type() = %q, want %q", got, tt.wantType)
			}
			if len(err.StackFrames()) == 0 {
				t.Errorf("StackFrames empty")
			}
		})
	}
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
			wantMsg:   "wrapped error",
			wantCause: wrapError,
		},
		{
			name:    "wrap string-only error",
			err:     fmt.Errorf("string error"),
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
			if err == nil {
				t.Fatal("Wrap returned nil")
			}
			if got := err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Errorf("errors.Is did not find wantCause")
			}
			if len(err.StackFrames()) == 0 {
				t.Errorf("StackFrames empty")
			}
		})
	}
}

func TestErrorf(t *testing.T) {
	cases := []struct {
		name   string
		format string
		args   []any
		want   string
	}{
		{name: "simple", format: "test %s", args: []any{"error"}, want: "test error"},
		{name: "multi-arg", format: "%s: %d", args: []any{"count", 42}, want: "count: 42"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := errorx.Errorf(c.format, c.args...)
			if err == nil {
				t.Fatal("Errorf returned nil")
			}
			if got := err.Error(); got != c.want {
				t.Errorf("Error() = %q, want %q", got, c.want)
			}
			if got := err.Type(); got != "*errors.errorString" {
				t.Errorf("Type() = %q, want %q", got, "*errors.errorString")
			}
		})
	}
}

func TestErrorfPreservesWrap(t *testing.T) {
	sentinel := errors.New("sentinel")
	err := errorx.Errorf("ctx: %w", sentinel)
	if !errors.Is(err, sentinel) {
		t.Errorf("errors.Is did not see through Errorf %%w")
	}
}

func TestNewErrorfAliasOfErrorf(t *testing.T) {
	err := errorx.NewErrorf("legacy %s", "spelling")
	if err == nil || err.Error() != "legacy spelling" {
		t.Errorf("NewErrorf failed: %v", err)
	}
}

func TestCauseReturnsRoot(t *testing.T) {
	root := errors.New("root")
	wrapped := errorx.WrapPrefix(errorx.Wrap(root, 0), "ctx", 0)
	te, ok := wrapped.(*errorx.TraceError)
	if !ok {
		t.Fatalf("WrapPrefix returned %T, want *TraceError", wrapped)
	}
	if got := te.Cause(); got != root {
		t.Errorf("Cause() = %v, want %v", got, root)
	}
}

func TestUnwrapReturnsImmediate(t *testing.T) {
	root := errors.New("root")
	inner := errorx.Wrap(root, 0)
	outer := errorx.WrapPrefix(inner, "ctx", 0)

	got := errors.Unwrap(outer)
	if got != inner {
		t.Errorf("Unwrap(outer) = %v, want inner=%v", got, inner)
	}
}

func TestStackFramesReturnsCopy(t *testing.T) {
	err := errorx.NewError(errors.New("x"))
	a := err.StackFrames()
	if len(a) == 0 {
		t.Fatal("no frames")
	}
	a[0] = errorx.StackFrame{Name: "tampered"}
	b := err.StackFrames()
	if b[0].Name == "tampered" {
		t.Errorf("StackFrames did not return a copy: mutation leaked into internal state")
	}
}

func TestStackReturnsCopy(t *testing.T) {
	err := errorx.NewError(errors.New("x"))
	a := err.Stack()
	if len(a) == 0 {
		t.Fatal("no pcs")
	}
	a[0] = 0xdeadbeef
	b := err.Stack()
	if b[0] == 0xdeadbeef {
		t.Errorf("Stack did not return a copy")
	}
}

func TestMetadataReturnsCopy(t *testing.T) {
	err := errorx.NewError(errors.New("x"))
	md := json.RawMessage(`{"a":1}`)
	if e := err.SetMetadata(&md); e != nil {
		t.Fatalf("SetMetadata: %v", e)
	}
	got := err.Metadata()
	if got == nil {
		t.Fatal("Metadata returned nil")
	}
	(*got)[0] = 'X'
	again := err.Metadata()
	if again == nil || (*again)[0] != '{' {
		t.Errorf("Metadata did not return an independent copy")
	}
}

func TestMetadataValidation(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		err := errorx.NewError(errors.New("x"))
		md := json.RawMessage(`{"k":"v"}`)
		if e := err.SetMetadata(&md); e != nil {
			t.Errorf("SetMetadata(valid) returned error: %v", e)
		}
	})
	t.Run("nil clears", func(t *testing.T) {
		err := errorx.NewError(errors.New("x"))
		md := json.RawMessage(`{"k":"v"}`)
		if e := err.SetMetadata(&md); e != nil {
			t.Fatalf("SetMetadata: %v", e)
		}
		if e := err.SetMetadata(nil); e != nil {
			t.Errorf("SetMetadata(nil) returned error: %v", e)
		}
		if err.Metadata() != nil {
			t.Errorf("Metadata not cleared after SetMetadata(nil)")
		}
	})
	t.Run("invalid is rejected at set time", func(t *testing.T) {
		err := errorx.NewError(errors.New("x"))
		good := json.RawMessage(`{"a":1}`)
		if e := err.SetMetadata(&good); e != nil {
			t.Fatalf("seed metadata: %v", e)
		}
		bad := json.RawMessage(`{not json}`)
		if e := err.SetMetadata(&bad); e == nil {
			t.Errorf("SetMetadata(invalid) returned nil, want error")
		}
		md := err.Metadata()
		if md == nil || string(*md) != `{"a":1}` {
			t.Errorf("invalid SetMetadata overwrote previous value: %v", md)
		}
	})
}

func TestUnmarshalMetadata(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		type want struct {
			Key   string `json:"key"`
			Value int    `json:"value"`
		}
		err := errorx.NewError(errors.New("x"))
		md := json.RawMessage(`{"key":"test","value":123}`)
		if e := err.SetMetadata(&md); e != nil {
			t.Fatalf("SetMetadata: %v", e)
		}
		var w want
		if e := err.UnmarshalMetadata(&w); e != nil {
			t.Fatalf("UnmarshalMetadata: %v", e)
		}
		if w.Key != "test" || w.Value != 123 {
			t.Errorf("decoded = %+v", w)
		}
	})
	t.Run("absent", func(t *testing.T) {
		var w struct{ Key string }
		err := errorx.NewError(errors.New("x"))
		if e := err.UnmarshalMetadata(&w); e != nil {
			t.Errorf("UnmarshalMetadata(no metadata) returned error: %v", e)
		}
		if w.Key != "" {
			t.Errorf("expected zero-value target, got %+v", w)
		}
	})
}

func TestMarshalJSONRecord(t *testing.T) {
	err := errorx.WrapPrefix(fmt.Errorf("root cause"), "ctx", 0)
	md := json.RawMessage(`{"req":"abc"}`)
	if e := err.SetMetadata(&md); e != nil {
		t.Fatalf("SetMetadata: %v", e)
	}

	raw, e := json.Marshal(err)
	if e != nil {
		t.Fatalf("Marshal: %v", e)
	}

	var got map[string]any
	if e := json.Unmarshal(raw, &got); e != nil {
		t.Fatalf("Unmarshal: %v", e)
	}

	if got["message"] != "ctx: root cause" {
		t.Errorf("message = %v, want %q", got["message"], "ctx: root cause")
	}
	if got["cause"] != "root cause" {
		t.Errorf("cause = %v, want %q", got["cause"], "root cause")
	}
	if got["prefix"] != "ctx" {
		t.Errorf("prefix = %v, want %q", got["prefix"], "ctx")
	}
	if got["type"] != "*errors.errorString" {
		t.Errorf("type = %v, want %q", got["type"], "*errors.errorString")
	}
	if _, ok := got["stack_frames"]; !ok {
		t.Errorf("stack_frames absent")
	}
	if _, ok := got["metadata"]; !ok {
		t.Errorf("metadata absent")
	}
	if bytes.Contains(raw, []byte("source_line")) {
		t.Errorf("JSON contains source_line: %s", raw)
	}
}

func TestMarshalJSONNilTraceError(t *testing.T) {
	var te *errorx.TraceError
	raw, err := json.Marshal(te)
	if err != nil {
		t.Fatalf("Marshal(nil *TraceError): %v", err)
	}
	if string(raw) != "null" {
		t.Errorf("Marshal(nil *TraceError) = %q, want %q", raw, "null")
	}
}

func TestFormatVerbs(t *testing.T) {
	err := errorx.WrapPrefix(fmt.Errorf("base"), "ctx", 0)
	te := err.(*errorx.TraceError)

	if got := fmt.Sprintf("%s", te); got != "ctx: base" {
		t.Errorf("%%s = %q", got)
	}
	if got := fmt.Sprintf("%v", te); got != "ctx: base" {
		t.Errorf("%%v = %q", got)
	}
	if got := fmt.Sprintf("%q", te); got != `"ctx: base"` {
		t.Errorf("%%q = %q", got)
	}
	plus := fmt.Sprintf("%+v", te)
	if !strings.HasPrefix(plus, "ctx: base\n") {
		t.Errorf("%%+v should start with the error message")
	}
	if !strings.Contains(plus, "errorx_test") {
		t.Errorf("%%+v should contain stack info; got: %s", plus)
	}
}

func TestMaxStackDepthClamp(t *testing.T) {
	old := errorx.MaxStackDepth
	defer func() { errorx.MaxStackDepth = old }()

	errorx.MaxStackDepth = -1
	err := errorx.NewError(errors.New("x"))
	if err == nil {
		t.Fatal("NewError returned nil under negative MaxStackDepth")
	}
	if len(err.Stack()) == 0 {
		t.Errorf("Stack empty under clamped MaxStackDepth")
	}
}
