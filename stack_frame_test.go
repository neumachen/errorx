package errorx_test

import (
	"runtime"
	"testing"

	"github.com/neumachen/errorx"
	"github.com/stretchr/testify/require"
)

func TestNewStackFrame(t *testing.T) {
	// Get current PC for testing
	pc, _, _, ok := runtime.Caller(0)
	require.True(t, ok, "Failed to get current PC")

	tests := []struct {
		name string
		pc   uintptr
		want struct {
			hasFunc     bool
			pkgContains string
			namePrefix  string
		}
	}{
		{
			name: "valid program counter",
			pc:   pc,
			want: struct {
				hasFunc     bool
				pkgContains string
				namePrefix  string
			}{
				hasFunc:     true,
				pkgContains: "errorx_test",
				namePrefix:  "TestNewStackFrame",
			},
		},
		{
			name: "zero program counter",
			pc:   0,
			want: struct {
				hasFunc     bool
				pkgContains string
				namePrefix  string
			}{
				hasFunc: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := errorx.NewStackFrame(tt.pc)

			if tt.want.hasFunc {
				require.NotNil(t, frame.Func())
				require.Contains(t, frame.Package, tt.want.pkgContains)
				require.True(t, len(frame.Name) > 0)
				require.True(t, len(frame.File) > 0)
				require.Greater(t, frame.LineNumber, 0)
			} else {
				require.Nil(t, frame.Func())
			}
		})
	}
}

func TestStackFrame_String(t *testing.T) {
	pc, _, _, ok := runtime.Caller(0)
	require.True(t, ok, "Failed to get current PC")

	tests := []struct {
		name       string
		frame      func() errorx.StackFrame
		wantPrefix string
	}{
		{
			name: "valid frame",
			frame: func() errorx.StackFrame {
				return errorx.NewStackFrame(pc)
			},
			wantPrefix: "file:",
		},
		{
			name: "zero PC frame",
			frame: func() errorx.StackFrame {
				return errorx.NewStackFrame(0)
			},
			wantPrefix: "file:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := tt.frame()
			str := frame.String()
			require.Contains(t, str, tt.wantPrefix)
		})
	}
}

func TestStackFrame_SourceLine(t *testing.T) {
	pc, _, _, ok := runtime.Caller(0)
	require.True(t, ok, "Failed to get current PC")

	tests := []struct {
		name    string
		frame   func() errorx.StackFrame
		wantErr bool
	}{
		{
			name: "valid frame",
			frame: func() errorx.StackFrame {
				return errorx.NewStackFrame(pc)
			},
			wantErr: false,
		},
		{
			name: "invalid file path",
			frame: func() errorx.StackFrame {
				f := errorx.NewStackFrame(pc)
				f.File = "nonexistent/file.go"
				return f
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := tt.frame()
			line, err := frame.SourceLine()

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, line)
			}
		})
	}
}
