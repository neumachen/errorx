package errorx

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// StackFrame describes a single resolved frame of a captured stack. The
// fields carry source-location metadata but never contain the source-code
// line itself; the latter is available on demand via SourceLine.
type StackFrame struct {
	// File is the absolute path to the source file containing the frame.
	File string `json:"file"`
	// LineNumber is the 1-based line within File.
	LineNumber int `json:"line_number"`
	// Name is the function name with any package prefix stripped.
	Name string `json:"name"`
	// Package is the import path of the package that contains the
	// function, including a trailing slash when applicable.
	Package string `json:"package"`
	// ProgramCounter is the raw runtime program counter for the frame.
	// It may be zero for frames that were parsed from a text stack trace.
	ProgramCounter uintptr `json:"program_counter"`
}

// NewStackFrame builds a StackFrame from a program counter. Frames whose
// program counter does not resolve to a known function are returned with
// only ProgramCounter populated.
func NewStackFrame(pc uintptr) StackFrame {
	f := StackFrame{ProgramCounter: pc}
	fn := f.Func()
	if fn == nil {
		return f
	}
	f.Package, f.Name = packageAndName(fn)
	// pc-1 because the recorded program counters are usually return
	// addresses; subtracting one lands on the calling instruction.
	f.File, f.LineNumber = fn.FileLine(pc - 1)
	return f
}

// Func returns the runtime.Func describing the frame, or nil if the program
// counter does not resolve.
func (s StackFrame) Func() *runtime.Func {
	if s.ProgramCounter == 0 {
		return nil
	}
	return runtime.FuncForPC(s.ProgramCounter)
}

// String returns a one-frame description suitable for diagnostic stack
// dumps. The format is:
//
//	<package>/<name>
//	\t<file>:<line> +0x<pc>
//
// String never reads source files from disk. To attach the source line, call
// SourceLine explicitly and append it.
func (s StackFrame) String() string {
	var b strings.Builder
	if s.Package != "" || s.Name != "" {
		b.WriteString(s.Package)
		b.WriteString(s.Name)
		b.WriteByte('\n')
	}
	b.WriteByte('\t')
	if s.File != "" {
		b.WriteString(s.File)
		fmt.Fprintf(&b, ":%d", s.LineNumber)
	}
	fmt.Fprintf(&b, " +0x%x\n", s.ProgramCounter)
	return b.String()
}

// SourceLine returns the line of source code referenced by the frame. It
// reads the file from disk, so callers should treat the result as opt-in
// debug aid and avoid invoking it on hot paths or untrusted file paths.
func (s *StackFrame) SourceLine() (string, error) {
	if s.File == "" {
		return "", errors.New("errorx: StackFrame.SourceLine: no file recorded")
	}
	data, err := os.ReadFile(s.File)
	if err != nil {
		return "", fmt.Errorf("errorx: SourceLine: %w", err)
	}
	lines := bytes.Split(data, []byte{'\n'})
	if s.LineNumber <= 0 || s.LineNumber >= len(lines) {
		return "???", nil
	}
	return string(bytes.Trim(lines[s.LineNumber-1], " \t")), nil
}

// packageAndName splits the runtime function's qualified name into a package
// path (with trailing slash for the leading import path component, when
// present) and a function name with center-dots normalized.
func packageAndName(fn *runtime.Func) (string, string) {
	return splitPackageAndName(fn.Name())
}

// splitPackageAndName performs the same split on a raw qualified function
// name, as returned by runtime.Frame.Function.
func splitPackageAndName(qualified string) (string, string) {
	name := qualified
	pkg := ""

	if lastslash := strings.LastIndex(name, "/"); lastslash >= 0 {
		pkg += name[:lastslash] + "/"
		name = name[lastslash+1:]
	}
	if period := strings.Index(name, "."); period >= 0 {
		pkg += name[:period]
		name = name[period+1:]
	}

	name = strings.ReplaceAll(name, "·", ".")
	return pkg, name
}
