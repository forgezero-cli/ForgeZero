# FZP preprocessor

ForgeZero includes a lightweight preprocessor named FZP. It is not a full C preprocessor replacement, but it covers the subset that is most useful for build-time configuration and conditional inclusion.

## Supported directives

- `#define`: create or overwrite a macro.
- `#undef`: remove a macro.
- `#include`: include another file if it resolves inside an allowed path.
- `#ifdef`, `#ifndef`, `#if`, `#elif`, `#else`, `#endif`: conditionally emit content.
- `#error`: stop preprocessing with a clear failure.
- `#pragma`: ignored by the current implementation.

## Expression support

The conditional parser supports simple boolean and numeric expressions using operators such as `&&`, `||`, `!`, comparison operators, and arithmetic. It also understands `defined(NAME)`.

## Why this exists

The preprocessor is a small, controlled mechanism for making source trees adaptable to different build contexts. It is deliberately limited so that the semantics stay understandable and the implementation remains maintainable.
