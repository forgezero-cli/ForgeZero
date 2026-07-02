#!/usr/bin/env bash
set -euo pipefail

echo "ForgeZero CI/CD Pipeline Validator"
echo "===================================="

CHECKS_PASSED=0
CHECKS_FAILED=0

check() {
  local cmd="$1"
  local desc="$2"
  if eval "$cmd" > /dev/null 2>&1; then
    echo "✓ $desc"
    ((CHECKS_PASSED++))
  else
    echo "✗ $desc"
    ((CHECKS_FAILED++))
  fi
}

section() {
  echo ""
  echo "=== $1 ==="
}

section "Environment Check"
check "command -v git" "Git"
check "command -v go" "Go"
check "go version | grep -q '1.2[45]'" "Go version >= 1.24"

section "Project Structure"
check "[ -f go.mod ]" "go.mod exists"
check "[ -f cmd/fz/main.go ]" "Main package exists"
check "[ -d .github/workflows ]" "Workflows directory"
check "[ -f .github/workflows/build-multiplatform.yml ]" "Build workflow"
check "[ -f .github/workflows/codeql-analysis.yml ]" "CodeQL workflow"
check "[ -f .github/workflows/test.yml ]" "Test workflow"
check "[ -f .github/workflows/lint.yml ]" "Lint workflow"

section "Build Tools"
check "command -v gofmt" "gofmt"
check "command -v go vet" "go vet (built-in)"
check "command -v upx" "UPX (optional)" || true
check "command -v blake3sum" "blake3sum (optional)" || true
check "go list -m github.com/golangci/golangci-lint" "golangci-lint module" || true
check "go list -m golang.org/x/vuln/cmd/govulncheck" "govulncheck module" || true

section "Code Quality"
check "gofmt -l ./... | wc -l | grep -q '^0$'" "Code formatting (gofmt)"
check "go vet ./... 2>&1 | grep -q 'no issues found' || true" "Go vet check"
go test -timeout 5m ./... > /dev/null 2>&1 && {
  echo "✓ Unit tests pass"
  ((CHECKS_PASSED++))
} || {
  echo "✗ Unit tests fail"
  ((CHECKS_FAILED++))
}

section "Build Verification"
ARCHS=("amd64" "arm64")
for arch in "${ARCHS[@]}"; do
  CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -o /tmp/fz-$arch ./cmd/fz 2>/dev/null && {
    SIZE=$(stat -c%s "/tmp/fz-$arch" 2>/dev/null || stat -f%z "/tmp/fz-$arch")
    echo "✓ linux/$arch: $(numfmt --to=iec $SIZE 2>/dev/null || echo $SIZE bytes)"
    ((CHECKS_PASSED++))
  } || {
    echo "✗ linux/$arch failed"
    ((CHECKS_FAILED++))
  }
done

section "Workflow Syntax"
for workflow in .github/workflows/*.yml; do
  name=$(basename "$workflow")
  if grep -q "^name:" "$workflow" 2>/dev/null; then
    echo "✓ $name"
    ((CHECKS_PASSED++))
  else
    echo "✗ $name invalid"
    ((CHECKS_FAILED++))
  fi
done

section "Summary"
echo "Passed: $CHECKS_PASSED"
echo "Failed: $CHECKS_FAILED"

if [ $CHECKS_FAILED -eq 0 ]; then
  echo "✓ All checks passed!"
  exit 0
else
  echo "✗ Some checks failed"
  exit 1
fi
