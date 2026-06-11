# Contributing to fz(ForgeZero)

> The assembly swiss army knife — built with discipline, shipped with intent.

Thank you for considering a contribution to `fz`. This document establishes the standards and workflow expected of all contributors. Please read it in full before opening issues or submitting pull requests.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Building and Testing](#building-and-testing)
- [Code Standards](#code-standards)
- [Commit Convention](#commit-convention)
- [Pull Request Process](#pull-request-process)
- [Proposing New Features](#proposing-new-features)
- [Reporting Issues](#reporting-issues)
- [License](#license)

---

## Code of Conduct

All contributors are expected to engage professionally and constructively. Disrespectful, dismissive, or hostile behaviour will not be tolerated. We are here to build good software together.

---

## Getting Started

Contributions are welcome in the following forms:

- **Bug reports** — reproducible, well-documented issues filed via [GitHub Issues](https://github.com/forgezero-cli/ForgeZero/issues)
- **Feature proposals** — opened as issues before any implementation begins
- **Pull requests** — bug fixes, features, refactors, or documentation improvements
- **Documentation** — corrections, clarifications, and examples

### Before submitting

Run the following commands locally. If any of them fails, fix it before opening the PR:

```bash
go fmt ./...
go mod tidy
go test -race ./...
go vet ./...
staticcheck ./...
```

---


## Development Setup

### Prerequisites

| Requirement | Version |
|-------------|---------|
| Go          | ≥ 1.21  |
| NASM        | any recent |
| GCC + Binutils | any recent |
| Clang *(optional)* | for sanitizer tests |

### Clone

```bash
git clone https://github.com/forgezero-cli/ForgeZero.git
cd ForgeZero
```

### Install system dependencies

```bash
# Required for assembly tests
sudo apt install nasm gcc binutils

# Optional: strict sanitizer tests
sudo apt install clang
```

---

## Building and Testing

### Build

```bash
go build -o fz ./cmd/fz
```

### Run the full test suite

```bash
go test ./... -cover
```

### Run with race detector *(required before submitting a PR)*

```bash
go test -race ./...
```

### Static analysis

Install `staticcheck` if not already present:

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
```

Then run:

```bash
go vet ./...
staticcheck ./...
```

All checks must pass with zero warnings before a pull request is considered ready for review.

---

## Code Standards

- **Formatting** — run `go fmt ./...` before every commit; no exceptions
- **Functions** — keep them small, focused, and with a single clear responsibility
- **Naming** — use meaningful, unambiguous names; avoid abbreviations unless they are idiomatic in Go
- **Comments** — omit comments where the code is self-explanatory; add them only where intent cannot be inferred from reading the code
- **Coverage** — new code must not decrease the overall test coverage percentage
- **Error handling** — every error must be handled explicitly; do not swallow errors silently

- **Tests must pass** — no pull request will be reviewed if `go test ./...` fails
- **No debugging code** — remove all `fmt.Println`, `log.Print`, or commented-out code before committing
- **No panic (except unrecoverable errors)** — use error returns where possible

## Mandatory Testing Requirements

Every pull request must include proof that all tests pass. **No exceptions.**

### Required Test Commands

Run these commands before submitting your PR:

```bash
# Full test suite with race detection (MUST pass)
go test -race ./...

# Verbose output for clarity (MUST pass)
go test -v ./...

# Performance benchmarks (must not regress)
go test -bench=. -run=^$ ./...
```

If any of these commands fails, your PR will be rejected automatically.

### Test Coverage for New Features

All new functionality must be placed in `internal/` with the following structure:

```text
internal/your_feature_name/
├── feature.go          # core logic
├── feature_test.go     # unit tests (MANDATORY)
└── ...                 # any auxiliary files
```

**Rule:** `feature_test.go` must exist. No tests = no merge.

### Running Tests for a Specific Feature

```bash
# Run tests for a specific feature
go test -v ./internal/your_feature_name/...

# Run a specific test function by name
go test -v -run TestFunctionName ./internal/your_feature_name/...
```

## Naming Conventions (STRICT)

### Functions

Every function name must clearly describe what it does.

**BAD:**

```go
func v(s string)
func proc(d []byte)
func f() int
```

**GOOD:**

```go
func PrintStringOnVGA(msg string) string
func ParseElfHeader(data []byte) (*ElfHeader, error)
func CalculateChecksum(buffer []byte) uint32
```

### Variables

Use descriptive names. Single-letter variables are forbidden except for short loops.

**BAD:**

```go
v := something...
x := process(d)
t := time.Now()
```

**GOOD:**

```go
message := something...
result := process(data)
currentTime := time.Now()
```

Exception: Loop indices (`i`, `j`, `k`) are allowed for short, simple loops.

### Packages / Feature Directories

| BAD | GOOD |
|------|------|
| `internal/x` | `internal/elf_parser` |
| `internal/utils` | `internal/string_utils` or `internal/file_utils` |
| `internal/feature` | `internal/watchdog` |

## Zero Allocation Policy

ForgeZero targets performance-critical environments. Avoid allocations where possible.

### Forbidden Patterns (if avoidable)

```go
message := fmt.Sprintf("error: %s", err)

var buf bytes.Buffer
buf.WriteString(data)

tmp := []string{...}
```

### Preferred Patterns

```go
writeFmt(2, "error: %s\n", err)

buf := bytes.NewBuffer(make([]byte, 0, 4096))

var bufferPool = sync.Pool{
    New: func() interface{} { return make([]byte, 4096) },
}
```

### Benchmarking Allocations

```bash
go test -bench=. -benchmem ./...
```

Look for `0 allocs/op` in the output for performance-critical functions.

## No Debug Prints

```bash
grep -r "fmt.Println\|fmt.Printf\|log.Print\|println(" internal/ cmd/ --include="*.go"
```

### Allowed

- `writeFmt()`
- `writeStderr()`
- Logging via `-verbose`

### Forbidden

- `fmt.Println`
- `log.Print` (outside tests)
- `println()`

## PR Checklist (Copy This)

- `go test -race ./...` passes
- `go test -v ./...` passes
- `go test -bench=. -run=^$ ./...` shows no regression
- New features include `_test.go`
- No single-letter variables (except loop indices)
- No debug prints
- `go fmt ./...` run
- `go vet ./...` and `staticcheck ./...` clean

Attach test output to your PR.


## Rules for External Contributors

To protect the project's long-term maintainability and code quality, the following rules apply:

- **No AI-generated code** — code that is obviously machine-generated without human review will be rejected. We can tell.
- **Plagiarism is forbidden** — copying code or documentation from other contributors (or from external sources) without attribution is grounds for immediate PR closure and a permanent ban.
- **You must build and test locally** — if you cannot build `fz` on your machine, do not open a PR. Figure out the build process first.
- **No "drive-by" PRs** — every PR must include tests and documentation (if applicable). Skeleton PRs without content will be closed.

Violations will be reported to GitHub if repeated.

---

## Commit Convention

This project follows the [Conventional Commits](https://www.conventionalcommits.org/) specification. All commit messages must conform to this format:

```
<type>: <short imperative summary>
```

| Type       | When to use                                      |
|------------|--------------------------------------------------|
| `feat`     | A new user-facing feature                        |
| `fix`      | A bug fix                                        |
| `test`     | Adding or improving tests                        |
| `docs`     | Documentation changes only                      |
| `refactor` | Code restructuring without behaviour change      |
| `chore`    | Tooling, CI, or dependency updates              |
| `perf`     | Performance improvements                         |

**Examples:**

```
feat: add --watch flag for incremental builds
fix: handle empty object files in linker stage
test: improve coverage for linker error paths
docs: document --output flag behaviour
refactor: extract runner into a standalone interface
```

Commit messages are part of the project's permanent history. Write them as if they will be read by someone debugging a regression two years from now — because they will be.

---

## Pull Request Process

1. **Fork** the repository and create a feature branch from `main`
2. **Write tests** — all new behaviour must be covered; failure paths are not optional
3. **Verify** — `go test -race ./...`, `go vet ./...`, and `staticcheck ./...` must all pass
4. **Update documentation** — if your change affects behaviour visible to users, update the relevant docs
5. **Open the PR** with a clear title (following commit convention) and a description that explains:
   - *What* changed
   - *Why* it was changed
   - Any relevant context or trade-offs

PRs that lack tests, break existing tests, or do not follow code standards will be returned for revision before review begins.

---

## Proposing New Features

Open an issue before writing any code. This is not a bureaucratic step — it is a practical one. A brief discussion upfront saves everyone time by confirming the feature aligns with the project's direction before implementation begins.

A good feature proposal includes:

- The problem being solved
- The proposed solution or interface
- Any known trade-offs or alternatives considered

---

## Reporting Issues

When filing a bug report, include the following:

- **`fz` version** — output of `fz -version`
- **Operating system and architecture** — e.g. `Linux amd64`, `macOS arm64`
- **Steps to reproduce** — minimal, complete, and unambiguous
- **Expected behaviour** — what should have happened
- **Actual behaviour** — what happened instead

Reports that cannot be reproduced from the information provided may be closed without action.

---

## License

By submitting a contribution, you agree that your work will be licensed under the [MIT License](./LICENSE) that covers this project.

---

*Thank you for taking the time to contribute to `fz`.*

**(c) alexvoste**
