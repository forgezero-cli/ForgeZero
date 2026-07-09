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
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

type TrustStore struct {
	Keys []string
}

func LoadTrustStore(path string) (*TrustStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &TrustStore{}, nil
		}
		return nil, err
	}
	text := string(data)
	keys := make([]string, 0)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		keys = append(keys, line)
	}
	return &TrustStore{Keys: keys}, nil
}

func (s *TrustStore) Contains(key string) bool {
	if s == nil {
		return false
	}
	for _, existing := range s.Keys {
		if subtleEqual(existing, key) {
			return true
		}
	}
	return false
}

func (s *TrustStore) Add(key string) {
	if s == nil {
		return
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	for _, existing := range s.Keys {
		if subtleEqual(existing, key) {
			return
		}
	}
	s.Keys = append(s.Keys, key)
}

func (s *TrustStore) Save(path string) error {
	if s == nil {
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("# ForgeZero trusted keys\n")
	for _, key := range s.Keys {
		buf.WriteString(key)
		buf.WriteByte('\n')
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func subtleEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func ComputeHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func VerifyFileHash(path, expected string) error {
	actual, err := ComputeHash(path)
	if err != nil {
		return err
	}
	if !subtleEqual(actual, expected) {
		return errors.New("hash mismatch")
	}
	return nil
}

func VerifyPackagePath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if strings.Contains(absPath, "..") {
		return errors.New("invalid package path")
	}
	return nil
}

func WriteManifest(path string, data []byte) error {
	return utils.SecureWriteFile(path, data)
}
