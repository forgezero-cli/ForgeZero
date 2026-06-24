// SPDX-LICENSE-INDITIFIER MIT

package audit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

const (
	SeverityHigh   = "HIGH"
	SeverityMedium = "MEDIUM"
	SeverityLow    = "LOW"
)

type Finding struct {
	Package  string `json:"package"`
	Summary  string `json:"summary"`
	Path     string `json:"path"`
	Severity string `json:"severity"`
	Version  string `json:"version,omitempty"`
	URL      string `json:"url,omitempty"`
}

type Result struct {
	Findings []Finding `json:"findings"`
}

func (r *Result) HasHighSeverity() bool {
	for _, finding := range r.Findings {
		if finding.Severity == SeverityHigh {
			return true
		}
	}
	return false
}

type auditRule struct {
	Name     string
	Keywords []string
	Summary  string
	URL      string
}

var knownVulnerabilities = []auditRule{
	{
		Name:     "OpenSSL",
		Keywords: []string{"openssl", "libssl"},
		Summary:  "OpenSSL has frequent security advisories; verify the version and patches.",
		URL:      "https://www.openssl.org/news/vulnerabilities.html",
	},
	{
		Name:     "cURL",
		Keywords: []string{"curl", "libcurl"},
		Summary:  "cURL / libcurl may contain remote code execution or buffer overflow vulnerabilities.",
		URL:      "https://curl.se/docs/knownbugs.html",
	},
	{
		Name:     "glibc",
		Keywords: []string{"glibc", "gnu libc"},
		Summary:  "glibc vulnerabilities can lead to privilege escalation and remote code execution.",
		URL:      "https://www.gnu.org/software/libc/",
	},
	{
		Name:     "zlib",
		Keywords: []string{"zlib"},
		Summary:  "zlib vulnerabilities are common in decompression routines.",
		URL:      "https://zlib.net/",
	},
	{
		Name:     "SQLite",
		Keywords: []string{"sqlite"},
		Summary:  "SQLite may include advisories for injection or memory corruption.",
		URL:      "https://www.sqlite.org/",
	},
	{
		Name:     "libpng",
		Keywords: []string{"libpng", "png"},
		Summary:  "libpng has had multiple high-risk vulnerabilities.",
		URL:      "https://libpng.sourceforge.io/",
	},
	{
		Name:     "Bash",
		Keywords: []string{"bash"},
		Summary:  "Bash shell vulnerabilities can allow code execution in scripts.",
		URL:      "https://www.gnu.org/software/bash/",
	},
}

var configRiskKeywords = []string{
	"curl",
	"wget",
	"http://",
	"https://",
	"shell",
	"remote",
	"git@",
	"ssh://",
}

type secretRule struct {
	ID      string
	Pattern *regexp.Regexp
	Summary string
}

type licenseRule struct {
	Name     string
	Keywords []string
	Summary  string
	URL      string
}

var secretRules = []secretRule{
	{
		ID:      "aws-secret-access-key",
		Pattern: regexp.MustCompile(`(?i)aws[_-]?secret[_-]?access[_-]?key\s*[:=]\s*[\"']?[A-Za-z0-9/+=]{40,}`),
		Summary: "Hardcoded AWS secret access key found.",
	},
	{
		ID:      "aws-access-key-id",
		Pattern: regexp.MustCompile(`(?i)aws[_-]?access[_-]?key(?:_id)?\s*[:=]\s*[\"']?[A-Z0-9]{16,}`),
		Summary: "Hardcoded AWS access key ID found.",
	},
	{
		ID:      "api-key",
		Pattern: regexp.MustCompile(`(?i)(?:api|secret|token|password|passphrase|client[_-]?secret)[\s]*[:=][\s]*[\"']?[A-Za-z0-9-_]{16,}`),
		Summary: "Hardcoded API key or secret token detected.",
	},
	{
		ID:      "private-key-block",
		Pattern: regexp.MustCompile(`(?i)-----BEGIN (?:RSA |EC |DSA )?PRIVATE KEY-----`),
		Summary: "Private key block found in source file.",
	},
	{
		ID:      "ssh-key",
		Pattern: regexp.MustCompile(`(?i)ssh-(?:rsa|ed25519|dss)\s+[A-Za-z0-9+/=]{100,}`),
		Summary: "SSH public or private key material found.",
	},
}

