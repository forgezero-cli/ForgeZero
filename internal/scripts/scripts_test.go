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

	var commandsCalled int
	runCommand = func(ctx context.Context, verbose bool, stdout, stderr io.Writer, name string, args ...string) (string, error) {
		commandsCalled++
		return "", nil
	}

	s := &ScriptsConfigure{
		Commands: []string{"echo hello", "true"},
		Verbose:  true,
	}

	if err := s.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if commandsCalled != 2 {
		t.Fatalf("expected 2 commands executed, got %d", commandsCalled)
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

func TestScriptsConfigureRunBashInline(t *testing.T) {
	oldInline := bashInline
	defer func() { bashInline = oldInline }()

	var gotScript string
	bashInline = func(ctx context.Context, script string, verbose bool) error {
		gotScript = script
		if !verbose {
			t.Fatal("expected verbose true")
		}
		return nil
	}

	s := &ScriptsConfigure{
		Commands: []string{"bash:echo hello"},
		Verbose:  true,
	}

	if err := s.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotScript != "echo hello" {
		t.Fatalf("expected bash script body passed through, got %q", gotScript)
	}
}

func TestScriptsConfigureRunBashInlineError(t *testing.T) {
	oldInline := bashInline
	defer func() { bashInline = oldInline }()

	bashInline = func(ctx context.Context, script string, verbose bool) error {
		return errors.New("bash failure")
	}

	s := &ScriptsConfigure{
		Commands: []string{"bash:echo hello"},
		Verbose:  false,
	}

	err := s.Run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
