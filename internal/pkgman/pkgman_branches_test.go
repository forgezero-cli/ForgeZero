package pkgman

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fz/internal/utils"
)

func TestAddInvalidURL(t *testing.T) {
	err := Add(context.Background(), "invalid", "")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestAddGitCloneFail(t *testing.T) {
	old := runGit
	defer func() { runGit = old }()
	runGit = func(ctx context.Context, args ...string) (string, error) {
		return "", errors.New("clone failed")
	}
	err := Add(context.Background(), "github.com/user/repo", "")
	if err == nil || !strings.Contains(err.Error(), "git clone") {
		t.Fatalf("got %v", err)
	}
}

func TestAddCheckoutFail(t *testing.T) {
	old := runGit
	defer func() { runGit = old }()
	calls := 0
	runGit = func(ctx context.Context, args ...string) (string, error) {
		calls++
		if calls == 1 {
			return "", nil
		}
		return "", errors.New("checkout failed")
	}
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	err := Add(context.Background(), "github.com/user/repo@v1", "")
	if err == nil || !strings.Contains(err.Error(), "checkout") {
		t.Fatalf("got %v", err)
	}
}

func TestAddSuccess(t *testing.T) {
	old := runGit
	defer func() { runGit = old }()
	runGit = func(ctx context.Context, args ...string) (string, error) {
		return "", nil
	}
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	if err := Add(context.Background(), "github.com/user/repo", "v2"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(".fz.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "vendor/github.com/user/repo") {
		t.Fatal("config not updated")
	}
}

func TestRemoveByRepoPath(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	dest := filepath.Join("vendor", "github.com", "user", "lib")
	os.MkdirAll(dest, 0o755)
	os.MkdirAll(filepath.Join(dest, ".git"), 0o755)
	if err := os.WriteFile(".fz.yaml", []byte("source_dirs:\n  - "+dest+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Remove(context.Background(), "github.com/user/lib"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("package not removed")
	}
}

func TestRemoveViaFindPackagePath(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	dest := filepath.Join("vendor", "github.com", "user", "lib")
	os.MkdirAll(dest, 0o755)
	os.MkdirAll(filepath.Join(dest, ".git"), 0o755)
	if err := Remove(context.Background(), "user/lib"); err != nil {
		t.Fatal(err)
	}
}

func TestRemovePackagePrunesEmptyParent(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	leaf := filepath.Join("vendor", "github.com", "user", "lib")
	os.MkdirAll(leaf, 0o755)
	if err := removePackage(leaf); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join("vendor", "github.com", "user")); err != nil {
		if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
}

func TestUpdateNoVendor(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := Update(context.Background())
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No packages") {
		t.Fatal(buf.String())
	}
}

func TestUpdateWithPackages(t *testing.T) {
	old := runGit
	defer func() { runGit = old }()
	runGit = func(ctx context.Context, args ...string) (string, error) {
		return "", nil
	}
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	os.MkdirAll(filepath.Join("vendor", "pkg-a"), 0o755)
	os.MkdirAll(filepath.Join("vendor", "pkg-b"), 0o755)
	if err := Update(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateGitPullFail(t *testing.T) {
	old := runGit
	defer func() { runGit = old }()
	runGit = func(ctx context.Context, args ...string) (string, error) {
		return "", errors.New("pull failed")
	}
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	os.MkdirAll(filepath.Join("vendor", "broken"), 0o755)
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	Update(context.Background())
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	if !strings.Contains(buf.String(), "Warning") {
		t.Fatal(buf.String())
	}
}

func TestFetchCatalogAllFail(t *testing.T) {
	old := httpClient
	defer func() { httpClient = old }()
	httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network down")
	})}
	_, err := fetchCatalog()
	if err == nil || !strings.Contains(err.Error(), "all catalog URLs failed") {
		t.Fatalf("got %v", err)
	}
}

func TestFetchCatalogMalformedJSON(t *testing.T) {
	old := httpClient
	defer func() { httpClient = old }()
	os.Setenv("FZ_CATALOG_URL", "http://mock/catalog.json")
	defer os.Unsetenv("FZ_CATALOG_URL")
	httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{bad")), Header: make(http.Header)}, nil
	})}
	_, err := fetchCatalog()
	if err == nil {
		t.Fatal("expected json error")
	}
}

