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

package builder

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

type DepBuilder struct {
	ctx             context.Context
	depPath         string
	depName         string
	depCfg          *config.Config
	globalAutoBuild *config.AutoBuildConfig
	verbose         bool
}

func NewDepBuilder(ctx context.Context, depPath, depName string, depCfg *config.Config, globalAutoBuild *config.AutoBuildConfig, verbose bool) *DepBuilder {
	return &DepBuilder{
		ctx:             ctx,
		depPath:         depPath,
		depName:         depName,
		depCfg:          depCfg,
		globalAutoBuild: globalAutoBuild,
		verbose:         verbose,
	}
}

func (db *DepBuilder) logf(level, msg string) {
	if db.globalAutoBuild != nil && db.globalAutoBuild.LogLevel == "quiet" {
		if level == "error" {
			_, _ = os.Stdout.WriteString("[ERROR] ")
			_, _ = os.Stdout.WriteString(msg)
			_, _ = os.Stdout.WriteString("\n")
		}
		return
	}
	_, _ = os.Stdout.WriteString("[")
	_, _ = os.Stdout.WriteString(level)
	_, _ = os.Stdout.WriteString("::")
	_, _ = os.Stdout.WriteString(strings.ToUpper(db.depName))
	_, _ = os.Stdout.WriteString("] ")
	_, _ = os.Stdout.WriteString(msg)
	_, _ = os.Stdout.WriteString("\n")
}

func (db *DepBuilder) setupEnvironment() map[string]string {
	env := os.Environ()
	envMap := make(map[string]string)
	for _, pair := range env {
		idx := strings.IndexByte(pair, '=')
		if idx > 0 {
			envMap[pair[:idx]] = pair[idx+1:]
		}
	}

	if db.globalAutoBuild != nil && len(db.globalAutoBuild.DefaultEnvironment) > 0 {
		for k, v := range db.globalAutoBuild.DefaultEnvironment {
			envMap[k] = v
		}
	}

	if db.depCfg != nil && len(db.depCfg.DepBuild.Environment) > 0 {
		for k, v := range db.depCfg.DepBuild.Environment {
			envMap[k] = v
		}
	}

	return envMap
}

func (db *DepBuilder) runScript(scriptName, script string) error {
	if script == "" {
		return nil
	}

	db.logf("info", "Running "+scriptName+" script")

	tmpPath, err := writeTempShellScript(script)
	if err != nil {
		return err
	}
	if tmpPath == "" {
		return nil
	}
	defer os.Remove(tmpPath)

	envMap := db.setupEnvironment()
	envSlice := make([]string, 0, len(envMap))
	for k, v := range envMap {
		envSlice = append(envSlice, k+"="+v)
	}

	cmd := exec.CommandContext(db.ctx, "sh", tmpPath)
	cmd.Dir = db.depPath
	cmd.Env = envSlice
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		db.logf("error", scriptName+" script failed")
		return err
	}

	db.logf("info", scriptName+" script completed")
	return nil
}

func (db *DepBuilder) runPreBuildScripts() error {
	if db.depCfg == nil || len(db.depCfg.DepBuild.PreBuild) == 0 {
		return nil
	}

	db.logf("info", "Running pre-build scripts")

	for _, script := range db.depCfg.DepBuild.PreBuild {
		if err := db.runScript("pre_build", script); err != nil {
			return err
		}
	}

	return nil
}

func (db *DepBuilder) runPostBuildScripts() error {
	if db.depCfg == nil || len(db.depCfg.DepBuild.PostBuild) == 0 {
		return nil
	}

	db.logf("info", "Running post-build scripts")

	for _, script := range db.depCfg.DepBuild.PostBuild {
		if err := db.runScript("post_build", script); err != nil {
			if db.globalAutoBuild != nil && db.globalAutoBuild.ContinueOnError {
				db.logf("warn", "Post-build script error (continuing)")
				continue
			}
			return err
		}
	}

	return nil
}

func (db *DepBuilder) shouldIncludeFile(path string) bool {
	if db.depCfg == nil {
		return true
	}

	basename := filepath.Base(path)

	if len(db.depCfg.DepBuild.ExcludeFiles) > 0 {
		for _, pattern := range db.depCfg.DepBuild.ExcludeFiles {
			if matched, _ := filepath.Match(pattern, basename); matched {
				db.logf("debug", "Excluding: "+path)
				return false
			}
		}
	}

	if len(db.depCfg.DepBuild.OnlyFiles) > 0 {
		for _, pattern := range db.depCfg.DepBuild.OnlyFiles {
			if matched, _ := filepath.Match(pattern, basename); matched {
				return true
			}
		}
		db.logf("debug", "Not in only_files: "+path)
		return false
	}

	return true
}

func (db *DepBuilder) Build(outArchive string, excludePatterns []string, jobs int, buildType string) (string, error) {
	db.logf("info", "Build starting for: "+db.depName)

	if db.depCfg != nil && !db.depCfg.DepBuild.Enabled {
		db.logf("warn", "Dependency disabled in fz.toml")
		return "", errors.New("dependency build disabled")
	}

	if err := db.runPreBuildScripts(); err != nil {
		return "", err
	}

	if err := db.runCustomSteps(); err != nil {
		return "", err
	}

	db.logf("info", "Collecting sources from: "+db.depPath)

	srcDirs := []string{filepath.Join(db.depPath, "src"), filepath.Join(db.depPath, "lib")}

	hasSourceDir := false
	for _, srcDir := range srcDirs {
		if info, err := os.Stat(srcDir); err == nil && info.IsDir() {
			hasSourceDir = true
			break
		}
	}
	if !hasSourceDir {
		srcDirs = []string{db.depPath}
	}
	var sourceFiles []string

	for _, srcDir := range srcDirs {
		if info, err := os.Stat(srcDir); err != nil || !info.IsDir() {
			continue
		}

		_ = utils.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				name := info.Name()
				if name == "test" || name == "tests" || name == "examples" || name == "fuzzing" {
					return filepath.SkipDir
				}
				return nil
			}

			if !utils.SupportedExtension(strings.ToLower(filepath.Ext(path))) {
				return nil
			}

			if !db.shouldIncludeFile(path) {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			s := string(data)
			if strings.Contains(s, "int main") || strings.Contains(s, "main(") || strings.Contains(s, "main (") {
				db.logf("debug", "Skipping main: "+path)
				return nil
			}

			sourceFiles = append(sourceFiles, path)
			return nil
		})
	}

	if len(sourceFiles) == 0 {
		db.logf("warn", "No source files found")
		return "", errors.New("no source files found")
	}

	db.logf("info", "Found source files to compile")

	if err := db.runPostBuildScripts(); err != nil {
		db.logf("warn", "Post-build error (continuing)")
	}

	return outArchive, nil
}
