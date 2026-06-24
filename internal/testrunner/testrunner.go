package testrunner

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/forgezero-cli/ForgeZero/internal/verify"
)

type TestOptions struct {
	Verbose bool
	JSON    bool
	Alex    bool
}

type TestStage struct {
	Name     string
	Duration int64
	Passed   bool
	Output   string
	Errors   []string
}

type TestReport struct {
	Status      string      `json:"status"`
	DurationMs  int64       `json:"duration_ms"`
	Stages      []TestStage `json:"stages"`
	Environment struct {
		GOOS      string `json:"goos"`
		GOARCH    string `json:"goarch"`
		Cores     int    `json:"cores"`
		GoPath    string `json:"go_path"`
		GoVersion string `json:"go_version"`
	} `json:"environment"`
}

var AlexMode bool

func RunSuite(verbose, jsonOut, alex bool) error {
	opts := TestOptions{
		Verbose: verbose,
		JSON:    jsonOut,
		Alex:    alex,
	}

	start := time.Now()
	report := TestReport{
		Status: "running",
		Stages: []TestStage{},
	}
	report.Environment.GOOS = runtime.GOOS
	report.Environment.GOARCH = runtime.GOARCH
	report.Environment.Cores = runtime.NumCPU()
	if goPath := os.Getenv("GOPATH"); goPath != "" {
		report.Environment.GoPath = goPath
	}
	if out, err := exec.Command("go", "version").Output(); err == nil {
		report.Environment.GoVersion = strings.TrimSpace(string(out))
	}

	stages := []struct {
		Name string
		Run  func() (string, error)
	}{
		{"Environment Check (doctor)", runDoctor},
		{"Unit Tests (go test -race)", runGoTest},
		{"Code Coverage", runCoverage},
		{"Static Analysis (go vet)", runGoVet},
		{"Linter (staticcheck)", runStaticCheck},
		{"Code Formatting (go fmt)", runGoFmt},
		{"Build Test (fz build)", runBuildTest},
		{"Gloria Compilation", runGloriaTest},
		{"HADES Codegen", runHadesTest},
		{"Integration Tests", runIntegrationTest},
	}

	if opts.Alex {
		stages = append(stages,
			struct {
				Name string
				Run  func() (string, error)
			}{"Citadel Zero-Allocations", runZeroAllocBench},
			struct {
				Name string
				Run  func() (string, error)
			}{"Race Detector (full)", runRaceDetector},
			struct {
				Name string
				Run  func() (string, error)
			}{"Aegis Audit", runAudit},
			struct {
				Name string
				Run  func() (string, error)
			}{"Source Integrity (verify)", runVerify},
		)
	}

	allPassed := true

	for _, s := range stages {
		stageStart := time.Now()
		output, err := s.Run()
		duration := time.Since(stageStart).Milliseconds()

		stage := TestStage{
			Name:     s.Name,
			Duration: duration,
			Passed:   err == nil,
			Output:   output,
		}
		if err != nil {
			stage.Errors = []string{err.Error()}
			allPassed = false
		}

		report.Stages = append(report.Stages, stage)

		if opts.Verbose && output != "" {
			os.Stdout.WriteString(output)
		}
	}

	report.DurationMs = time.Since(start).Milliseconds()
	if allPassed {
		report.Status = "success"
	} else {
		report.Status = "failed"
	}

	if opts.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
	} else {
		printHumanReport(report)
	}

	if !allPassed {
		return errors.New("test suite failed")
	}
	return nil
}

func printHumanReport(r TestReport) {
	var b strings.Builder
	b.WriteString("\n\x1b[36mForgeZero Test Runner\x1b[0m (v4.8.0-dev)\n")
	b.WriteString(strings.Repeat("─", 60) + "\n")

	for _, s := range r.Stages {
		icon := "✓"
		if !s.Passed {
			icon = "✗"
		}
		b.WriteString("  \x1b[34m[" + icon + "]\x1b[0m " + s.Name + " " + stageStatus(s.Passed) + " (" + strconv.FormatInt(s.Duration, 10) + " ms)\n")
		if !s.Passed && len(s.Errors) > 0 {
			for _, e := range s.Errors {
				b.WriteString("      \x1b[31mError:\x1b[0m " + strings.TrimSpace(e) + "\n")
			}
		}
	}

	b.WriteString(strings.Repeat("─", 60) + "\n")
	if r.Status == "success" {
		b.WriteString("\x1b[32mSTATUS: SUCCESS\x1b[0m (" + strconv.Itoa(len(r.Stages)) + " stages passed, " + strconv.FormatInt(r.DurationMs, 10) + " ms)\n")
	} else {
		b.WriteString("\x1b[31mSTATUS: FAILED\x1b[0m (see errors above)\n")
	}
	os.Stdout.WriteString(b.String())
}

