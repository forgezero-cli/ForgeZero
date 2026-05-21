package verify

import (
	"os"
	"path/filepath"
	"testing"

	"fz/internal/utils"
)

func TestWriteManifestSecurePerm(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "manifest.json")
	file := filepath.Join(root, "tracked.txt")
	if err := os.WriteFile(file, []byte("data"), utils.FilePerm); err != nil {
		t.Fatal(err)
	}
	if err := WriteManifest(manifest, root); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != utils.FilePerm {
		t.Errorf("manifest perm = %o, want %o", info.Mode().Perm(), utils.FilePerm)
	}
}

func TestCollectFilesRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "real.txt")
	if err := os.WriteFile(target, []byte("x"), utils.FilePerm); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlink not supported")
	}
	_, err := collectFiles(root)
	if err == nil {
		t.Error("expected symlink rejection")
	}
}
