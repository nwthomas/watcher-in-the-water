package ipstate

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_missingFile(t *testing.T) {
	t.Parallel()
	s, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if s.PublicIP != "" {
		t.Fatalf("got %+v", s)
	}
}

func TestSaveLoad_roundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "state.json")
	want := State{
		PublicIP:  "192.0.2.1",
		UpdatedAt: time.Date(2025, 3, 22, 12, 0, 0, 0, time.UTC),
	}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.PublicIP != want.PublicIP {
		t.Fatalf("PublicIP = %q", got.PublicIP)
	}
	if !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Fatalf("UpdatedAt = %v want %v", got.UpdatedAt, want.UpdatedAt)
	}
}

func TestSave_createsParentDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "nested", "state.json")
	if err := Save(path, State{PublicIP: "192.0.2.5", UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
