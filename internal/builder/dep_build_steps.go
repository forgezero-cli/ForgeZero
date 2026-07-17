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
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/variables"
)

type stepGroupKey struct {
	stage int
	group string
}

type stepGroup struct {
	key        stepGroupKey
	steps      []config.BuildStep
	parallel   bool
	firstIndex int
}

func (db *DepBuilder) runCustomSteps() error {
	if db.depCfg == nil || len(db.depCfg.DepBuild.Steps) == 0 {
		return nil
	}
	envMap := db.setupEnvironment()
	envMap["FZ_DEP_NAME"] = db.depName
	envMap["FZ_DEP_PATH"] = db.depPath
	if abs, err := filepath.Abs(db.depPath); err == nil {
		envMap["FZ_DEP_PATH"] = abs
	}
	steps, err := db.expandCustomSteps(db.depCfg.DepBuild.Steps, envMap)
	if err != nil {
		return err
	}
	groups := make(map[stepGroupKey]*stepGroup, len(steps))
	order := make([]stepGroupKey, 0, len(steps))
	for idx, step := range steps {
		groupName := step.Group
		if groupName == "" {
			groupName = "#" + strconv.Itoa(idx)
		}
		key := stepGroupKey{stage: step.Stage, group: groupName}
		entry, ok := groups[key]
		if !ok {
			entry = &stepGroup{key: key, firstIndex: idx}
			groups[key] = entry
			order = append(order, key)
		}
		entry.steps = append(entry.steps, step)
		if step.Parallel {
			entry.parallel = true
		}
	}
	sort.Slice(order, func(i, j int) bool {
		if order[i].stage != order[j].stage {
			return order[i].stage < order[j].stage
		}
		return groups[order[i]].firstIndex < groups[order[j]].firstIndex
	})
	for _, key := range order {
		if err := db.runStepGroup(groups[key], envMap); err != nil {
			return err
		}
	}
	return nil
}

func (db *DepBuilder) expandCustomSteps(rawSteps []config.BuildStep, envMap map[string]string) ([]config.BuildStep, error) {
	if len(db.depCfg.DepBuild.StepSets) == 0 {
		return rawSteps, nil
	}
	stepSets := make(map[string]config.BuildStep, len(db.depCfg.DepBuild.StepSets))
	for _, set := range db.depCfg.DepBuild.StepSets {
		if set.Name == "" {
			continue
		}
		stepSets[set.Name] = set.BuildStep
	}
	expanded := make([]config.BuildStep, 0, len(rawSteps))
	for _, step := range rawSteps {
		if step.StepSet == "" {
			expanded = append(expanded, step)
			continue
		}
		tmpl, ok := stepSets[step.StepSet]
		if !ok {
			return nil, errors.New("undefined step_set: " + step.StepSet)
		}
		expanded = append(expanded, db.expandStepSet(step, tmpl, envMap))
	}
	return expanded, nil
}

func (db *DepBuilder) expandStepSet(step config.BuildStep, tmpl config.BuildStep, envMap map[string]string) config.BuildStep {
	params := envMap
	if len(step.With) > 0 {
		params = make(map[string]string, len(envMap)+len(step.With))
		for k, v := range envMap {
			params[k] = v
		}
		for k, v := range step.With {
			params[k] = variables.ExpandString(v, envMap)
		}
	}
	result := tmpl
	result.StepSet = ""
	result.Group = expandStepField(step.Group, tmpl.Group, params)
	result.Stage = resolveStage(step.Stage, tmpl.Stage)
	result.Parallel = step.Parallel || tmpl.Parallel
	result.Try = step.Try || tmpl.Try
	result.Catch = step.Catch || tmpl.Catch
	result.Finally = step.Finally || tmpl.Finally
	result.Command = expandStepField(step.Command, tmpl.Command, params)
	result.Run = expandStepField(step.Run, tmpl.Run, params)
	result.If = expandStepField(step.If, tmpl.If, params)
	result.Elif = expandStepField(step.Elif, tmpl.Elif, params)
	result.Else = step.Else || tmpl.Else
	result.Persistent = step.Persistent || tmpl.Persistent
	result.Inputs = expandStringSlice(step.Inputs, tmpl.Inputs, params)
	result.Outputs = expandStringSlice(step.Outputs, tmpl.Outputs, params)
	return result
}

