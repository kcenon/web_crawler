package plugin

import (
	"testing"

	"github.com/kcenon/web_crawler/pkg/storage"
)

func TestStorageBackends_ImplementStoragePlugin(t *testing.T) {
	// Verify Name() returns expected identifiers.
	tests := []struct {
		name string
		p    StoragePlugin
		want string
	}{
		{"FileStorage", storage.NewFileStorage(storage.FileConfig{Path: "/dev/null"}), "file"},
		{"CSVStorage", storage.NewCSVStorage(storage.CSVConfig{Path: "/dev/null"}), "csv"},
		{"PostgresPlugin", storage.NewPostgresPlugin(storage.PostgresConfig{DSN: "postgres://localhost/test"}), "postgres"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Name(); got != tt.want {
				t.Errorf("Name() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRegisterBuiltinStorage(t *testing.T) {
	r := NewRegistry()

	file := storage.NewFileStorage(storage.FileConfig{Path: "/dev/null"})
	csvs := storage.NewCSVStorage(storage.CSVConfig{Path: "/dev/null"})

	if err := RegisterBuiltinStorage(r, file, csvs); err != nil {
		t.Fatal(err)
	}

	names := r.ListStorage()
	if len(names) != 2 {
		t.Fatalf("registered %d plugins, want 2", len(names))
	}

	// Verify retrieval works.
	got, err := r.GetStorage("file")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name() != "file" {
		t.Errorf("got Name() = %q, want %q", got.Name(), "file")
	}
}

func TestRegisterBuiltinStorage_Duplicate(t *testing.T) {
	r := NewRegistry()

	file1 := storage.NewFileStorage(storage.FileConfig{Path: "/dev/null"})
	file2 := storage.NewFileStorage(storage.FileConfig{Path: "/dev/null"})

	if err := RegisterBuiltinStorage(r, file1); err != nil {
		t.Fatal(err)
	}

	// Registering a second plugin with the same Name should fail.
	err := RegisterBuiltinStorage(r, file2)
	if err == nil {
		t.Error("expected duplicate registration error")
	}
}
