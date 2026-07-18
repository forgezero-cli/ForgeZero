/*
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

package bashrun

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func RunInline(ctx context.Context, script string, verbose bool) error {
	script = strings.TrimSpace(script)
	if script == "" {
		return nil
	}
	if path, err := utils.FindExecutable(ctx, "bash"); err == nil {
		return runWithShell(ctx, path, script, verbose)
	}
	if path, err := utils.FindExecutable(ctx, "sh"); err == nil {
		return runWithShell(ctx, path, script, verbose)
	}
	return runInternal(script, verbose)
}

func runWithShell(ctx context.Context, shellPath, script string, verbose bool) error {
	tmp, err := os.CreateTemp("", "fz-script-*.sh")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.WriteString(script); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	tmp.Close()
	if err := os.Chmod(name, 0o700); err != nil {
		os.Remove(name)
		return err
	}
	defer os.Remove(name)

	cmd := exec.CommandContext(ctx, shellPath, name)
	cmd.Dir, _ = os.Getwd()
	cmd.Env = os.Environ()
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

func runInternal(script string, verbose bool) error {
	lines := strings.Split(script, "\n")
	cwd, _ := os.Getwd()
	env := os.Environ()
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		if strings.HasPrefix(ln, "cd ") {
			arg := strings.TrimSpace(strings.TrimPrefix(ln, "cd "))
			if !filepath.IsAbs(arg) {
				arg = filepath.Join(cwd, arg)
			}
			cwd = arg
			continue
		}
		if strings.HasPrefix(ln, "export ") {
			kv := strings.TrimSpace(strings.TrimPrefix(ln, "export "))
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) == 2 {
				k := parts[0]
				v := parts[1]
				env = ensureEnv(env, k, v)
			}
			continue
		}
		parts := strings.Fields(ln)
		if len(parts) == 0 {
			continue
		}
		bin := parts[0]
		args := parts[1:]
		cmd := exec.CommandContext(context.Background(), bin, args...)
		cmd.Dir = cwd
		cmd.Env = env
		if verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		if err := cmd.Run(); err != nil {
			return errors.New("command failed: " + ln + ": " + err.Error())
		}
	}
	return nil
}

func ensureEnv(env []string, k, v string) []string {
	out := make([]string, 0, len(env)+1)
	found := false
	for _, e := range env {
		if strings.HasPrefix(e, k+"=") {
			out = append(out, k+"="+v)
			found = true
			continue
		}
		out = append(out, e)
	}
	if !found {
		out = append(out, k+"="+v)
	}
	return out
}
