package sbom

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"fz/internal/config"
	"fz/internal/seal"
	"fz/internal/utils"
)

type SBOM struct {
	BomFormat   string      `json:"bomFormat"`
	SpecVersion string      `json:"specVersion"`
	Version     int         `json:"version"`
	Metadata    Metadata    `json:"metadata"`
	Components  []Component `json:"components,omitempty"`
}

type Metadata struct {
	Timestamp  string     `json:"timestamp"`
	Tools      []Tool     `json:"tools,omitempty"`
	Component  Component  `json:"component"`
	Properties []Property `json:"properties,omitempty"`
}

type Tool struct {
	Vendor  string `json:"vendor,omitempty"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type Component struct {
	Type       string     `json:"type,omitempty"`
	Name       string     `json:"name"`
	Version    string     `json:"version,omitempty"`
	Hashes     []Hash     `json:"hashes,omitempty"`
	Properties []Property `json:"properties,omitempty"`
}

type Hash struct {
	Algorithm string `json:"alg,omitempty"`
	Content   string `json:"content,omitempty"`
}

type Property struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

func Generate(root, vendorDir, buildVersion string, cfg *config.Config, target string) (*SBOM, error) {
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getwd: %w", err)
		}
		root = cwd
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("abs root: %w", err)
	}
	if err := utils.EnsureInsideRoot(rootAbs, rootAbs); err != nil {
		return nil, err
	}
	if vendorDir == "" {
		vendorDir = "vendor"
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	utils.SetExecutionRoot(rootAbs)
	tools := detectToolchainVersions(target)
	components, err := scanVendorComponents(rootAbs, vendorDir)
	if err != nil {
		return nil, err
	}
	metadata := Metadata{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Tools:     tools,
		Component: Component{Type: "application", Name: "fz", Version: buildVersion},
		Properties: []Property{
			{Name: "os", Value: runtime.GOOS},
			{Name: "arch", Value: runtime.GOARCH},
			{Name: "target", Value: target},
			{Name: "vendor_dir", Value: filepath.Clean(vendorDir)},
		},
	}
	if len(cfg.ToolChecksums) > 0 {
		keys := make([]string, 0, len(cfg.ToolChecksums))
		for k := range cfg.ToolChecksums {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, name := range keys {
			metadata.Properties = append(metadata.Properties, Property{
				Name:  fmt.Sprintf("tool.checksum.%s", name),
				Value: cfg.ToolChecksums[name],
			})
		}
	}
	return &SBOM{
		BomFormat:   "CycloneDX",
		SpecVersion: "1.4",
		Version:     1,
		Metadata:    metadata,
		Components:  components,
	}, nil
}

func Marshal(sbom *SBOM) ([]byte, error) {
	return json.MarshalIndent(sbom, "", "  ")
}

func ExportEncryptedSBOM(root, vendorDir, buildVersion string, cfg *config.Config, target string) ([]byte, error) {
	sbomDoc, err := Generate(root, vendorDir, buildVersion, cfg, target)
	if err != nil {
		return nil, err
	}
	plain, err := Marshal(sbomDoc)
	if err != nil {
		return nil, err
	}
	mid, err := seal.MachineID()
	if err != nil || mid == "" {
		mid = "forgezero"
	}
	key := []byte(mid)
	out := make([]byte, len(plain))
	for i := range plain {
		out[i] = plain[i] ^ key[i%len(key)]
	}
	return out, nil
}

func scanVendorComponents(root, vendorDir string) ([]Component, error) {
	rootAbs := filepath.Clean(root)
	vendorPath := filepath.Join(root, vendorDir)
	if err := utils.EnsureInsideRoot(rootAbs, vendorPath); err != nil {
		return nil, fmt.Errorf("vendor path: %w", err)
	}
	resolvedVendor, err := utils.ResolveSecurePath(vendorPath)
	if err != nil {
		return nil, fmt.Errorf("resolve vendor: %w", err)
	}
	info, err := utils.StatResolved(resolvedVendor)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat vendor: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("vendor path is not a directory: %s", vendorPath)
	}
	entries, err := utils.ReadDirResolved(resolvedVendor)
	if err != nil {
		return nil, fmt.Errorf("read vendor: %w", err)
	}
	components := make([]Component, 0, len(entries))
	for _, entry := range entries {
		path := filepath.Join(resolvedVendor, entry.Name())
		hash, err := hashVendorEntry(rootAbs, path, entry)
		if err != nil {
			return nil, err
		}
		components = append(components, Component{
			Type:       "library",
			Name:       entry.Name(),
			Hashes:     []Hash{{Algorithm: "BLAKE3", Content: hash}},
			Properties: []Property{{Name: "path", Value: filepath.ToSlash(strings.TrimPrefix(path, rootAbs+string(filepath.Separator)))}},
		})
	}
	sort.Slice(components, func(i, j int) bool { return components[i].Name < components[j].Name })
	return components, nil
}

func hashVendorEntry(rootAbs, path string, entry os.DirEntry) (string, error) {
	info, lerr := utils.LstatPath(path)
	if lerr != nil {
		return "", fmt.Errorf("lstat %s: %w", path, lerr)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		resolved, rerr := utils.EvalSymlinksPath(path)
		if rerr != nil {
			return "", fmt.Errorf("eval symlink %s: %w", path, rerr)
		}
		if err := utils.EnsureInsideRoot(rootAbs, resolved); err != nil {
			fmt.Fprintf(os.Stderr, "SECURITY WARNING: vendor symlink %s outside project root %s\n", path, rootAbs)
			return utils.HashDirWithRoot(rootAbs, path)
		}
		st, serr := utils.StatResolved(resolved)
		if serr != nil {
			return "", fmt.Errorf("stat %s: %w", resolved, serr)
		}
		if st.IsDir() {
			return utils.HashDirWithRoot(rootAbs, resolved)
		}
		return utils.HashFile(resolved)
	}
	if entry.IsDir() {
		return utils.HashDirWithRoot(rootAbs, path)
	}
	return utils.HashFile(path)
}

func detectToolchainVersions(target string) []Tool {
	if target == "" {
		target = "x86_64-linux-gnu"
	}
	candidates := []struct {
		name string
		args []string
	}{
		{name: "gcc", args: []string{"--version"}},
		{name: "clang", args: []string{"--version"}},
		{name: "emcc", args: []string{"--version"}},
		{name: "nasm", args: []string{"-v"}},
		{name: "wasm-ld", args: []string{"--version"}},
	}
	var tools []Tool
	for _, c := range candidates {
		version, ok := queryToolVersion(c.name, c.args...)
		if !ok {
			continue
		}
		tools = append(tools, Tool{Vendor: "GNU", Name: c.name, Version: version})
	}
	if strings.Contains(target, "wasm") || strings.Contains(target, "wasm32") {
		tools = append(tools, Tool{Vendor: "WebAssembly", Name: "wasm-target", Version: target})
	}
	return tools
}

func queryToolVersion(name string, args ...string) (string, bool) {
	if _, err := exec.LookPath(name); err != nil {
		return "", false
	}
	output, err := utils.RunCommandSilent(context.Background(), false, name, args...)
	if err != nil {
		return strings.TrimSpace(output), true
	}
	output = strings.TrimSpace(output)
	if output == "" {
		return "unknown", true
	}
	lines := strings.Split(output, "\n")
	return strings.TrimSpace(lines[0]), true
}
