package plugin

import (
	"errors"
	"testing"
)

func TestErrDuplicatePlugin_Error(t *testing.T) {
	err := &ErrDuplicatePlugin{Category: "storage", Name: "pg"}
	want := `plugin: storage plugin "pg" already registered`
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestErrPluginNotFound_Error(t *testing.T) {
	err := &ErrPluginNotFound{Category: "parser", Name: "xml"}
	want := `plugin: parser plugin "xml" not found`
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestErrNilPlugin_Error(t *testing.T) {
	err := &ErrNilPlugin{Category: "notifier"}
	want := "plugin: cannot register nil notifier plugin"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestErrors_AsType(t *testing.T) {
	// Verify errors.As works with our custom types.
	dup := &ErrDuplicatePlugin{Category: "storage", Name: "pg"}
	var target *ErrDuplicatePlugin
	if !errors.As(dup, &target) {
		t.Error("errors.As failed for ErrDuplicatePlugin")
	}

	nf := &ErrPluginNotFound{Category: "parser", Name: "xml"}
	var target2 *ErrPluginNotFound
	if !errors.As(nf, &target2) {
		t.Error("errors.As failed for ErrPluginNotFound")
	}

	nilp := &ErrNilPlugin{Category: "exporter"}
	var target3 *ErrNilPlugin
	if !errors.As(nilp, &target3) {
		t.Error("errors.As failed for ErrNilPlugin")
	}
}
