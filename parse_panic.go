package errorx

import (
	"strconv"
	"strings"
)

// uncaughtPanic represents an uncaught panic.
type uncaughtPanic struct{ message string }

func (p uncaughtPanic) Error() string {
	return p.message
}

// ParsePanic parses the panic information from the given text and returns an Error reference.
func ParsePanic(panicToParse string) (Error, error) {
	lines := strings.Split(panicToParse, "\n")

	state := "start"
	var message string
	var stack []StackFrame

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if state == "start" {
			if strings.HasPrefix(line, "panic: ") {
				message = strings.TrimPrefix(line, "panic: ")
				state = "seek"
			} else {
				return nil, Errorf("errorx.PanicParser: Invalid line (no prefix): %s", line)
			}
		} else if state == "seek" {
			if strings.HasPrefix(line, "goroutine ") && strings.HasSuffix(line, "[running]:") {
				state = "parsing"
			}
		} else if state == "parsing" {
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
				return nil, Errorf("errorx.PanicParser: Invalid line (unpaired): %s", line)
			}

			frame, err := parsePanicFrame(line, lines[i], createdBy)
			if err != nil {
				return nil, err
			}

			stack = append(stack, *frame)
			if createdBy {
				state = "done"
				break
			}
		}
	}

	if state == "done" || state == "parsing" {
		return &errorData{cause: uncaughtPanic{message}, stackFrames: stack}, nil
	}
	return nil, Errorf("could not parse panic: %v", panicToParse)
}

// parsePanicFrame parses a single stack frame from the panic information.
func parsePanicFrame(name string, line string, createdBy bool) (*StackFrame, error) {
	idx := strings.LastIndex(name, "(")
	if idx == -1 && !createdBy {
		return nil, Errorf("errorx.PanicParser: Invalid line (no call): %s", name)
	}
	if idx != -1 {
		name = name[:idx]
	}
	pkg := ""

	if lastslash := strings.LastIndex(name, "/"); lastslash >= 0 {
		pkg += name[:lastslash] + "/"
		name = name[lastslash+1:]
	}
	if period := strings.Index(name, "."); period >= 0 {
		pkg += name[:period]
		name = name[period+1:]
	}

	name = strings.Replace(name, "·", ".", -1)

	if !strings.HasPrefix(line, "\t") {
		return nil, Errorf("errorx.PanicParser: Invalid line (no tab): %s", line)
	}

	idx = strings.LastIndex(line, ":")
	if idx == -1 {
		return nil, Errorf("errorx.PanicParser: Invalid line (no line number): %s", line)
	}
	file := line[1:idx]

	number := line[idx+1:]
	if idx = strings.Index(number, " +"); idx > -1 {
		number = number[:idx]
	}

	lno, err := strconv.ParseInt(number, 10, 32)
	if err != nil {
		return nil, Errorf("errorx.PanicParser: Invalid line (bad line number): %s", line)
	}

	return &StackFrame{
		File:       file,
		LineNumber: int(lno),
		Package:    pkg,
		Name:       name,
	}, nil
}
