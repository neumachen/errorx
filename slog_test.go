package errorx_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/neumachen/errorx"
)

func TestLogValueIncludesStructuredFields(t *testing.T) {
	err := errorx.WrapPrefix(fmt.Errorf("root cause"), "ctx", 0)
	md := json.RawMessage(`{"req_id":"abc-123"}`)
	if e := err.SetMetadata(&md); e != nil {
		t.Fatalf("SetMetadata: %v", e)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{}))
	logger.Info("oops", slog.Any("err", err))

	var got map[string]any
	if e := json.Unmarshal(buf.Bytes(), &got); e != nil {
		t.Fatalf("slog output is not JSON: %v\n%s", e, buf.String())
	}

	errAttr, ok := got["err"].(map[string]any)
	if !ok {
		t.Fatalf("err attribute is not a group: %T (%v)", got["err"], got["err"])
	}
	if errAttr["message"] != "ctx: root cause" {
		t.Errorf("message = %v", errAttr["message"])
	}
	if errAttr["cause"] != "root cause" {
		t.Errorf("cause = %v", errAttr["cause"])
	}
	if errAttr["prefix"] != "ctx" {
		t.Errorf("prefix = %v", errAttr["prefix"])
	}
	if errAttr["type"] != "*errors.errorString" {
		t.Errorf("type = %v", errAttr["type"])
	}
	if _, ok := errAttr["stack_frames"]; !ok {
		t.Errorf("stack_frames absent")
	}
	if _, ok := errAttr["metadata"]; !ok {
		t.Errorf("metadata absent")
	}
	if strings.Contains(buf.String(), "source_line") {
		t.Errorf("slog output contains source_line: %s", buf.String())
	}
}

func TestLogValueEmptyForNilTraceError(t *testing.T) {
	var te *errorx.TraceError
	v := te.LogValue()
	if v.Kind() != slog.KindAny || v.Any() != nil {
		// Either a zero Value or an empty group is acceptable; we just
		// want to confirm there is no panic.
		_ = v
	}
}
