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

package sbom

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/seal"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func GenerateAndStoreSBOM(root, vendorDir, buildVersion string, cfg *config.Config, target, outPath string) error {
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return errors.New("getwd: " + err.Error())
		}
		root = cwd
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return errors.New("abs root: " + err.Error())
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	sbomDoc, err := Generate(rootAbs, vendorDir, buildVersion, cfg, target)
	if err != nil {
		return err
	}
	plain, err := Marshal(sbomDoc)
	if err != nil {
		return err
	}
	if err := utils.SecureWriteFile(outPath, plain); err != nil {
		return errors.New("write sbom: " + err.Error())
	}
	merkle, err := utils.BuildMerkleRoot(rootAbs)
	if err == nil {
		var mbuf [48]byte
		n := copy(mbuf[:], "sbom:merkle:")
		copy(mbuf[n:], merkle[:])
		seal.UpdateGlobalState(mbuf[:n+len(merkle)])
	}
	return nil
}