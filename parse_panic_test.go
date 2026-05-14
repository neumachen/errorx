package errorx_test

import (
	"reflect"
	"runtime/debug"
	"testing"

	"github.com/neumachen/errorx"
)

var createdBy = `panic: hello!

goroutine 54 [running]:
runtime.panic(0x35ce40, 0xc208039db0)
	/0/c/go/src/pkg/runtime/panic.c:279 +0xf5
github.com/neumachen/testgo/webkit/g/app/controllers.func.001()
	/0/go/src/github.com/neumachen/testgo/webkit/g/app/controllers/app.go:13 +0x74
net/http.(*Server).Serve(0xc20806c780, 0x910c88, 0xc20803e168, 0x0, 0x0)
	/0/c/go/src/pkg/net/http/server.go:1698 +0x91
created by github.com/neumachen/testgo/webkit/g/app/controllers.App.Index
	/0/go/src/github.com/neumachen/testgo/webkit/g/app/controllers/app.go:14 +0x3e

goroutine 16 [IO wait]:
net.runtime_pollWait(0x911c30, 0x72, 0x0)
	/0/c/go/src/pkg/runtime/netpoll.goc:146 +0x66
`

var normalSplit = `panic: hello!

goroutine 54 [running]:
runtime.panic(0x35ce40, 0xc208039db0)
	/0/c/go/src/pkg/runtime/panic.c:279 +0xf5
github.com/neumachen/testgo/webkit/g/app/controllers.func.001()
	/0/go/src/github.com/neumachen/testgo/webkit/g/app/controllers/app.go:13 +0x74
net/http.(*Server).Serve(0xc20806c780, 0x910c88, 0xc20803e168, 0x0, 0x0)
	/0/c/go/src/pkg/net/http/server.go:1698 +0x91

goroutine 16 [IO wait]:
net.runtime_pollWait(0x911c30, 0x72, 0x0)
	/0/c/go/src/pkg/runtime/netpoll.goc:146 +0x66
`

var lastGoroutine = `panic: hello!

goroutine 16 [IO wait]:
net.runtime_pollWait(0x911c30, 0x72, 0x0)
	/0/c/go/src/pkg/runtime/netpoll.goc:146 +0x66

goroutine 54 [running]:
runtime.panic(0x35ce40, 0xc208039db0)
	/0/c/go/src/pkg/runtime/panic.c:279 +0xf5
github.com/neumachen/testgo/webkit/g/app/controllers.func.001()
	/0/go/src/github.com/neumachen/testgo/webkit/g/app/controllers/app.go:13 +0x74
net/http.(*Server).Serve(0xc20806c780, 0x910c88, 0xc20803e168, 0x0, 0x0)
	/0/c/go/src/pkg/net/http/server.go:1698 +0x91
`

var baseFrames = []errorx.StackFrame{
	{
		File:       "/0/c/go/src/pkg/runtime/panic.c",
		LineNumber: 279,
		Name:       "panic",
		Package:    "runtime",
	},
	{
		File:       "/0/go/src/github.com/neumachen/testgo/webkit/g/app/controllers/app.go",
		LineNumber: 13,
		Name:       "func.001",
		Package:    "github.com/neumachen/testgo/webkit/g/app/controllers",
	},
	{
		File:       "/0/c/go/src/pkg/net/http/server.go",
		LineNumber: 1698,
		Name:       "(*Server).Serve",
		Package:    "net/http",
	},
}

var createdByExpected = append(append([]errorx.StackFrame{}, baseFrames...),
	errorx.StackFrame{
		File:       "/0/go/src/github.com/neumachen/testgo/webkit/g/app/controllers/app.go",
		LineNumber: 14,
		Name:       "App.Index",
		Package:    "github.com/neumachen/testgo/webkit/g/app/controllers",
	})

func TestParsePanic(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []errorx.StackFrame
	}{
		{name: "createdBy", input: createdBy, want: createdByExpected},
		{name: "normalSplit", input: normalSplit, want: baseFrames},
		{name: "lastGoroutine", input: lastGoroutine, want: baseFrames},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errx, err := errorx.ParsePanic(tc.input)
			if err != nil {
				t.Fatalf("ParsePanic(%s): %v", tc.name, err)
			}
			if got := errx.Type(); got != "panic" {
				t.Errorf("Type = %q, want %q", got, "panic")
			}
			if got := errx.Error(); got != "hello!" {
				t.Errorf("Error = %q, want %q", got, "hello!")
			}
			frames := errx.StackFrames()
			if len(frames) > 0 && frames[0].Func() != nil {
				t.Errorf("frames[0].Func() should be nil for parsed panic frames")
			}
			if !reflect.DeepEqual(frames, tc.want) {
				t.Errorf("frames mismatch: got=%#v want=%#v", frames, tc.want)
			}
		})
	}
}

func TestParsePanicEmptyAndGarbage(t *testing.T) {
	cases := []string{
		"",
		"not a panic\n",
		"panic: only one line",
		"panic: x\n\ngoroutine 1 [running]:\nonly one half",
	}
	for _, in := range cases {
		_, err := errorx.ParsePanic(in)
		if err == nil {
			t.Errorf("ParsePanic(%q) returned nil error; want error", in)
		}
	}
}

func TestFromPanicRecover(t *testing.T) {
	var err *errorx.TraceError
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = errorx.FromPanic(r, debug.Stack())
			}
		}()
		panic("boom")
	}()

	if err == nil {
		t.Fatal("FromPanic returned nil error after recover")
	}
	if err.Type() != "panic" {
		t.Errorf("Type = %q, want %q", err.Type(), "panic")
	}
	if err.Error() != "boom" {
		t.Errorf("Error = %q, want %q", err.Error(), "boom")
	}
	if rs := err.RuntimeStack(); len(rs) == 0 {
		t.Errorf("RuntimeStack is empty for FromPanic with debug.Stack()")
	}
}

func TestFromPanicNoStackArgumentCapturesFromCaller(t *testing.T) {
	err := errorx.FromPanic("oops", nil)
	if err == nil {
		t.Fatal("FromPanic returned nil")
	}
	if err.Error() != "oops" {
		t.Errorf("Error = %q, want %q", err.Error(), "oops")
	}
	frames := err.StackFrames()
	if len(frames) == 0 {
		t.Fatal("expected at least one stack frame when debug stack not supplied")
	}
}

// FuzzParsePanic ensures the parser is robust against arbitrary input. It
// must never panic and may either succeed or return an error.
func FuzzParsePanic(f *testing.F) {
	f.Add(createdBy)
	f.Add(normalSplit)
	f.Add(lastGoroutine)
	f.Add("")
	f.Add("panic: x\n")
	f.Add("panic: x\n\ngoroutine 1 [running]:\nfoo()\n\tfile.go:1 +0x1\n")
	f.Add(string(debug.Stack()))

	f.Fuzz(func(t *testing.T, s string) {
		got, err := errorx.ParsePanic(s)
		if err != nil && got != nil {
			t.Fatalf("ParsePanic returned (non-nil, non-nil) for %q", s)
		}
		if got != nil {
			_ = got.Error()
			_ = got.Type()
			_ = got.StackFrames()
		}
	})
}