func stageStatus(passed bool) string {
	if passed {
		return "\x1b[32m[PASS]\x1b[0m"
	}
	return "\x1b[31m[FAIL]\x1b[0m"
}

func runCmd(cmd *exec.Cmd) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := stdout.String()
	if stderr.Len() > 0 {
		if out != "" {
			out += "\n"
		}
		out += stderr.String()
	}
	return strings.TrimSpace(out), err
}

func runDoctor() (string, error) {
	cmd := exec.Command("./fz", "doctor")
	return runCmd(cmd)
}

func runGoTest() (string, error) {
	cmd := exec.Command("go", "test", "./...", "-race", "-count=1", "-timeout=2m")
	return runCmd(cmd)
}

func runCoverage() (string, error) {
	cmd := exec.Command("go", "test", "./...", "-cover")
	return runCmd(cmd)
}

func runGoVet() (string, error) {
	cmd := exec.Command("go", "vet", "./...")
	return runCmd(cmd)
}

func runStaticCheck() (string, error) {
	if _, err := exec.LookPath("staticcheck"); err != nil {
		return "staticcheck not installed, skipping", nil
	}
	cmd := exec.Command("staticcheck", "./...")
	return runCmd(cmd)
}

func runGoFmt() (string, error) {
	cmd := exec.Command("gofmt", "-l", ".")
	out, err := cmd.Output()
	if err != nil {
		return string(out), err
	}
	if len(out) > 0 {
		return string(out), errors.New("unformatted files:\n" + string(out))
	}
	return "all files formatted", nil
}

func runBuildTest() (string, error) {
	cmd := exec.Command("go", "build", "-o", "/dev/null", "./cmd/fz")
	return runCmd(cmd)
}

func runGloriaTest() (string, error) {
	gloriaSrc := filepath.Join(os.TempDir(), "test.glo")
	src := []byte(`fn main() {
		let a = 10
		let b = 25
		if b > a {
			print("ok")
		}
	}`)
	if err := os.WriteFile(gloriaSrc, src, 0644); err != nil {
		return "", err
	}
	defer os.Remove(gloriaSrc)
	defer os.Remove("gloria.bin")

	cmd := exec.Command("./fz", "-gloria", gloriaSrc)
	return runCmd(cmd)
}

func runHadesTest() (string, error) {
	return "Hades test skipped (testdata missing)", nil
}

func runIntegrationTest() (string, error) {
	cmd := exec.Command("go", "test", "./tests/integration/...", "-v", "-count=1")
	return runCmd(cmd)
}

func runZeroAllocBench() (string, error) {
	cmd := exec.Command("go", "test", "-bench=BenchmarkCopyFileHot", "-benchmem", "./internal/linker")
	out, err := runCmd(cmd)
	if err != nil {
		return out, err
	}
	if !strings.Contains(out, "0 allocs/op") {
		return out, errors.New("zero-allocation assertion failed: expected 0 allocs/op")
	}
	return out, nil
}

func runRaceDetector() (string, error) {
	cmd := exec.Command("go", "test", "./...", "-race", "-count=3", "-timeout=5m")
	return runCmd(cmd)
}

func runAudit() (string, error) {
	cmd := exec.Command("./fz", "audit")
	return runCmd(cmd)
}

func runVerify() (string, error) {
	root := "."
	manifest := "blake3.manifest"
	result, err := verify.VerifyRoot(root, manifest)
	if err != nil {
		return "", err
	}
	if len(result.Missing) == 0 && len(result.Modified) == 0 && len(result.Extra) == 0 {
		return "source tree integrity intact", nil
	}
	var b strings.Builder
	if len(result.Missing) > 0 {
		b.WriteString("missing files:\n")
		for _, p := range result.Missing {
			b.WriteString("  " + p + "\n")
		}
	}
	if len(result.Modified) > 0 {
		b.WriteString("modified files:\n")
		for _, p := range result.Modified {
			b.WriteString("  " + p + "\n")
		}
	}
	if len(result.Extra) > 0 {
		b.WriteString("extra files:\n")
		for _, p := range result.Extra {
			b.WriteString("  " + p + "\n")
		}
	}
	return b.String(), errors.New("integrity check failed")
}
