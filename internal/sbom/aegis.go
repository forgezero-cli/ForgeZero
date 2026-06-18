package sbom

import (
	"errors"
	"os"
	"path/filepath"

	"fz/internal/config"
	"fz/internal/seal"
	"fz/internal/utils"
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