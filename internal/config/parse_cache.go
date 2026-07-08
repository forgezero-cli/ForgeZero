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

package config

import (
	"os"
	"path/filepath"
	"sync"
)

type cacheEntry struct {
	modTime int64
	size    int64
	cfg     *Config
}

var configCache sync.Map

func cacheKey(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return abs
}

func loadConfigCache(path string, fi os.FileInfo) (*Config, bool) {
	entryValue, ok := configCache.Load(cacheKey(path))
	if !ok {
		return nil, false
	}
	entry := entryValue.(*cacheEntry)
	if entry.size != fi.Size() || entry.modTime != fi.ModTime().UnixNano() {
		return nil, false
	}
	return cloneConfig(entry.cfg), true
}

func storeConfigCache(path string, fi os.FileInfo, cfg *Config) {
	configCache.Store(cacheKey(path), &cacheEntry{
		modTime: fi.ModTime().UnixNano(),
		size:    fi.Size(),
		cfg:     cloneConfig(cfg),
	})
}

func clearConfigCache() {
	configCache.Range(func(key, value interface{}) bool {
		configCache.Delete(key)
		return true
	})
}

func cloneConfig(in *Config) *Config {
	if in == nil {
		return nil
	}
	out := *in
	out.SourceDirs = cloneStringSlice(in.SourceDirs)
	out.SourceFiles = cloneStringSlice(in.SourceFiles)
	out.Exclude = cloneStringSlice(in.Exclude)
	out.Include = cloneStringSlice(in.Include)
	out.Scripts = cloneStringSlice(in.Scripts)
	out.Libs = cloneStringSlice(in.Libs)
	out.AuditIgnore = cloneStringSlice(in.AuditIgnore)
	out.ToolChecksums = cloneStringMap(in.ToolChecksums)
	out.Variables = cloneStringMap(in.Variables)
	out.Flags.Asm = cloneStringSlice(in.Flags.Asm)
	out.Flags.Cc = cloneStringSlice(in.Flags.Cc)
	out.Flags.Ld = cloneStringSlice(in.Flags.Ld)
	out.ToolchainSettings.SearchPriority = cloneStringSlice(in.ToolchainSettings.SearchPriority)
	out.ToolchainSettings.EnvAllow = cloneStringSlice(in.ToolchainSettings.EnvAllow)
	out.ToolchainSettings.ToolPaths = cloneStringMap(in.ToolchainSettings.ToolPaths)
	out.Hooks = cloneHooks(in.Hooks)
	out.BuildRules = cloneBuildRules(in.BuildRules)
	out.ISO.CustomArgs = cloneStringSlice(in.ISO.CustomArgs)
	return &out
}

func cloneStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneHooks(in Hooks) Hooks {
	out := Hooks{OnFailure: in.OnFailure}
	if len(in.PreBuild) == 0 {
		return out
	}
	out.PreBuild = make([]Hook, len(in.PreBuild))
	copy(out.PreBuild, in.PreBuild)
	return out
}

func cloneBuildRules(in []BuildRule) []BuildRule {
	if len(in) == 0 {
		return nil
	}
	out := make([]BuildRule, len(in))
	copy(out, in)
	for i := range out {
		out[i].Inputs = cloneStringSlice(in[i].Inputs)
		out[i].Outputs = cloneStringSlice(in[i].Outputs)
	}
	return out
}