func expandStepField(override, base string, params map[string]string) string {
	value := base
	if override != "" {
		value = override
	}
	if value == "" {
		return ""
	}
	return variables.ExpandString(value, params)
}

func expandStringSlice(override, base []string, params map[string]string) []string {
	values := base
	if len(override) > 0 {
		values = override
	}
	if len(values) == 0 {
		return nil
	}
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = variables.ExpandString(value, params)
	}
	return result
}

func resolveStage(override, base int) int {
	if override != 0 {
		return override
	}
	return base
}

func (db *DepBuilder) runStepGroup(group *stepGroup, envMap map[string]string) error {
	eligible := make([]config.BuildStep, 0, len(group.steps))
	for idx := range group.steps {
		step := group.steps[idx]
		if step.Else {
			if db.groupHasMatch(group.steps[:idx], envMap) {
				continue
			}
		} else if step.If != "" || step.Elif != "" {
			condition := step.If
			if condition == "" {
				condition = step.Elif
			}
			ok, err := db.evalStepCondition(condition, envMap)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
		}
		eligible = append(eligible, step)
	}
	if group.parallel && !groupHasControlFlow(eligible) {
		return db.runParallelSteps(eligible, envMap)
	}
	return db.runSequentialGroup(eligible, envMap)
}

func groupHasControlFlow(steps []config.BuildStep) bool {
	for _, step := range steps {
		if step.Try || step.Catch || step.Finally {
			return true
		}
	}
	return false
}

func (db *DepBuilder) runSequentialGroup(steps []config.BuildStep, envMap map[string]string) error {
	for i := 0; i < len(steps); i++ {
		step := steps[i]
		if step.Try {
			j := i + 1
			for j < len(steps) && !steps[j].Catch && !steps[j].Finally && !steps[j].Try {
				j++
			}
			k := j
			for k < len(steps) && (steps[k].Catch || steps[k].Finally) {
				k++
			}
			err := db.runSequentialSteps(steps[i:j], envMap)
			if err := db.runCatchFinally(err, steps[j:k], envMap); err != nil {
				return err
			}
			i = k - 1
			continue
		}
		if step.Catch {
			return errors.New("catch without try")
		}
		if err := db.runStep(step, envMap); err != nil {
			return err
		}
	}
	return nil
}

func (db *DepBuilder) runSequentialSteps(steps []config.BuildStep, envMap map[string]string) error {
	for _, step := range steps {
		if err := db.runStep(step, envMap); err != nil {
			return err
		}
	}
	return nil
}

func (db *DepBuilder) runCatchFinally(tryErr error, steps []config.BuildStep, envMap map[string]string) error {
	for _, step := range steps {
		if step.Catch {
			if tryErr == nil {
				continue
			}
			if err := db.runStep(step, envMap); err != nil {
				return err
			}
			tryErr = nil
			continue
		}
		if step.Finally {
			if err := db.runStep(step, envMap); err != nil && tryErr == nil {
				tryErr = err
			}
			continue
		}
		return errors.New("invalid try/catch/finally block")
	}
	return tryErr
}

func (db *DepBuilder) runParallelSteps(steps []config.BuildStep, envMap map[string]string) error {
	if len(steps) == 0 {
		return nil
	}
	max := runtime.GOMAXPROCS(0)
	if max <= 0 {
		max = 1
	}
	sem := make(chan struct{}, max)
	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error
	for _, step := range steps {
		step := step
		if step.Run != "" {
			if err := db.runStep(step, envMap); err != nil {
				return err
			}
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := db.runStep(step, envMap); err != nil {
				errOnce.Do(func() { firstErr = err })
			}
		}()
	}
	wg.Wait()
	return firstErr
}

func (db *DepBuilder) groupHasMatch(group []config.BuildStep, envMap map[string]string) bool {
	for _, step := range group {
		if step.Else {
			continue
		}
		condition := step.If
		if condition == "" {
			condition = step.Elif
		}
		if condition == "" {
			continue
		}
		ok, _ := db.evalStepCondition(condition, envMap)
		if ok {
			return true
		}
	}
	return false
}

