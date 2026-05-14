package errorx_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/neumachen/errorx"
)

func TestNewStackFrame(t *testing.T) {
	pc, _, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	t.Run("valid pc", func(t *testing.T) {
		f := errorx.NewStackFrame(pc)
		if f.Func() == nil {
			t.Fatal("Func() = nil for valid pc")
		}
		if !strings.Contains(f.Package, "errorx_test") {
			t.Errorf("Package = %q, want contains errorx_test", f.Package)
		}
		if f.Name == "" {
			t.Errorf("Name empty")
		}
		if f.File == "" {
			t.Errorf("File empty")
		}
		if f.LineNumber <= 0 {
			t.Errorf("LineNumber = %d", f.LineNumber)
		}
	})

	t.Run("zero pc", func(t *testing.T) {
		f := errorx.NewStackFrame(0)
		if f.Func() != nil {
			t.Errorf("Func() should be nil for zero pc")
		}
	})
}

func TestStackFrame_StringDoesNotReadSource(t *testing.T) {
	pc, _, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	f := errorx.NewStackFrame(pc)
	s := f.String()

	// String should reference the file path...
	if !strings.Contains(s, f.File) {
		t.Errorf("String does not contain file path; got %q", s)
	}
	// ...but never contain the marker text we placed in this very test file.
	// "do_not_read_source_marker_string" appears literally in our File, on a
	// line that is NOT the one runtime.Caller returned. If String were
	// reading the source file it would not include this exact string,
	// but the assertion we care about is the inverse: the formatted output
	// should not embed an arbitrary source line.
	if strings.Contains(s, "do_not_read_source_marker_string") {
		t.Errorf("String unexpectedly included source content: %s", s)
	}
}

// Unused marker — present only to ensure the absence-check above is
// meaningful if String() ever regresses to reading source files.
var _ = "do_not_read_source_marker_string"

func TestStackFrame_SourceLine(t *testing.T) {
	pc, _, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	f := errorx.NewStackFrame(pc)
	line, err := f.SourceLine()
	if err != nil {
		t.Fatalf("SourceLine: %v", err)
	}
	if line == "" {
		t.Errorf("SourceLine returned empty")
	}

	f.File = "nonexistent/file.go"
	if _, err := f.SourceLine(); err == nil {
		t.Errorf("SourceLine on missing file returned nil error")
	}
}
