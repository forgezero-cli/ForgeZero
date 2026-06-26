#!/usr/bin/env bash
set -euo pipefail

TEST_DIR="benchmark_test"
NUM_MODULES=2500 


echo "🛠️  ForgeZero vs NASM/Make Benchmark Script"
echo "=========================================="

for cmd in hyperfine nasm ld make; do
  command -v $cmd >/dev/null 2>&1 || {
    echo "❌ Missing: $cmd"
    exit 1
  }
done
if [[ ! -x "./fzt" ]]; then
  echo "❌ ./fzt not found or not executable. Run this from ForgeZero root."
  exit 1
fi

echo "🧹 Preparing environment..."
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR/src"

echo " Generating $NUM_MODULES test modules..."
for i in $(seq 1 $NUM_MODULES); do
  cat >"$TEST_DIR/src/mod_$i.asm" <<EOF
global mod_func_$i
section .text
mod_func_$i:
    mov rax, $i
    ret
EOF
done

cat >"$TEST_DIR/src/main.asm" <<'EOF'
global _start
section .text
_start:
    mov rax, 42
    mov rdi, rax
    mov rax, 60
    syscall
EOF

echo "📝 Generating Makefile..."
{
  echo 'SRCS := $(wildcard src/*.asm)'
  echo 'OBJS := $(SRCS:.asm=.o)'
  echo 'OUT := out_nasm'
  echo 'all: $(OUT)'
  echo '$(OUT): $(OBJS)'
  printf '\tld $^ -o $@\n'
  echo '%.o: %.asm'
  printf '\tnasm -f elf64 $< -o $@\n'
  echo 'clean:'
  printf '\trm -f $(OBJS) $(OUT)\n'
  echo '.PHONY: all clean'
} >"$TEST_DIR/Makefile"

echo "🔍 Verifying builds (dry-run)..."
cd "$TEST_DIR"

echo "  → ForgeZero..."
if ! ../fzt -dir . -out fz_out -verbose; then
  echo "❌ ForgeZero build failed. Check output above."
  exit 1
fi
rm -rf fz_out .fz_objs

echo "  → Make/NASM..."
if ! make -j4; then
  echo "❌ Make build failed. Check nasm/ld installation."
  exit 1
fi
rm -f out_nasm *.o
cd ..

echo ""
echo "⏱️  Running hyperfine benchmark..."
echo "   (Warmup: 3 runs | Runs: ~10 per command)"
cd "$TEST_DIR"

hyperfine --warmup 3 \
  --prepare "make clean && rm -rf .fz_objs fz_out" \
  "fz -profile perfomance -dir . -out fz_out -j $(nproc)" \
  "make -j $(nproc)" \

cd ..

echo ""
echo "🏁 Done. Results printed above."
echo "💡 Tip: Add '--export-markdown results.md' to hyperfine to save output."
