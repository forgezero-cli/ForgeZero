# Gloria

Gloria is a small custom language implemented inside the ForgeZero project. It is not intended to replace C or assembly for general-purpose work. Instead, it is a compact, low-level language for situations where direct control over generated code is useful.

## Syntax model

The implementation currently understands a minimal set of constructs:

- `fn name(args) { ... }` for functions.
- `let name = value` for local bindings.
- `return` to exit a function.
- `reg name` for register declarations in the current implementation model.
- `if` and `while` for control flow.
- builtins such as `out8` and `in8` for simple I/O-style operations.

## Example

```gloria
fn main() {
    let x = 1
    let y = 2
    return
}
```

## Implementation notes

The Gloria pipeline is a lexer, parser-like token handling layer, and an emitter that generates x86-64 machine code. It is intentionally lightweight and experimental. The current implementation is best thought of as a compact codegen exercise rather than a full general-purpose language runtime.

## Why it exists

Gloria exists to explore how much control a build system can expose without abandoning the convenience of a unified workflow. It is a good fit for experiments, low-level tooling, and code generation scenarios where a tiny custom language is easier to reason about than a full compiler frontend.