func (db *DepBuilder) runStep(step config.BuildStep, envMap map[string]string) error {
	if step.Command != "" {
		cmdLine := variables.ExpandString(step.Command, envMap)
		resolvedInputs := make([]string, 0, len(step.Inputs))
		for _, in := range step.Inputs {
			resolvedInputs = append(resolvedInputs, filepath.Join(db.depPath, variables.ExpandString(in, envMap)))
		}
		resolvedOutputs := make([]string, 0, len(step.Outputs))
		for _, out := range step.Outputs {
			resolvedOutputs = append(resolvedOutputs, filepath.Join(db.depPath, variables.ExpandString(out, envMap)))
		}
		if cached, err := db.restoreStepCache(cmdLine, resolvedInputs, resolvedOutputs, envMap); err != nil {
			return err
		} else if cached {
			return nil
		}
		if err := db.runCommandStep(cmdLine, envMap); err != nil {
			return err
		}
		return db.storeStepCache(cmdLine, resolvedInputs, resolvedOutputs, envMap)
	}
	if step.Run != "" {
		return db.runInternalStep(step.Run)
	}
	return nil
}

func (db *DepBuilder) cacheDir() string {
	return filepath.Join(db.depPath, ".fz_cache")
}

func (db *DepBuilder) restoreStepCache(action string, inputs, outputs []string, envMap map[string]string) (bool, error) {
	if len(outputs) == 0 || len(inputs) == 0 {
		return false, nil
	}
	cacheDir := db.cacheDir()
	return actionCacheRestore(db.ctx, inputs, action, outputs, cacheDir)
}

func (db *DepBuilder) storeStepCache(action string, inputs, outputs []string, envMap map[string]string) error {
	if len(outputs) == 0 || len(inputs) == 0 {
		return nil
	}
	cacheDir := db.cacheDir()
	env := envMapToSlice(envMap)
	return actionCacheStoreSync(inputs, action, outputs, env, cacheDir)
}

func envMapToSlice(envMap map[string]string) []string {
	if len(envMap) == 0 {
		return nil
	}
	env := make([]string, 0, len(envMap))
	for k, v := range envMap {
		env = append(env, k+"="+v)
	}
	sort.Strings(env)
	return env
}

func (db *DepBuilder) runCommandStep(command string, envMap map[string]string) error {
	if command == "" {
		return nil
	}
	db.logf("info", "Running command: "+command)
	if envMap == nil {
		envMap = make(map[string]string)
	}
	envSlice := make([]string, 0, len(envMap))
	for k, v := range envMap {
		envSlice = append(envSlice, k+"="+v)
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(db.ctx, "cmd.exe", "/C", command)
	} else {
		cmd = exec.CommandContext(db.ctx, "sh", "-c", command)
	}
	cmd.Dir = db.depPath
	cmd.Env = envSlice
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		db.logf("error", "Command failed: "+command)
		return err
	}
	db.logf("info", "Command completed: "+command)
	return nil
}

func (db *DepBuilder) runInternalStep(name string) error {
	switch strings.TrimSpace(strings.ToLower(name)) {
	case "pre_build":
		return db.runPreBuildScripts()
	case "post_build":
		return db.runPostBuildScripts()
	}
	return errors.New("unsupported internal step: " + name)
}

func (db *DepBuilder) evalStepCondition(cond string, envMap map[string]string) (bool, error) {
	if cond == "" {
		return false, nil
	}
	expanded := variables.ExpandString(cond, envMap)
	expanded = strings.TrimSpace(expanded)
	if expanded == "" {
		return false, nil
	}
	if strings.Contains(expanded, "!=") {
		parts := strings.SplitN(expanded, "!=", 2)
		return trimQuotes(strings.TrimSpace(parts[0])) != trimQuotes(strings.TrimSpace(parts[1])), nil
	}
	if strings.Contains(expanded, "==") {
		parts := strings.SplitN(expanded, "==", 2)
		return trimQuotes(strings.TrimSpace(parts[0])) == trimQuotes(strings.TrimSpace(parts[1])), nil
	}
	expanded = trimQuotes(expanded)
	if expanded == "0" || strings.EqualFold(expanded, "false") {
		return false, nil
	}
	return expanded != "", nil
}

func trimQuotes(value string) string {
	if len(value) >= 2 {
		first := value[0]
		last := value[len(value)-1]
		if (first == '\'' && last == '\'') || (first == '"' && last == '"') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
