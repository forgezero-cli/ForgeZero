/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version of the License.
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
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/drivers/scheduler"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func buildRulesGraph(rules []config.BuildRule) ([][]int, error) {
	outputs := make(map[string]int, len(rules)*2)
	for i, rule := range rules {
		for _, out := range rule.Outputs {
			key := normalizeRulePath(out)
			if _, ok := outputs[key]; ok {
				return nil, errors.New("duplicate build rule output: " + out)
			}
			outputs[key] = i
		}
	}
	graph := make([][]int, len(rules))
	for i, rule := range rules {
		deps := make([]int, 0, len(rule.Inputs))
		seen := make(map[int]struct{}, len(rule.Inputs))
		for _, in := range rule.Inputs {
			if dep, ok := outputs[normalizeRulePath(in)]; ok {
				seen[dep] = struct{}{}
			}
		}
		if rule.Depfile != "" {
			if fi, err := os.Stat(rule.Depfile); err == nil && !fi.IsDir() {
				parsed, err := utils.ParseDepFilePath(rule.Depfile)
				if err != nil {
					return nil, err
				}
				for _, dep := range parsed {
					if depIdx, ok := outputs[normalizeRulePath(dep)]; ok {
						seen[depIdx] = struct{}{}
					}
				}
			}
		}
		for dep := range seen {
			deps = append(deps, dep)
		}
		sort.Ints(deps)
		graph[i] = deps
	}
	return graph, nil
}

func normalizeRulePath(path string) string {
	return filepath.Clean(path)
}

func executeBuildRule(ctx context.Context, rule config.BuildRule, verbose bool) error {
	needsBuild, err := ruleNeedsBuild(rule)
	if err != nil {
		return err
	}
	if !needsBuild {
		if verbose {
			os.Stdout.WriteString("Skipping build rule: " + rule.Name + "\n")
		}
		return nil
	}
	cmd := expandRuleAction(rule)
	name, args := utils.ShellCommand(cmd)
	if verbose {
		os.Stdout.WriteString("Running build rule: " + rule.Name + "\n")
		os.Stdout.WriteString("Command: " + cmd + "\n")
	}
	_, err = utils.RunCommand(ctx, verbose, os.Stdout, os.Stderr, name, args...)
	return err
}

func ruleNeedsBuild(rule config.BuildRule) (bool, error) {
	if len(rule.Outputs) == 0 {
		return false, errors.New("build rule outputs are required")
	}
	outTimes := make([]time.Time, 0, len(rule.Outputs))
	for _, out := range rule.Outputs {
		fi, err := os.Stat(out)
		if err != nil {
			if os.IsNotExist(err) {
				return true, nil
			}
			return false, err
		}
		if fi.IsDir() {
			return false, errors.New("build rule output is a directory: " + out)
		}
		outTimes = append(outTimes, fi.ModTime())
	}
	if len(outTimes) == 0 {
		return true, nil
	}
	latestDep := time.Time{}
	deps, err := collectRuleDependencies(rule)
	if err != nil {
		return false, err
	}
	for _, dep := range deps {
		fi, err := os.Stat(dep)
		if err != nil {
			if os.IsNotExist(err) {
				return true, nil
			}
			return false, err
		}
		if fi.IsDir() {
			return false, errors.New("build rule dependency is a directory: " + dep)
		}
		if fi.ModTime().After(latestDep) {
			latestDep = fi.ModTime()
		}
	}
	if latestDep.IsZero() {
		return true, nil
	}
	for _, outTime := range outTimes {
		if outTime.Before(latestDep) {
			return true, nil
		}
	}
	return false, nil
}

func collectRuleDependencies(rule config.BuildRule) ([]string, error) {
	if len(rule.Inputs) == 0 && rule.Depfile == "" {
		return nil, nil
	}
	deps := make([]string, 0, len(rule.Inputs))
	deps = append(deps, rule.Inputs...)
	if rule.Depfile != "" {
		if fi, err := os.Stat(rule.Depfile); err == nil && !fi.IsDir() {
			parsed, err := utils.ParseDepFilePath(rule.Depfile)
			if err != nil {
				return nil, err
			}
			deps = append(deps, parsed...)
		}
	}
	return deps, nil
}

func expandRuleAction(rule config.BuildRule) string {
	action := rule.Action
	action = strings.ReplaceAll(action, "$in", shellJoin(rule.Inputs))
	action = strings.ReplaceAll(action, "$out", shellJoin(rule.Outputs))
	if rule.Depfile != "" {
		action = strings.ReplaceAll(action, "$depfile", shellEscape(rule.Depfile))
	}
	return action
}

func shellJoin(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(paths))
	for _, path := range paths {
		quoted = append(quoted, shellEscape(path))
	}
	return strings.Join(quoted, " ")
}

func shellEscape(value string) string {
	if value == "" {
		return "''"
	}
	if runtime.GOOS == "windows" {
		return windowsShellEscape(value)
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\'' || r == '\\' || r == '"' || r == '$' || r == '`' || r == '!' || r == ';'
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func windowsShellEscape(value string) string {
	if value == "" {
		return "\"\""
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '"'
	}) == -1 {
		return value
	}
	return "\"" + strings.ReplaceAll(value, "\"", "\"\"") + "\""
}

func finalRuleOutputs(rules []config.BuildRule) ([]string, string) {
	allOutputs := make([]string, 0, len(rules)*2)
	inputs := make(map[string]struct{}, len(rules)*2)
	for _, rule := range rules {
		for _, in := range rule.Inputs {
			inputs[normalizeRulePath(in)] = struct{}{}
		}
	}
	leafOutputs := make([]string, 0, len(rules)*2)
	for _, rule := range rules {
		for _, out := range rule.Outputs {
			clean := normalizeRulePath(out)
			allOutputs = append(allOutputs, clean)
			if _, ok := inputs[clean]; !ok {
				leafOutputs = append(leafOutputs, clean)
			}
		}
	}
	if len(leafOutputs) == 0 && len(allOutputs) > 0 {
		leafOutputs = append(leafOutputs, allOutputs[len(allOutputs)-1])
	}
	binary := ""
	if len(leafOutputs) > 0 {
		binary = leafOutputs[0]
	}
	return allOutputs, binary
}

func runBuildRules(ctx context.Context, cfg *config.Config, verbose bool, jobs int) (*BuildResult, error) {
	if cfg == nil || len(cfg.BuildRules) == 0 {
		return nil, nil
	}
	graph, err := buildRulesGraph(cfg.BuildRules)
	if err != nil {
		return nil, err
	}
	if jobs <= 0 {
		jobs = 1
	}
	dag := scheduler.NewDAGScheduler(jobs, len(cfg.BuildRules))
	for i := range cfg.BuildRules {
		rule := cfg.BuildRules[i]
		idx := i
		_, err := dag.Submit(func(ctx context.Context) error {
			return executeBuildRule(ctx, rule, verbose)
		}, graph[idx])
		if err != nil {
			return nil, err
		}
	}
	if err := dag.Run(ctx); err != nil {
		return nil, err
	}
	outs, binary := finalRuleOutputs(cfg.BuildRules)
	return &BuildResult{ObjectFiles: outs, Binary: binary}, nil
}
