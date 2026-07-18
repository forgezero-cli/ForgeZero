# ForgeZero — Benchmark Video

### New link to a youtube video showing the build process of a redis database.
[View on Youtube](https://www.youtube.com/watch?v=4mWq0VcKFO8)

[Watch on YouTube](https://www.youtube.com/watch?v=0mbejIDcxsw)

## Description

Full recorded benchmark run comparing ForgeZero (`fz`) against a
traditional `nasm -f elf64 && ld` / `make -j4` pipeline. This is the
same run backing the scaling benchmark table in the project
documentation: 20–400 modules, Intel Core i5-10310U (4C/8T, 1.7 GHz
base), Arch Linux, Samsung 980 NVMe, timed with `hyperfine` over 10 or
more runs per data point, showing a 2.35x–4.95x measured speedup for
`fz` over `make -j4` as module count increases.

The linked call-to-action text describes it as showing a 4.5x speedup.

## Reproducing it yourself

```bash
git clone https://github.com/forgezero-cli/ForgeZero
cd ForgeZero
chmod +x build.sh && ./build.sh
# or
bash build.sh # or sh build.sh

./bench.sh   # edit NUM_MODULES in the script to change scale

hyperfine --warmup 3 --prepare 'make clean && rm -rf .fz_objs fz_out' \
  './fz -dir . -out fz_out' 'make -j4' \
  --export-markdown results.md
```

