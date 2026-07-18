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
	"os"
	"path/filepath"
	"testing"
)

func TestTrustStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys.toml")
	store := &TrustStore{Keys: []string{"abc123"}}
	if err := store.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadTrustStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Contains("abc123") {
		t.Fatal("expected key to be present")
	}
}

func TestVerifyManifestMissingSignature(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte(`{"name":"pkg"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyManifest(path); err == nil {
		t.Fatal("expected missing signature error")
	}
}

func TestAddRejectsUntrustedManifestSignature(t *testing.T) {
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(pkgPath, 0o755); err != nil {
		t.Fatal(err)
	}
	trustPath := filepath.Join(dir, "keys.toml")
	SetTrustedKeysPath(trustPath)
	defer SetTrustedKeysPath(defaultTrustStorePath)
	manifest := []byte(`{"signature":"unknown:abc123","hash":""}`)
	if err := os.WriteFile(filepath.Join(pkgPath, "manifest.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgPath, "package.tar"), []byte("pkg"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(context.TODO(), pkgPath, ""); err == nil {
		t.Fatal("expected untrusted signature error")
	}
}

func TestTrustCommand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys.toml")
	SetTrustedKeysPath(path)
	defer SetTrustedKeysPath(defaultTrustStorePath)
	if err := Trust("test-key"); err != nil {
		t.Fatal(err)
	}
	store, err := LoadTrustedKeys()
	if err != nil {
		t.Fatal(err)
	}
	if !store.Contains("test-key") {
		t.Fatal("expected trusted key to be stored")
	}
}

func TestVerifyManifestRejectsUntrustedSignature(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	trustPath := filepath.Join(dir, "keys.toml")
	SetTrustedKeysPath(trustPath)
	defer SetTrustedKeysPath(defaultTrustStorePath)
	if err := os.WriteFile(path, []byte(`{"signature":"unknown:abc123"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyManifest(path); err == nil {
		t.Fatal("expected untrusted signature error")
	}
}
