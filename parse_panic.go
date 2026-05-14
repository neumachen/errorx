package errorx

import (
	"fmt"
	"strconv"
	"strings"
)

// uncaughtPanic carries the message recovered from a panic. It is used as
// the underlying cause for errors produced by ParsePanic and FromPanic and
// is what causes Type() to report "panic".
type uncaughtPanic struct{ message string }

func (p uncaughtPanic) Error() string { return p.message }

// ParsePanic converts a panic stack-trace string into a *TraceError. The
// input is expected to start with "panic: <message>" followed by the
// "goroutine N [running]:" section emitted by the Go runtime. Frames are
// parsed without recording program counters; StackFrames() will return the
// parsed entries as-is.
//
// ParsePanic never panics on malformed input; it returns a descriptive
// error instead. For newer code that has access to the recovered value and
// the raw debug.Stack() bytes, prefer FromPanic.
func ParsePanic(panicToParse string) (Error, error) {
	if panicToParse == "" {
		return nil, fmt.Errorf("errorx.ParsePanic: empty input")
	}
	lines := strings.Split(panicToParse, "\n")

	state := "start"
	var message string
	var stack []StackFrame

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		switch state {
		case "start":
			if strings.HasPrefix(line, "panic: ") {
				message = strings.TrimPrefix(line, "panic: ")
				state = "seek"
				continue
			}
			return nil, fmt.Errorf("errorx.ParsePanic: invalid line (no prefix): %q", line)

		case "seek":
			if strings.HasPrefix(line, "goroutine ") && strings.HasSuffix(line, "[running]:") {
				state = "parsing"
			}

		case "parsing":
			if line == "" {
				state = "done"
				break
			}
			createdBy := false
			if strings.HasPrefix(line, "created by ") {
				line = strings.TrimPrefix(line, "created by ")
				createdBy = true
			}

			i++
			if i >= len(lines) {
				return nil, fmt.Errorf("errorx.ParsePanic: invalid line (unpaired): %q", line)
			}

			frame, err := parsePanicFrame(line, lines[i], createdBy)
			if err != nil {
				return nil, err
			}

			stack = append(stack, *frame)
			if createdBy {
				state = "done"
			}
		}

		if state == "done" {
			break
		}
	}

	if state == "done" || state == "parsing" {
		te := &TraceError{
			cause:        uncaughtPanic{message: message},
			parsedFrames: stack,
		}
		return te, nil
	}
	return nil, fmt.Errorf("errorx.ParsePanic: could not parse panic: %q", panicToParse)
}

// parsePanicFrame parses one (function, file:line) pair from the panic
// stack section.
func parsePanicFrame(name string, line string, createdBy bool) (*StackFrame, error) {
	idx := strings.LastIndex(name, "(")
	if idx == -1 && !createdBy {
		return nil, fmt.Errorf("errorx.ParsePanic: invalid line (no call): %q", name)
	}
	if idx != -1 {
		name = name[:idx]
	}

	pkg, fname := splitPackageAndName(name)

	if !strings.HasPrefix(line, "\t") {
		return nil, fmt.Errorf("errorx.ParsePanic: invalid line (no tab): %q", line)
	}

	idx = strings.LastIndex(line, ":")
	if idx == -1 {
		return nil, fmt.Errorf("errorx.ParsePanic: invalid line (no line number): %q", line)
	}
	file := line[1:idx]

	number := line[idx+1:]
	if sp := strings.Index(number, " +"); sp > -1 {
		number = number[:sp]
	}

	lno, err := strconv.ParseInt(number, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("errorx.ParsePanic: invalid line (bad line number): %q", line)
	}

	return &StackFrame{
		File:       file,
		LineNumber: int(lno),
		Package:    pkg,
		Name:       fname,
	}, nil
}