var licenseRules = []licenseRule{
	{Name: "AGPL", Keywords: []string{"agpl"}, Summary: "AGPL license detected - incompatible with many proprietary distribution models.", URL: "https://www.gnu.org/licenses/agpl-3.0.html"},
	{Name: "GPL", Keywords: []string{"gnu general public license", "gpl"}, Summary: "GPL license detected - may impose strong copyleft obligations.", URL: "https://www.gnu.org/licenses/gpl-3.0.html"},
	{Name: "LGPL", Keywords: []string{"lgpl"}, Summary: "LGPL license detected - may require library compliance when linking.", URL: "https://www.gnu.org/licenses/lgpl-3.0.html"},
	{Name: "MPL", Keywords: []string{"mozilla public license", "mpl"}, Summary: "MPL license detected - source disclosure may be required for modifications.", URL: "https://www.mozilla.org/en-US/MPL/"},
	{Name: "EPL", Keywords: []string{"eclipse public license", "epl"}, Summary: "EPL license detected - review terms for distribution compatibility.", URL: "https://www.eclipse.org/legal/epl-2.0/"},
	{Name: "Proprietary", Keywords: []string{"all rights reserved", "proprietary"}, Summary: "Proprietary license terms detected in vendor package.", URL: "https://en.wikipedia.org/wiki/Software_license#Proprietary_licenses"},
}

var licenseFileNames = map[string]bool{
	"license":     true,
	"license.txt": true,
	"license.md":  true,
	"copying":     true,
	"copying.txt": true,
	"copying.md":  true,
}

func ScanProject(ctx context.Context, root, vendorDir string, cfg *config.Config) (*Result, error) {
	if root == "" {
		return nil, errors.New("project root is required")
	}
	if err := utils.EnsureInsideRoot(root, root); err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = &config.Config{}
	}

	findings := []Finding{}
	seen := map[string]bool{}

	vendorPath := filepath.Join(root, vendorDir)
	if err := scanVendor(ctx, root, vendorPath, cfg, &findings, seen); err != nil {
		return nil, err
	}
	if err := scanVendorLicenses(ctx, vendorPath, cfg, &findings, seen); err != nil {
		return nil, err
	}
	if err := scanSecrets(ctx, root, cfg, &findings, seen); err != nil {
		return nil, err
	}
	if err := scanConfigFiles(root, cfg, &findings, seen); err != nil {
		return nil, err
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Package == findings[j].Package {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Package < findings[j].Package
	})

	return &Result{Findings: findings}, nil
}

func scanVendor(ctx context.Context, root, vendorPath string, cfg *config.Config, findings *[]Finding, seen map[string]bool) error {
	info, err := os.Stat(vendorPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return errors.New("vendor path is not a directory: " + vendorPath)
	}

	return filepath.WalkDir(vendorPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".git") {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if isIgnored(cfg, rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		pathLower := strings.ToLower(rel)
		for _, rule := range knownVulnerabilities {
			if matchesAnyKeyword(pathLower, rule.Keywords) {
				key := rule.Name + ":" + rel
				if seen[key] {
					continue
				}
				*findings = append(*findings, Finding{
					Package:  rule.Name,
					Summary:  rule.Summary,
					Path:     rel,
					Severity: SeverityHigh,
					URL:      rule.URL,
				})
				seen[key] = true
			}
		}
		if strings.HasSuffix(pathLower, "go.mod") || strings.HasSuffix(pathLower, "package.json") || strings.HasSuffix(pathLower, "requirements.txt") || strings.HasSuffix(pathLower, "pyproject.toml") {
			if err := scanFileContent(path, findings, seen); err != nil {
				return err
			}
		}
		return nil
	})
}

func scanFileContent(path string, findings *[]Finding, seen map[string]bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := strings.ToLower(string(data))
	for _, rule := range knownVulnerabilities {
		if matchesAnyKeyword(content, rule.Keywords) {
			key := rule.Name + ":" + path
			if seen[key] {
				continue
			}
			*findings = append(*findings, Finding{
				Package:  rule.Name,
				Summary:  rule.Summary,
				Path:     path,
				Severity: SeverityHigh,
				URL:      rule.URL,
			})
			seen[key] = true
		}
	}
	return nil
}