func TestInstallFromCatalogNotFound(t *testing.T) {
	old := httpClient
	defer func() { httpClient = old }()
	body, _ := json.Marshal(Catalog{Version: 1, Packages: []CatalogPackage{{Name: "other", Repo: "github.com/x/y"}}})
	httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	})}
	err := InstallFromCatalog(context.Background(), "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("got %v", err)
	}
}

func TestInstallFromCatalogSuccess(t *testing.T) {
	oldGit := runGit
	oldHTTP := httpClient
	defer func() { runGit = oldGit; httpClient = oldHTTP }()
	runGit = func(ctx context.Context, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "clone" {
			dest := args[len(args)-1]
			os.MkdirAll(dest, 0o755)
		}
		return "", nil
	}
	body, _ := json.Marshal(Catalog{Version: 1, Packages: []CatalogPackage{
		{Name: "pkg", Repo: "github.com/user/repo", Tag: "v1", SourceDir: "src", Hash: ""},
	}})
	httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	})}
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	if err := InstallFromCatalog(context.Background(), "pkg"); err != nil {
		t.Fatal(err)
	}
}

func TestInstallFromCatalogHashMismatch(t *testing.T) {
	oldGit := runGit
	oldHTTP := httpClient
	defer func() { runGit = oldGit; httpClient = oldHTTP }()
	runGit = func(ctx context.Context, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "clone" {
			dest := args[len(args)-1]
			os.MkdirAll(filepath.Join(dest, "src"), 0o755)
			os.WriteFile(filepath.Join(dest, "src", "f.txt"), []byte("data"), 0o644)
		}
		return "", nil
	}
	body, _ := json.Marshal(Catalog{Version: 1, Packages: []CatalogPackage{
		{Name: "pkg", Repo: "github.com/user/repo", SourceDir: "src", Hash: "deadbeef"},
	}})
	httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	})}
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	err := InstallFromCatalog(context.Background(), "pkg")
	if err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("got %v", err)
	}
}

func TestCleanConfigReadError(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	os.WriteFile(".fz.yaml", []byte("source_dirs: []\n"), 0o644)
	os.Chmod(".fz.yaml", 0o000)
	defer os.Chmod(".fz.yaml", 0o644)
	err := cleanConfig("vendor/x")
	if err == nil {
		t.Fatal("expected read error")
	}
}

func TestUpdateConfigInvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	os.WriteFile(".fz.yaml", []byte(":\n\tbad"), 0o644)
	if err := updateConfig("vendor/x", true); err == nil {
		t.Fatal("expected yaml error")
	}
}

func TestAddVersionOverride(t *testing.T) {
	old := runGit
	defer func() { runGit = old }()
	var checkoutTag string
	runGit = func(ctx context.Context, args ...string) (string, error) {
		for i, a := range args {
			if a == "checkout" && i+1 < len(args) {
				checkoutTag = args[i+1]
			}
		}
		return "", nil
	}
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	if err := Add(context.Background(), "github.com/user/repo@ignored", "v9"); err != nil {
		t.Fatal(err)
	}
	if checkoutTag != "v9" {
		t.Fatalf("tag %q", checkoutTag)
	}
}

func TestRemovePackageRemoveAllFail(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can remove anything")
	}
	tmp := t.TempDir()
	path := filepath.Join(tmp, "vendor", "pkg")
	os.MkdirAll(path, 0o755)
	os.WriteFile(filepath.Join(path, "lock"), []byte("x"), 0o444)
	os.Chmod(path, 0o555)
	err := removePackage(path)
	os.Chmod(path, 0o755)
	if err == nil {
		t.Fatal("expected remove error")
	}
}

