# Quick start

The fastest way to understand ForgeZero is to build a tiny project and observe the pipeline.

## Step 1: initialize a project

```bash
fz -init
```

This creates a starter configuration and a small README scaffold.

## Step 2: build a directory

```bash
fz -dir ./src -out app
```

## Step 3: inspect the output

If the build succeeds, ForgeZero writes the requested binary and leaves behind any requested intermediate artifacts based on the configuration.

## Why this feels simple

The quick start path is intentionally short. The project wants the first successful build to be easy, while the advanced configuration remains available when the system grows more complex.
