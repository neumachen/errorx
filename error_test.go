package errorx_test

import (
	"errors"
	"testing"

	"github.com/neumachen/errorx"
)

func TestNew(t *testing.T) {
	err := errorx.New("test error")

	if err == nil {
		t.Error("New should return a non-nil error")
	}

	if err.Error() != "test error" {
		t.Errorf("New error message mismatch. Expected: %s, Got: %s", "test error", err.Error())
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("cause error")
	err := errorx.Wrap(cause, 0)

	if err == nil {
		t.Error("Wrap should return a non-nil error")
	}

	if err.Error() != "cause error" {
		t.Errorf("Wrap error message mismatch. Expected: %s, Got: %s", "cause error", err.Error())
	}
}

func TestWrapPrefix(t *testing.T) {
	cause := errors.New("cause error")
	err := errorx.WrapPrefix(cause, "prefix", 0)

	if err == nil {
		t.Error("WrapPrefix should return a non-nil error")
	}

	expectedMsg := "prefix: cause error"
	if err.Error() != expectedMsg {
		t.Errorf("WrapPrefix error message mismatch. Expected: %s, Got: %s", expectedMsg, err.Error())
	}
}

func TestIs(t *testing.T) {
	cause := errors.New("cause error")
	err := errorx.Wrap(cause, 0)

	if !errorx.Is(err, cause) {
		t.Error("Is should return true for the same error object")
	}

	anotherCause := errors.New("another cause error")
	if errorx.Is(err, anotherCause) {
		t.Error("Is should return false for different error objects")
	}
}

func TestErrorf(t *testing.T) {
	err := errorx.Errorf("test %s", "error")

	if err == nil {
		t.Error("Errorf should return a non-nil error")
	}

	expectedMsg := "test error"
	if err.Error() != expectedMsg {
		t.Errorf("Errorf error message mismatch. Expected: %s, Got: %s", expectedMsg, err.Error())
	}
}
