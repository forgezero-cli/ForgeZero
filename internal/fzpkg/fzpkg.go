/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package fzpkg

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

const defaultTrustStorePath = "keys.toml"

var trustedKeysPath = defaultTrustStorePath

func SetTrustedKeysPath(path string) {
	if strings.TrimSpace(path) != "" {
		trustedKeysPath = path
	}
}

func LoadTrustedKeys() (*TrustStore, error) {
	return LoadTrustStore(trustedKeysPath)
}

func Add(ctx context.Context, pkgURL, version string) error {
	_ = ctx
	_ = version
	if pkgURL == "" {
		return errors.New("package URL is required")
	}
	trustStore, err := LoadTrustedKeys()
	if err != nil {
		return err
	}
	if len(trustStore.Keys) == 0 {
		return errors.New("no trusted keys configured")
	}
	pkgPath := filepath.Clean(pkgURL)
	if err := VerifyPackagePath(pkgPath); err != nil {
		return err
	}
	manifestPath := filepath.Join(pkgPath, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		return errors.New("manifest missing: " + err.Error())
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest struct {
		Signature string `json:"signature"`
		Hash      string `json:"hash"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	if manifest.Signature == "" {
		return errors.New("package signature missing")
	}
	if manifest.Hash != "" {
		if err := VerifyFileHash(filepath.Join(pkgPath, "package.tar"), manifest.Hash); err != nil {
			return err
		}
	}
	return nil
}

func Remove(ctx context.Context, pkgURL string) error {
	_ = ctx
	_ = pkgURL
	return nil
}

func List() error {
	trustStore, err := LoadTrustedKeys()
	if err != nil {
		return err
	}
	if len(trustStore.Keys) == 0 {
		_, _ = os.Stdout.WriteString("No trusted keys configured.\n")
		return nil
	}
	for _, key := range trustStore.Keys {
		_, _ = os.Stdout.WriteString(key + "\n")
	}
	return nil
}

func Update(ctx context.Context) error {
	_ = ctx
	return nil
}

func Verify(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return errors.New("package name is required")
	}
	return VerifyPackagePath(pkgName)
}

func Sign(pkgName string) error {
	if strings.TrimSpace(pkgName) == "" {
		return errors.New("package name is required")
	}
	return nil
}

func Keys() error {
	return List()
}

func Trust(key string) error {
	trustStore, err := LoadTrustedKeys()
	if err != nil {
		return err
	}
	trustStore.Add(key)
	return trustStore.Save(trustedKeysPath)
}

func Install(ctx context.Context, pkgName string) error {
	_ = ctx
	return Add(ctx, pkgName, "")
}

func VerifyManifest(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var manifest struct {
		Signature string `json:"signature"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	if manifest.Signature == "" {
		return errors.New("manifest signature missing")
	}
	if !strings.Contains(manifest.Signature, ":") {
		return errors.New("manifest signature is malformed")
	}
	parts := strings.SplitN(manifest.Signature, ":", 2)
	keyID := strings.TrimSpace(parts[0])
	if keyID == "" {
		return errors.New("manifest signature key is empty")
	}
	store, err := LoadTrustedKeys()
	if err != nil {
		return err
	}
	if !store.Contains(keyID) {
		return errors.New("manifest signature key is not trusted")
	}
	return nil
}

func InstallFromCatalog(ctx context.Context, pkgName string) error {
	return Add(ctx, pkgName, "")
}

func WriteTrustStore(path string, keys []string) error {
	store := &TrustStore{Keys: append([]string(nil), keys...)}
	return store.Save(path)
}

func safeJoin(base, name string) string {
	cleanBase := filepath.Clean(base)
	cleanName := filepath.Clean(name)
	if cleanName == "." || cleanName == "" {
		return cleanBase
	}
	joined := filepath.Join(cleanBase, cleanName)
	if !strings.HasPrefix(joined, cleanBase+string(os.PathSeparator)) && joined != cleanBase {
		return cleanBase
	}
	return joined
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func init() {
	_ = ensureDir(".")
	_ = utils.SecureWriteFile(defaultTrustStorePath, []byte("# ForgeZero trusted keys\n"))
}
