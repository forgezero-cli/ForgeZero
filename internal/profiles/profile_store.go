package profiles

import (
	"os"
	"path/filepath"
	"strings"
)

const defaultProfileStoreFile = ".profile.config"

func defaultProfileStorePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cfgDir := filepath.Join(home, ".config", "fz")
	return filepath.Join(cfgDir, defaultProfileStoreFile), nil
}

func ValidateProfileName(name string) (string, bool) {
	p := ParseUserProfile(name)
	switch p.Name {
	case "balanced", "performance", "power-saver":
		return p.Name, true
	default:
		return "", false
	}
}

func ReadSavedProfile(storePath string) (string, error) {
	if storePath == "" {
		var err error
		storePath, err = defaultProfileStorePath()
		if err != nil {
			return "", err
		}
	}
	b, err := os.ReadFile(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "", nil
	}
	p, ok := ValidateProfileName(s)
	if !ok {
		return "", nil
	}
	return p, nil
}

func SaveProfile(storePath, profile string) error {
	if storePath == "" {
		var err error
		storePath, err = defaultProfileStorePath()
		if err != nil {
			return err
		}
	}
	p, ok := ValidateProfileName(profile)
	if !ok {
		return os.ErrInvalid
	}
	data := []byte(p)
	if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(storePath, data, 0o644)
}
