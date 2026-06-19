#!/usr/bin/env bash
set -euo pipefail

TEST_DIR="cmake_bench"
NUM_MODULES=1000
GENERATOR="${1:-Unix Makefiles}"

for cmd in hyperfine cmake gcc fz; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "missing: $cmd"; exit 1; }
done

if [[ "$GENERATOR" == "Ninja" ]] && ! command -v ninja &>/dev/null; then
  echo "missing: ninja"; exit 1
fi

echo "Generating $NUM_MODULES modules..."
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR/src"

for i in $(seq 1 "$NUM_MODULES"); do
  cat > "$TEST_DIR/src/mod_$i.c" <<EOF
int func_$i(void) { return $i; }
EOF
done

cat > "$TEST_DIR/src/main.c" <<'EOF'
int main(void) { return 0; }
EOF

cat > "$TEST_DIR/CMakeLists.txt" <<EOF
cmake_minimum_required(VERSION 3.10)
project(CMakeBench C)
file(GLOB SOURCES "src/*.c")
add_executable(cmake_out \${SOURCES})
EOF

cd "$TEST_DIR"

echo "Verify fz..."
fz -dir . -out fz_out -toolchain zig  >/dev/null 2>&1
rm -rf fz_out .fz_objs

echo "Verify cmake..."

cmake -G "$GENERATOR" -B build . >/dev/null 2>&1
cmake --build build -j $(nproc) >/dev/null 2>&1
rm -rf build out_cmake

echo "Benchmark..."
hyperfine --warmup 3 \
  --prepare "rm -rf .fz_objs fz_out" \
  "fz -dir . -out fz_out -toolchain clang " \
  --prepare "rm -rf build && cmake -G \"$GENERATOR\" -B build . >/dev/null 2>&1" \
  "cmake --build build -j $(nproc)"

cd ..
echo "done"
