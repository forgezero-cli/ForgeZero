package sbom

import (
	"fmt"
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
			return fmt.Errorf("getwd: %w", err)
		}
		root = cwd
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("abs root: %w", err)
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
		return fmt.Errorf("write sbom: %w", err)
	}
	merkle, err := utils.BuildMerkleRoot(rootAbs)
	if err == nil {
		var mbuf [48]byte
		n := copy(mbuf[:], []byte("sbom:merkle:"))
		copy(mbuf[n:], merkle[:])
		seal.UpdateGlobalState(mbuf[:n+len(merkle)])
	}
	return nil
}
