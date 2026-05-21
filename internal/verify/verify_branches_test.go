package verify

import (
	"os"
	"path/filepath"
	"testing"

	"fz/internal/utils"
)

func TestLoadManifestInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := utils.SecureWriteFile(path, []byte("{")); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest(path); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestWriteManifestInvalidRoot(t *testing.T) {
	dir := t.TempDir()
	if err := WriteManifest(filepath.Join(dir, "m.json"), "../evil"); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestVerifyRootMissingFile(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "proj")
	if err := os.MkdirAll(root, utils.DirPerm); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "a.txt")
	if err := os.WriteFile(file, []byte("1"), utils.FilePerm); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(dir, "manifest.json")
	if err := WriteManifest(manifest, root); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(file)
	result, err := VerifyRoot(root, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Missing) != 1 {
		t.Fatalf("missing = %v", result.Missing)
	}
}

func TestVerifyRootExtraFile(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "proj")
	if err := os.MkdirAll(root, utils.DirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("1"), utils.FilePerm); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(dir, "manifest.json")
	if err := WriteManifest(manifest, root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("2"), utils.FilePerm); err != nil {
		t.Fatal(err)
	}
	result, err := VerifyRoot(root, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Extra) != 1 {
		t.Fatalf("extra = %v", result.Extra)
	}
}

func TestCollectFilesInvalidRel(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "sub")
	if err := os.MkdirAll(nested, utils.DirPerm); err != nil {
		t.Fatal(err)
	}
	files, err := collectFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	_ = files
}

func TestBuildManifestInvalidRoot(t *testing.T) {
	if _, err := BuildManifest("../bad"); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoadManifestValidateFail(t *testing.T) {
	if _, err := LoadManifest("../etc/passwd"); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestShouldSkipDirGit(t *testing.T) {
	root := t.TempDir()
	git := filepath.Join(root, ".git", "objects")
	if err := os.MkdirAll(git, utils.DirPerm); err != nil {
		t.Fatal(err)
	}
	if !shouldSkipDir(root, git) {
		t.Fatal("expected skip git")
	}
}
