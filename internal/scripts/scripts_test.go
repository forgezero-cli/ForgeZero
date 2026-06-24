// SPDX-LICENSE-INDITIFIER MIT
// AUTHOR: ALEXVOSTE

package scripts

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestScriptsConfigureRunEmptyCommands(t *testing.T) {
	oldRun := runCommand
	defer func() { runCommand = oldRun }()

	s := &ScriptsConfigure{
		Commands: []string{},
		Verbose:  false,
	}

	if err := s.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestScriptsConfigureRunSuccessVerboseFalse(t *testing.T) {
	oldRun := runCommand
	defer func() { runCommand = oldRun }()

	runCommand = func(ctx context.Context, verbose bool, stdout, stderr io.Writer, name string, args ...string) (string, error) {
		if verbose {
			t.Error("expected verbose=false")
		}
		return "", nil
	}

	s := &ScriptsConfigure{
		Commands: []string{"echo hello", "true"},
		Verbose:  false,
	}

	if err := s.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestScriptsConfigureRunSuccessVerboseTrue(t *testing.T) {
	oldRun := runCommand
	defer func() { runCommand = oldRun }()

	var output []string
	runCommand = func(ctx context.Context, verbose bool, stdout, stderr io.Writer, name string, args ...string) (string, error) {
		if !verbose {
			t.Error("expected verbose=true")
		}

		output = append(output, name+" "+args[0])
		return "", nil
	}

	s := &ScriptsConfigure{
		Commands: []string{"echo hello", "true"},
		Verbose:  true,
	}

	if err := s.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(output) != 2 {
		t.Fatalf("expected 2 commands executed, got %d", len(output))
	}
}

func TestScriptsConfigureRunCommandFailure(t *testing.T) {
	oldRun := runCommand
	defer func() { runCommand = oldRun }()

	runCommand = func(ctx context.Context, verbose bool, stdout, stderr io.Writer, name string, args ...string) (string, error) {
		return "", errors.New("command failed")
	}

	s := &ScriptsConfigure{
		Commands: []string{"false"},
		Verbose:  false,
	}

	err := s.Run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestScriptsConfigureRunMultipleCommandsFailureStops(t *testing.T) {
	oldRun := runCommand
	defer func() { runCommand = oldRun }()

	calls := 0
	runCommand = func(ctx context.Context, verbose bool, stdout, stderr io.Writer, name string, args ...string) (string, error) {
		calls++
		if calls == 1 {
			return "", nil
		}
		return "", errors.New("second command failed")
	}

	s := &ScriptsConfigure{
		Commands: []string{"true", "false", "true"},
		Verbose:  false,
	}

	err := s.Run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}
