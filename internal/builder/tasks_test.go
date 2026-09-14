package builder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/config"
)

func TestRunTasksInvalidation(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.txt")
	output := filepath.Join(dir, "output.txt")
	if err := os.WriteFile(input, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	task := config.Task{
		Name:    "generate",
		Stage:   "pre-build",
		Command: "printf generated > " + output,
		Inputs:  []string{input},
		Outputs: []string{output},
	}
	if err := runTasks(context.Background(), []config.Task{task}, false, 1); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != "generated" {
		t.Fatalf("output = %q", first)
	}
	firstInfo, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := runTasks(context.Background(), []config.Task{task}, false, 1); err != nil {
		t.Fatal(err)
	}
	secondInfo, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if !secondInfo.ModTime().Equal(firstInfo.ModTime()) {
		t.Fatal("unchanged task was executed")
	}
	if err := os.WriteFile(input, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runTasks(context.Background(), []config.Task{task}, false, 1); err != nil {
		t.Fatal(err)
	}
}

func TestRunTasksStagesAndDependencies(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first")
	second := filepath.Join(dir, "second")
	tasks := []config.Task{
		{Name: "a", Stage: "pre-build", Command: "printf a > " + first, Outputs: []string{first}},
		{Name: "b", Stage: "pre-build", Command: "cat " + first + " > " + second, Inputs: []string{first}, Outputs: []string{second}},
	}
	if err := runTasks(context.Background(), tasks, false, 2); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "a" {
		t.Fatalf("dependency output = %q", data)
	}
}