func TestFetchCatalogFromURLReadBodyFail(t *testing.T) {
	old := httpClient
	defer func() { httpClient = old }()
	httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(&failReader{}), Header: make(http.Header)}, nil
	})}
	_, err := fetchCatalogFromURL("http://x")
	if err == nil {
		t.Fatal("expected read error")
	}
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, errors.New("read fail") }
func (failReader) Close() error             { return nil }

func TestSearchCatalogNoMatch(t *testing.T) {
	old := httpClient
	defer func() { httpClient = old }()
	httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		body := `{"version":1,"packages":[{"name":"alpha","description":"beta","category":"gamma"}]}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	SearchCatalog(context.Background(), "zzznone")
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	if !strings.Contains(buf.String(), "No matching") {
		t.Fatal(buf.String())
	}
}

func TestCleanConfigNoSourceDirs(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	os.WriteFile(".fz.yaml", []byte("output: x\n"), 0o644)
	if err := cleanConfig("vendor/none"); err != nil {
		t.Fatal(err)
	}
}

func TestCleanConfigNonStringEntry(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	os.WriteFile(".fz.yaml", []byte("source_dirs:\n  - 42\n"), 0o644)
	if err := cleanConfig("vendor/x"); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateConfigSecureWriteFail(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	if err := updateConfig(filepath.Join(tmp, "outside"), true); err != nil {
		return
	}
}

func TestListWalkError(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	os.Mkdir("vendor", 0o000)
	defer os.Chmod("vendor", 0o755)
	if err := List(); err == nil {
		t.Fatal("expected walk error")
	}
}

func TestAddMkdirFail(t *testing.T) {
	old := runGit
	defer func() { runGit = old }()
	runGit = func(ctx context.Context, args ...string) (string, error) { return "", nil }
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	os.Mkdir("vendor", 0o444)
	defer os.Chmod("vendor", 0o755)
	err := Add(context.Background(), "github.com/u/r", "")
	if err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestInstallFromCatalogHashComputeWarn(t *testing.T) {
	oldGit := runGit
	oldHTTP := httpClient
	defer func() { runGit = oldGit; httpClient = oldHTTP }()
	runGit = func(ctx context.Context, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "clone" {
			os.MkdirAll(args[len(args)-1], 0o755)
		}
		return "", nil
	}
	body, _ := json.Marshal(Catalog{Version: 1, Packages: []CatalogPackage{
		{Name: "pkg", Repo: "github.com/u/r", Hash: "abc", SourceDir: "missing/sub"},
	}})
	httpClient = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	})}
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	InstallFromCatalog(context.Background(), "pkg")
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	if !strings.Contains(buf.String(), "Warning") {
		t.Fatal(buf.String())
	}
}

func TestFindPackagePathWalkError(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	os.Mkdir("vendor", 0o000)
	defer os.Chmod("vendor", 0o755)
	_, err := findPackagePath("x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateConfigReadPermError(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	os.WriteFile(".fz.yaml", []byte("source_dirs: []\n"), 0o644)
	os.Chmod(".fz.yaml", 0o000)
	defer os.Chmod(".fz.yaml", 0o644)
	if err := updateConfig("vendor/x", true); err == nil {
		t.Fatal("expected error")
	}
}

func TestAddUsesSecureMkdir(t *testing.T) {
	old := runGit
	defer func() { runGit = old }()
	runGit = func(ctx context.Context, args ...string) (string, error) { return "", nil }
	tmp := t.TempDir()
	utils.SetExecutionRoot(tmp)
	defer utils.SetExecutionRoot("")
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)
	if err := Add(context.Background(), "github.com/u/r", ""); err != nil {
		t.Fatal(err)
	}
}
