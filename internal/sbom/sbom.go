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
			return nil, err
		}
		root = cwd
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
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
		var keys []string
		for k := range cfg.ToolChecksums {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, name := range keys {
			metadata.Properties = append(metadata.Properties, Property{Name: fmt.Sprintf("tool.checksum.%s", name), Value: cfg.ToolChecksums[name]})
		}
	}
	sbom := &SBOM{
		BomFormat:   "CycloneDX",
		SpecVersion: "1.4",
		Version:     1,
		Metadata:    metadata,
		Components:  components,
	}
	return sbom, nil
}

func Marshal(sbom *SBOM) ([]byte, error) {
	return json.MarshalIndent(sbom, "", "  ")
}

func scanVendorComponents(root, vendorDir string) ([]Component, error) {
	rootAbs := filepath.Clean(root)
	vendorPath := filepath.Join(root, vendorDir)
	info, err := os.Stat(vendorPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("vendor path is not a directory: %s", vendorPath)
	}
	entries, err := os.ReadDir(vendorPath)
	if err != nil {
		return nil, err
	}
	var components []Component
	for _, entry := range entries {
		path := filepath.Join(vendorPath, entry.Name())
		var hash string

		if info, lerr := os.Lstat(path); lerr == nil && info.Mode()&os.ModeSymlink != 0 {
			resolved, rerr := filepath.EvalSymlinks(path)
			if rerr != nil {
				return nil, rerr
			}
			if st, serr := os.Stat(resolved); serr == nil && st.IsDir() {
				hash, err = utils.HashDirWithRoot(rootAbs, resolved)
			} else {
				hash, err = utils.HashFile(resolved)
			}
		} else if entry.IsDir() {
			hash, err = utils.HashDirWithRoot(rootAbs, path)
		} else {
			hash, err = utils.HashFile(path)
		}

		if err != nil {
			return nil, err
		}
		component := Component{
			Type:       "library",
			Name:       entry.Name(),
			Version:    "",
			Hashes:     []Hash{{Algorithm: "BLAKE3", Content: hash}},
			Properties: []Property{{Name: "path", Value: filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator)))}},
		}
		components = append(components, component)
	}
	sort.Slice(components, func(i, j int) bool { return components[i].Name < components[j].Name })
	return components, nil
}

func detectToolchainVersions(target string) []Tool {
	tools := []Tool{}
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
	for _, candidate := range candidates {
		version, ok := queryToolVersion(candidate.name, candidate.args...)
		if !ok {
			continue
		}
		tools = append(tools, Tool{Vendor: "GNU", Name: candidate.name, Version: version})
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
