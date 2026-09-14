package builder

import (
	"context"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/drivers/scheduler"
	fzerr "github.com/forgezero-cli/ForgeZero/internal/errors"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func runTasks(ctx context.Context, tasks []config.Task, verbose bool, jobs int) error {
	if len(tasks) == 0 {
		return nil
	}
	if jobs < 1 {
		jobs = 1
	}
	stages := taskStages(tasks)
	for _, stage := range stages {
		indices := make([]int, 0, len(tasks))
		for i := range tasks {
			if taskStage(tasks[i].Stage) == stage {
				indices = append(indices, i)
			}
		}
		if err := runTaskStage(ctx, tasks, indices, verbose, jobs); err != nil {
			return err
		}
	}
	return nil
}

func taskStages(tasks []config.Task) []string {
	seen := make(map[string]struct{}, len(tasks))
	other := make([]string, 0, len(tasks))
	for _, task := range tasks {
		stage := taskStage(task.Stage)
		if _, ok := seen[stage]; ok {
			continue
		}
		seen[stage] = struct{}{}
		other = append(other, stage)
	}
	out := make([]string, 0, len(other))
	for _, stage := range []string{"pre-build", "build", "post-build"} {
		if _, ok := seen[stage]; ok {
			out = append(out, stage)
			delete(seen, stage)
		}
	}
	for _, stage := range other {
		if _, ok := seen[stage]; ok {
			out = append(out, stage)
			delete(seen, stage)
		}
	}
	return out
}

func taskStage(stage string) string {
	stage = strings.ToLower(strings.TrimSpace(stage))
	if stage == "" {
		return "pre-build"
	}
	return stage
}

func runTaskStage(ctx context.Context, tasks []config.Task, indices []int, verbose bool, jobs int) error {
	graph := make([][]int, len(indices))
	owners := make(map[string]int, len(indices)*2)
	for local, index := range indices {
		for _, output := range tasks[index].Outputs {
			owners[taskPath(output)] = local
		}
	}
	for local, index := range indices {
		seen := make(map[int]struct{}, len(tasks[index].Inputs))
		for _, input := range tasks[index].Inputs {
			if dep, ok := owners[taskPath(input)]; ok && dep != local {
				seen[dep] = struct{}{}
			}
		}
		for dep := range seen {
			graph[local] = append(graph[local], dep)
		}
	}
	dag := scheduler.NewDAGScheduler(jobs, len(indices))
	var firstErr error
	var errMu sync.Mutex
	for local, index := range indices {
		local, index := local, index
		_, err := dag.Submit(scheduler.AcquireTask(func(uintptr, uintptr) error {
			err := executeTask(ctx, tasks[index], verbose)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
			}
			return err
		}, 0, 0), graph[local])
		if err != nil {
			return err
		}
	}
	if err := dag.Run(ctx); err != nil {
		return err
	}
	return firstErr
}

func executeTask(ctx context.Context, task config.Task, verbose bool) error {
	if task.Command == "" || len(task.Outputs) == 0 {
		return fzerr.NewMsg(fzerr.CodeBuildActionFailed, task.Name)
	}
	needRun := false
	for _, output := range task.Outputs {
		if _, err := os.Stat(taskPath(output)); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			needRun = true
		}
	}
	key := ""
	if !needRun {
		var err error
		key, err = taskFingerprint(task)
		if err != nil {
			return err
		}
		if data, readErr := os.ReadFile(taskMarker(task)); readErr == nil && string(data) == key {
			return nil
		}
		needRun = true
	}
	if !needRun {
		return nil
	}
	if verbose {
		_, _ = os.Stdout.WriteString("Running task: " + task.Name + "\n")
	}
	name, args := taskShell(task.Command)
	if _, err := utils.RunCommand(ctx, verbose, os.Stdout, os.Stderr, name, args...); err != nil {
		return err
	}
	key, err := taskFingerprint(task)
	if err != nil {
		return err
	}
	return os.WriteFile(taskMarker(task), []byte(key), 0o600)
}

func taskFingerprint(task config.Task) (string, error) {
	inputs := append([]string(nil), task.Inputs...)
	digest, err := actionCacheKey(inputs, task.Command, nil)
	if err != nil {
		return "", err
	}
	return string(hexDigest(digest)), nil
}

func taskMarker(task config.Task) string {
	if len(task.Outputs) == 0 {
		return ".fz_task"
	}
	return taskPath(task.Outputs[0]) + ".fz-task"
}

func taskPath(path string) string {
	if runtime.GOOS == "windows" {
		path = strings.ReplaceAll(path, "/", "\\")
	} else {
		path = strings.ReplaceAll(path, "\\", "/")
	}
	return path
}

func taskShell(command string) (string, []string) {
	if taskShellCommand == "cmd.exe" {
		return taskShellCommand, []string{"/c", command}
	}
	return taskShellCommand, []string{"-c", command}
}

var taskShellCommand = func() string {
	if runtime.GOOS == "windows" {
		return "cmd.exe"
	}
	return "sh"
}()

func hexDigest(digest [32]byte) []byte {
	const table = "0123456789abcdef"
	out := make([]byte, 64)
	for i, value := range digest {
		out[i*2] = table[value>>4]
		out[i*2+1] = table[value&15]
	}
	return out
}