func scanConfigFiles(root string, cfg *config.Config, findings *[]Finding, seen map[string]bool) error {
	paths := []string{".fz.yaml", "fz.yaml", ".fz.yml", "fz.yml"}
	for _, p := range paths {
		configPath := filepath.Join(root, p)
		if _, err := os.Stat(configPath); err != nil {
			continue
		}
		if isIgnored(cfg, p) {
			continue
		}
		data, err := os.ReadFile(configPath)
		if err != nil {
			return err
		}
		content := strings.ToLower(string(data))
		if matchesAnyKeyword(content, configRiskKeywords) {
			key := "config:" + p
			if !seen[key] {
				*findings = append(*findings, Finding{
					Package:  "Configuration",
					Summary:  "Potential risky configuration or remote fetch usage found in " + p,
					Path:     p,
					Severity: SeverityHigh,
					URL:      "https://en.wikipedia.org/wiki/Secure_coding",
				})
				seen[key] = true
			}
		}
	}
	return nil
}

func matchesAnyKeyword(text string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

func scanVendorLicenses(ctx context.Context, vendorPath string, cfg *config.Config, findings *[]Finding, seen map[string]bool) error {
	if vendorPath == "" {
		return nil
	}
	if _, err := os.Stat(vendorPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return filepath.WalkDir(vendorPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			return nil
		}
		name := strings.ToLower(filepath.Base(path))
		if !licenseFileNames[name] {
			return nil
		}
		rel, err := filepath.Rel(vendorPath, path)
		if err != nil {
			rel = path
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := strings.ToLower(string(data))
		for _, rule := range licenseRules {
			if matchesAnyKeyword(content, rule.Keywords) {
				key := "license:" + rule.Name + ":" + rel
				if seen[key] {
					continue
				}
				*findings = append(*findings, Finding{
					Package:  "License",
					Summary:  rule.Summary,
					Path:     filepath.ToSlash(path),
					Severity: SeverityMedium,
					URL:      rule.URL,
				})
				seen[key] = true
			}
		}
		return nil
	})
}

func scanSecrets(ctx context.Context, root string, cfg *config.Config, findings *[]Finding, seen map[string]bool) error {
	extensions := map[string]bool{
		".c": true, ".cpp": true, ".cc": true, ".cxx": true, ".h": true, ".hpp": true, ".hxx": true,
		".s": true, ".S": true, ".asm": true, ".fasm": true,
		".yaml": true, ".yml": true, ".json": true, ".env": true, ".toml": true, ".ini": true, ".sh": true, ".ps1": true, ".md": true, ".txt": true,
	}
	fileNames := map[string]bool{"dockerfile": true, "docker-compose.yml": true, "docker-compose.yaml": true}
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if name == ".git" || name == ".fz_cache" || name == ".fz_objs" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if isIgnored(cfg, rel) {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !extensions[ext] && !fileNames[strings.ToLower(filepath.Base(path))] {
			return nil
		}
		return scanSecretFile(path, findings, seen)
	})
}

func scanSecretFile(path string, findings *[]Finding, seen map[string]bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	for _, rule := range secretRules {
		if rule.Pattern.MatchString(content) {
			key := "secret:" + rule.ID + ":" + path
			if seen[key] {
				continue
			}
			*findings = append(*findings, Finding{
				Package:  "HardcodedSecret",
				Summary:  rule.Summary,
				Path:     filepath.ToSlash(path),
				Severity: SeverityHigh,
			})
			seen[key] = true
		}
	}
	return nil
}

func isIgnored(cfg *config.Config, path string) bool {
	if cfg == nil || len(cfg.AuditIgnore) == 0 {
		return false
	}
	pathLower := strings.ToLower(path)
	for _, ignore := range cfg.AuditIgnore {
		ignore = strings.TrimSpace(strings.ToLower(ignore))
		if ignore == "" {
			continue
		}
		if strings.Contains(pathLower, ignore) {
			return true
		}
	}
	return false
}
