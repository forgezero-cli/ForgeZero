/*
 * Copyright (c) 2026 ForgeZero-cli
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package audit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/drivers/workerpool"
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
	{Name: "OpenSSL", Keywords: []string{"openssl", "libssl"}, Summary: "OpenSSL has frequent security advisories; verify the version and patches.", URL: "https://www.openssl.org/news/vulnerabilities.html"},
	{Name: "cURL", Keywords: []string{"curl", "libcurl"}, Summary: "cURL / libcurl may contain remote code execution or buffer overflow vulnerabilities.", URL: "https://curl.se/docs/knownbugs.html"},
	{Name: "glibc", Keywords: []string{"glibc", "gnu libc"}, Summary: "glibc vulnerabilities can lead to privilege escalation and remote code execution.", URL: "https://www.gnu.org/software/libc/"},
	{Name: "zlib", Keywords: []string{"zlib"}, Summary: "zlib vulnerabilities are common in decompression routines.", URL: "https://zlib.net/"},
	{Name: "SQLite", Keywords: []string{"sqlite"}, Summary: "SQLite may include advisories for injection or memory corruption.", URL: "https://www.sqlite.org/"},
	{Name: "libpng", Keywords: []string{"libpng", "png"}, Summary: "libpng has had multiple high-risk vulnerabilities.", URL: "https://libpng.sourceforge.io/"},
	{Name: "Bash", Keywords: []string{"bash"}, Summary: "Bash shell vulnerabilities can allow code execution in scripts.", URL: "https://www.gnu.org/software/bash/"},
	{Name: "libjpeg", Keywords: []string{"libjpeg", "jpeg", "jpeglib"}, Summary: "libjpeg has had numerous buffer overflow and memory corruption issues.", URL: "https://libjpeg.sourceforge.net/"},
	{Name: "libxml2", Keywords: []string{"libxml2", "xml"}, Summary: "libxml2 has a history of XXE and DoS vulnerabilities.", URL: "http://xmlsoft.org/"},
	{Name: "Expat", Keywords: []string{"expat", "xmlparse"}, Summary: "Expat parser has had multiple CVEs for DoS and memory issues.", URL: "https://libexpat.github.io/"},
	{Name: "libtiff", Keywords: []string{"libtiff", "tiff"}, Summary: "libtiff vulnerabilities can lead to denial of service or code execution.", URL: "https://www.libtiff.org/"},
	{Name: "libvorbis", Keywords: []string{"libvorbis", "vorbis"}, Summary: "libvorbis has had several CVEs for memory corruption.", URL: "https://www.xiph.org/vorbis/"},
	{Name: "libtheora", Keywords: []string{"libtheora", "theora"}, Summary: "libtheora may contain buffer overflow vulnerabilities.", URL: "https://www.theora.org/"},
	{Name: "FFmpeg", Keywords: []string{"ffmpeg", "libavcodec", "libavformat"}, Summary: "FFmpeg has a large attack surface with many CVEs.", URL: "https://ffmpeg.org/"},
	{Name: "ImageMagick", Keywords: []string{"imagemagick", "magick", "convert"}, Summary: "ImageMagick has had many vulnerabilities including RCE and DoS.", URL: "https://imagemagick.org/script/security.php"},
	{Name: "Ghostscript", Keywords: []string{"ghostscript", "gs"}, Summary: "Ghostscript vulnerabilities are commonly exploited for RCE.", URL: "https://www.ghostscript.com/"},
	{Name: "OpenJPEG", Keywords: []string{"openjpeg", "jpeg2000"}, Summary: "OpenJPEG has several heap buffer overflow CVEs.", URL: "https://www.openjpeg.org/"},
	{Name: "libwebp", Keywords: []string{"libwebp", "webp"}, Summary: "libwebp had a severe heap overflow CVE-2023-4863.", URL: "https://developers.google.com/speed/webp/"},
	{Name: "libavif", Keywords: []string{"libavif"}, Summary: "libavif may have memory corruption issues in AV1 decoding.", URL: "https://aomediacodec.github.io/av1-spec/"},
	{Name: "libvpx", Keywords: []string{"libvpx", "vpx"}, Summary: "libvpx has had several CVEs for VP8/VP9 decoding.", URL: "https://www.webmproject.org/"},
	{Name: "LZO", Keywords: []string{"lzo"}, Summary: "LZO compression library has had memory corruption CVEs.", URL: "https://www.oberhumer.com/opensource/lzo/"},
	{Name: "lz4", Keywords: []string{"lz4"}, Summary: "lz4 had a heap overflow CVE-2021-3520.", URL: "https://lz4.github.io/lz4/"},
	{Name: "zstd", Keywords: []string{"zstd"}, Summary: "zstd compression library has had some CVEs for buffer overflow.", URL: "https://facebook.github.io/zstd/"},
	{Name: "GNU Make", Keywords: []string{"make", "gnu make"}, Summary: "GNU Make has had privilege escalation and arbitrary code execution CVEs.", URL: "https://www.gnu.org/software/make/"},
	{Name: "GNU tar", Keywords: []string{"tar", "gnu tar"}, Summary: "GNU tar has had path traversal and buffer overflow vulnerabilities.", URL: "https://www.gnu.org/software/tar/"},
	{Name: "GNU sed", Keywords: []string{"sed", "gnu sed"}, Summary: "GNU sed had a heap overflow CVE-2021-3467.", URL: "https://www.gnu.org/software/sed/"},
	{Name: "GNU awk", Keywords: []string{"awk", "gnu awk"}, Summary: "GNU awk had a use-after-free vulnerability CVE-2023-4156.", URL: "https://www.gnu.org/software/gawk/"},
	{Name: "GNU grep", Keywords: []string{"grep", "gnu grep"}, Summary: "GNU grep had a heap overflow CVE-2015-1345.", URL: "https://www.gnu.org/software/grep/"},
	{Name: "GNU findutils", Keywords: []string{"find", "xargs", "gnu find"}, Summary: "findutils had code injection CVEs.", URL: "https://www.gnu.org/software/findutils/"},
	{Name: "libarchive", Keywords: []string{"libarchive", "archive"}, Summary: "libarchive has had multiple CVEs for path traversal and RCE.", URL: "https://www.libarchive.org/"},
	{Name: "libxslt", Keywords: []string{"libxslt"}, Summary: "libxslt had XXE and DoS vulnerabilities.", URL: "http://xmlsoft.org/libxslt/"},
	{Name: "libgcrypt", Keywords: []string{"libgcrypt", "gcrypt"}, Summary: "libgcrypt has had side-channel and memory corruption issues.", URL: "https://gnupg.org/software/libgcrypt/"},
	{Name: "libgpg-error", Keywords: []string{"libgpg-error", "gpg-error"}, Summary: "libgpg-error had a CVE for memory corruption.", URL: "https://gnupg.org/software/libgpg-error/"},
	{Name: "libassuan", Keywords: []string{"libassuan"}, Summary: "libassuan had a CVE for IPC vulnerability.", URL: "https://gnupg.org/software/libassuan/"},
	{Name: "GnuTLS", Keywords: []string{"gnutls"}, Summary: "GnuTLS has had several CVEs for certificate validation and memory corruption.", URL: "https://www.gnutls.org/"},
	{Name: "NSS", Keywords: []string{"nss", "nspr"}, Summary: "NSS and NSPR have had many CVEs for SSL/TLS issues.", URL: "https://developer.mozilla.org/en-US/docs/Mozilla/Projects/NSS"},
	{Name: "libcurl", Keywords: []string{"libcurl"}, Summary: "libcurl vulnerabilities can lead to RCE and information disclosure.", URL: "https://curl.se/docs/knownbugs.html"},
	{Name: "libexpat", Keywords: []string{"libexpat", "expat"}, Summary: "libexpat had a CVE for DoS via malformed XML.", URL: "https://libexpat.github.io/"},
	{Name: "libxml", Keywords: []string{"libxml", "libxml2"}, Summary: "libxml2 has many CVEs for XXE and buffer overflows.", URL: "http://xmlsoft.org/"},
	{Name: "libxslt", Keywords: []string{"libxslt"}, Summary: "libxslt had XXE and denial-of-service issues.", URL: "http://xmlsoft.org/libxslt/"},
	{Name: "ICU", Keywords: []string{"icu", "unicode"}, Summary: "ICU has had several CVEs for buffer overflows and DoS.", URL: "https://icu.unicode.org/"},
	{Name: "PCRE", Keywords: []string{"pcre", "pcre2", "perl regex"}, Summary: "PCRE has had many CVEs for buffer overflows and DoS.", URL: "https://www.pcre.org/"},
	{Name: "GNU readline", Keywords: []string{"readline", "libreadline"}, Summary: "readline had a CVE for memory corruption.", URL: "https://tiswww.case.edu/php/chet/readline/rltop.html"},
	{Name: "ncurses", Keywords: []string{"ncurses"}, Summary: "ncurses had a CVE for stack overflow.", URL: "https://invisible-island.net/ncurses/"},
	{Name: "libtool", Keywords: []string{"libtool"}, Summary: "libtool had a CVE for arbitrary code execution.", URL: "https://www.gnu.org/software/libtool/"},
	{Name: "automake", Keywords: []string{"automake"}, Summary: "automake had a CVE for arbitrary code execution.", URL: "https://www.gnu.org/software/automake/"},
	{Name: "autoconf", Keywords: []string{"autoconf"}, Summary: "autoconf had a CVE for arbitrary code execution.", URL: "https://www.gnu.org/software/autoconf/"},
	{Name: "pkg-config", Keywords: []string{"pkg-config"}, Summary: "pkg-config had a CVE for arbitrary code execution.", URL: "https://www.freedesktop.org/wiki/Software/pkg-config/"},
	{Name: "cmake", Keywords: []string{"cmake"}, Summary: "cmake had a CVE for arbitrary code execution.", URL: "https://cmake.org/"},
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
	{ID: "aws-secret-access-key", Pattern: regexp.MustCompile(`(?i)aws[_-]?secret[_-]?access[_-]?key\s*[:=]\s*[\"']?[A-Za-z0-9/+=]{40,}`), Summary: "Hardcoded AWS secret access key found."},
	{ID: "aws-access-key-id", Pattern: regexp.MustCompile(`(?i)aws[_-]?access[_-]?key(?:_id)?\s*[:=]\s*[\"']?[A-Z0-9]{16,}`), Summary: "Hardcoded AWS access key ID found."},
	{ID: "api-key", Pattern: regexp.MustCompile(`(?i)(?:api|secret|token|password|passphrase|client[_-]?secret)[\s]*[:=][\s]*[\"']?[A-Za-z0-9-_]{16,}`), Summary: "Hardcoded API key or secret token detected."},
	{ID: "private-key-block", Pattern: regexp.MustCompile(`(?i)-----BEGIN (?:RSA |EC |DSA )?PRIVATE KEY-----`), Summary: "Private key block found in source file."},
	{ID: "ssh-key", Pattern: regexp.MustCompile(`(?i)ssh-(?:rsa|ed25519|dss)\s+[A-Za-z0-9+/=]{100,}`), Summary: "SSH public or private key material found."},
	{ID: "github-token", Pattern: regexp.MustCompile(`(?i)gh[pousr]_[A-Za-z0-9_]{36,}`), Summary: "GitHub personal access token or OAuth token found."},
	{ID: "gitlab-token", Pattern: regexp.MustCompile(`(?i)glpat-[A-Za-z0-9-_]{20,}`), Summary: "GitLab personal access token found."},
	{ID: "slack-token", Pattern: regexp.MustCompile(`(?i)xox[baprs]-[0-9a-zA-Z-]{10,}`), Summary: "Slack API token found."},
	{ID: "stripe-secret-key", Pattern: regexp.MustCompile(`(?i)sk_live_[A-Za-z0-9]{24,}`), Summary: "Stripe live secret key found."},
	{ID: "stripe-publishable-key", Pattern: regexp.MustCompile(`(?i)pk_live_[A-Za-z0-9]{24,}`), Summary: "Stripe live publishable key found."},
	{ID: "google-api-key", Pattern: regexp.MustCompile(`(?i)AIza[0-9A-Za-z\-_]{35,}`), Summary: "Google API key found."},
	{ID: "heroku-api-key", Pattern: regexp.MustCompile(`(?i)[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`), Summary: "Heroku API key (UUID) found."},
	{ID: "jwt-token", Pattern: regexp.MustCompile(`(?i)eyJ[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+`), Summary: "JWT token (likely hardcoded) found."},
	{ID: "docker-config", Pattern: regexp.MustCompile(`(?i)"auths":\s*{[^}]*"password":\s*"[^"]+"`), Summary: "Docker config with stored password found."},
	{ID: "npm-token", Pattern: regexp.MustCompile(`(?i)npm_[A-Za-z0-9]{36,}`), Summary: "npm access token found."},
	{ID: "pypi-token", Pattern: regexp.MustCompile(`(?i)pypi-[A-Za-z0-9_-]{30,}`), Summary: "PyPI API token found."},
	{ID: "rubygems-token", Pattern: regexp.MustCompile(`(?i)[0-9a-fA-F]{32,}`), Summary: "RubyGems API key found."},
	{ID: "azure-connection-string", Pattern: regexp.MustCompile(`(?i)DefaultEndpointsProtocol=[^;]+;AccountName=[^;]+;AccountKey=[A-Za-z0-9+/=]+`), Summary: "Azure storage connection string with account key found."},
	{ID: "azure-service-bus", Pattern: regexp.MustCompile(`(?i)Endpoint=sb://[^;]+;SharedAccessKeyName=[^;]+;SharedAccessKey=[^;]+`), Summary: "Azure Service Bus connection string with shared access key found."},
	{ID: "twitter-bearer", Pattern: regexp.MustCompile(`(?i)AAAAAAAAAAAAAAAAAAAA[0-9A-Za-z%]{80,}`), Summary: "Twitter Bearer token found."},
	{ID: "discord-webhook", Pattern: regexp.MustCompile(`(?i)https://discord\.com/api/webhooks/[0-9]+/[A-Za-z0-9_-]+`), Summary: "Discord webhook URL found."},
	{ID: "slack-webhook", Pattern: regexp.MustCompile(`(?i)https://hooks\.slack\.com/services/[A-Z0-9]+/[A-Z0-9]+/[A-Za-z0-9]+`), Summary: "Slack webhook URL found."},
}


var licenseRules = []licenseRule{
	{Name: "AGPL", Keywords: []string{"agpl"}, Summary: "AGPL license detected - incompatible with many proprietary distribution models.", URL: "https://www.gnu.org/licenses/agpl-3.0.html"},
	{Name: "GPL", Keywords: []string{"gnu general public license", "gpl"}, Summary: "GPL license detected - may impose strong copyleft obligations.", URL: "https://www.gnu.org/licenses/gpl-3.0.html"},
	{Name: "LGPL", Keywords: []string{"lgpl"}, Summary: "LGPL license detected - may require library compliance when linking.", URL: "https://www.gnu.org/licenses/lgpl-3.0.html"},
	{Name: "MPL", Keywords: []string{"mozilla public license", "mpl"}, Summary: "MPL license detected - source disclosure may be required for modifications.", URL: "https://www.mozilla.org/en-US/MPL/"},
	{Name: "EPL", Keywords: []string{"eclipse public license", "epl"}, Summary: "EPL license detected - review terms for distribution compatibility.", URL: "https://www.eclipse.org/legal/epl-2.0/"},
	{Name: "Proprietary", Keywords: []string{"all rights reserved", "proprietary"}, Summary: "Proprietary license terms detected in vendor package.", URL: "https://en.wikipedia.org/wiki/Software_license#Proprietary_licenses"},
	{Name: "BSD-3-Clause", Keywords: []string{"bsd 3-clause", "redistribution and use in source and binary forms", "without specific prior written permission"}, Summary: "BSD-3-Clause license - permissive but requires attribution.", URL: "https://opensource.org/licenses/BSD-3-Clause"},
	{Name: "BSD-2-Clause", Keywords: []string{"bsd 2-clause", "redistribution and use in source and binary forms", "this list of conditions and the following disclaimer"}, Summary: "BSD-2-Clause license - permissive with minimal conditions.", URL: "https://opensource.org/licenses/BSD-2-Clause"},
	{Name: "MIT", Keywords: []string{"mit license", "permission is hereby granted", "without restriction"}, Summary: "MIT license - permissive with attribution requirement.", URL: "https://opensource.org/licenses/MIT"},
	{Name: "Apache-2.0", Keywords: []string{"apache license", "version 2.0", "http://www.apache.org/licenses/"}, Summary: "Apache-2.0 license - permissive with patent grant and attribution.", URL: "https://www.apache.org/licenses/LICENSE-2.0"},
	{Name: "ISC", Keywords: []string{"isc license", "permission to use, copy, modify, and/or distribute"}, Summary: "ISC license - permissive, similar to MIT.", URL: "https://opensource.org/licenses/ISC"},
	{Name: "Artistic", Keywords: []string{"artistic license"}, Summary: "Artistic license - used by Perl, may have copyleft terms.", URL: "https://opensource.org/licenses/Artistic-2.0"},
	{Name: "Unlicense", Keywords: []string{"unlicense", "public domain"}, Summary: "Unlicense - dedicates work to public domain.", URL: "https://unlicense.org/"},
	{Name: "CC0", Keywords: []string{"cc0", "creative commons zero"}, Summary: "CC0 - public domain dedication.", URL: "https://creativecommons.org/publicdomain/zero/1.0/"},
	{Name: "WTFPL", Keywords: []string{"wtfpl", "do what the fuck you want"}, Summary: "WTFPL - very permissive license.", URL: "http://www.wtfpl.net/"},
	{Name: "Zlib", Keywords: []string{"zlib license", "this software is provided 'as-is'"}, Summary: "Zlib license - permissive for compression libraries.", URL: "https://opensource.org/licenses/Zlib"},
	{Name: "AFL", Keywords: []string{"academic free license", "afl"}, Summary: "AFL - permissive academic license.", URL: "https://opensource.org/licenses/AFL-3.0"},
	{Name: "Eclipse", Keywords: []string{"eclipse public license", "epl"}, Summary: "EPL - copyleft with patent clauses.", URL: "https://www.eclipse.org/legal/epl-2.0/"},
	{Name: "CDDL", Keywords: []string{"common development and distribution license", "cddl"}, Summary: "CDDL - weak copyleft with patent grant.", URL: "https://opensource.org/licenses/CDDL-1.0"},
	{Name: "Mulan", Keywords: []string{"mulan ps", "mulan public license"}, Summary: "Mulan Public License - Chinese open source license.", URL: "https://license.coscl.org.cn/MulanPSL2/"},
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

	var files []string
	err = filepath.WalkDir(vendorPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".git") {
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
		files = append(files, path)
		return nil
	})
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	pool := workerpool.NewWorkerPool(runtime.NumCPU())
	defer pool.Stop()

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, path := range files {
		wg.Add(1)
		p := path
		pool.Submit(func(ctx context.Context) error {
			defer wg.Done()
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return nil
			}
			pathLower := strings.ToLower(rel)
			for _, rule := range knownVulnerabilities {
				if matchesAnyKeyword(pathLower, rule.Keywords) {
					key := rule.Name + ":" + rel
					mu.Lock()
					if !seen[key] {
						*findings = append(*findings, Finding{
							Package:  rule.Name,
							Summary:  rule.Summary,
							Path:     rel,
							Severity: SeverityHigh,
							URL:      rule.URL,
						})
						seen[key] = true
					}
					mu.Unlock()
					break
				}
			}

			if strings.HasSuffix(pathLower, "go.mod") || strings.HasSuffix(pathLower, "package.json") ||
				strings.HasSuffix(pathLower, "requirements.txt") || strings.HasSuffix(pathLower, "pyproject.toml") {
				data, err := os.ReadFile(p)
				if err != nil {
					return nil
				}
				content := strings.ToLower(string(data))
				for _, rule := range knownVulnerabilities {
					if matchesAnyKeyword(content, rule.Keywords) {
						key := rule.Name + ":" + rel
						mu.Lock()
						if !seen[key] {
							*findings = append(*findings, Finding{
								Package:  rule.Name,
								Summary:  rule.Summary,
								Path:     rel,
								Severity: SeverityHigh,
								URL:      rule.URL,
							})
							seen[key] = true
						}
						mu.Unlock()
						break
					}
				}
			}
			return nil
		})
	}

	wg.Wait()
	return nil
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

	var files []string
	err := filepath.WalkDir(vendorPath, func(path string, d os.DirEntry, err error) error {
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
		files = append(files, path)
		return nil
	})
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	pool := workerpool.NewWorkerPool(runtime.NumCPU())
	defer pool.Stop()

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, path := range files {
		wg.Add(1)
		p := path
		pool.Submit(func(ctx context.Context) error {
			defer wg.Done()
			rel, err := filepath.Rel(vendorPath, p)
			if err != nil {
				rel = p
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			content := strings.ToLower(string(data))
			for _, rule := range licenseRules {
				if matchesAnyKeyword(content, rule.Keywords) {
					key := "license:" + rule.Name + ":" + rel
					mu.Lock()
					if !seen[key] {
						*findings = append(*findings, Finding{
							Package:  "License",
							Summary:  rule.Summary,
							Path:     filepath.ToSlash(p),
							Severity: SeverityMedium,
							URL:      rule.URL,
						})
						seen[key] = true
					}
					mu.Unlock()
					break
				}
			}
			return nil
		})
	}

	wg.Wait()
	return nil
}

func scanSecrets(ctx context.Context, root string, cfg *config.Config, findings *[]Finding, seen map[string]bool) error {
	extensions := map[string]bool{
		".c": true, ".cpp": true, ".cc": true, ".cxx": true, ".h": true, ".hpp": true, ".hxx": true,
		".s": true, ".S": true, ".asm": true, ".fasm": true,
		".yaml": true, ".yml": true, ".json": true, ".env": true, ".toml": true, ".ini": true, ".sh": true, ".ps1": true, ".md": true, ".txt": true,
	}
	fileNames := map[string]bool{"dockerfile": true, "docker-compose.yml": true, "docker-compose.yaml": true}

	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
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
		files = append(files, path)
		return nil
	})
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	pool := workerpool.NewWorkerPool(runtime.NumCPU())
	defer pool.Stop()

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, path := range files {
		wg.Add(1)
		p := path
		pool.Submit(func(ctx context.Context) error {
			defer wg.Done()
			data, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			content := string(data)
			for _, rule := range secretRules {
				if rule.Pattern.MatchString(content) {
					key := "secret:" + rule.ID + ":" + p
					mu.Lock()
					if !seen[key] {
						*findings = append(*findings, Finding{
							Package:  "HardcodedSecret",
							Summary:  rule.Summary,
							Path:     filepath.ToSlash(p),
							Severity: SeverityHigh,
						})
						seen[key] = true
					}
					mu.Unlock()
					break
				}
			}
			return nil
		})
	}

	wg.Wait()
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

func matchesAnyKeyword(text string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
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