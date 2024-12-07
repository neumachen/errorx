package errorx

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// StackFrame contains information about a stack frame in a call stack.
type StackFrame struct {
	File           string  `json:"file"`            // The path to the file containing this ProgramCounter
	LineNumber     int     `json:"line_number"`     // The line number in that file
	Name           string  `json:"name"`            // The name of the function that contains this ProgramCounter
	Package        string  `json:"package"`         // The package that contains this function
	ProgramCounter uintptr `json:"program_counter"` // The underlying ProgramCounter
}

// NewStackFrame populates a StackFrame object from the program counter.
func NewStackFrame(newProgramCounter uintptr) StackFrame {
	newStackFrame := StackFrame{ProgramCounter: newProgramCounter}
	if newStackFrame.Func() == nil {
		return newStackFrame
	}
	newStackFrame.Package, newStackFrame.Name = packageAndName(newStackFrame.Func())

	// pc -1 because the program counters we use are usually return addresses,
	// and we want to show the line that corresponds to the function call
	newStackFrame.File, newStackFrame.LineNumber = newStackFrame.Func().FileLine(newProgramCounter - 1)
	return newStackFrame
}

// Func returns the function that contained this frame.
func (s StackFrame) Func() *runtime.Func {
	if s.ProgramCounter == 0 {
		return nil
	}
	return runtime.FuncForPC(s.ProgramCounter)
}

// String returns the formatted stack frame, similar to how Go does in runtime/debug.Stack().
func (s StackFrame) String() string {
	str := fmt.Sprintf("file:%s line_number%d (0x%x)\n", s.File, s.LineNumber, s.ProgramCounter)

	source, err := s.SourceLine()
	if err != nil {
		return str
	}

	return str + fmt.Sprintf("\t%s: %s\n", s.Name, source)
}

// SourceLine gets the line of code (from File and Line) of the original source if possible.
func (s *StackFrame) SourceLine() (string, error) {
	data, err := os.ReadFile(s.File)
	if err != nil {
		return "", NewError(err)
	}

	lines := bytes.Split(data, []byte{'\n'})
	if s.LineNumber <= 0 || s.LineNumber >= len(lines) {
		return "???", nil
	}
	// -1 because line-numbers are 1 based, but our array is 0 based
	return string(bytes.Trim(lines[s.LineNumber-1], " \t")), nil
}

// packageAndName extracts the package and name from the function.
func packageAndName(fn *runtime.Func) (string, string) {
	name := fn.Name()
	pkg := ""

	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//  runtime/debug.*T·ptrmethod
	// and want
	//  *T.ptrmethod
	// Since the package path might contains dots (e.g. code.google.com/...),
	// we first remove the path prefix if there is one.
	if lastslash := strings.LastIndex(name, "/"); lastslash >= 0 {
		pkg += name[:lastslash] + "/"
		name = name[lastslash+1:]
	}
	if period := strings.Index(name, "."); period >= 0 {
		pkg += name[:period]
		name = name[period+1:]
	}

	name = strings.Replace(name, "·", ".", -1)
	return pkg, name
}
