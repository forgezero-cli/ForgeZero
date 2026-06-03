# TODO

This document outlines the current technical direction of Gloria and identifies areas where contributions are welcome.

## High Priority

### Expression Parser

The current compiler handles arithmetic through ad-hoc parsing in the code generation phase. This approach does not scale well as the language grows.

Goal:

* Introduce a dedicated expression parser.
* Implement proper operator precedence.
* Support nested expressions and parentheses.
* Remove special-case arithmetic handling from `CompileFunc`.

Example:

```gloria
let a = 10 + 20
let b = a * 4
let c = (a + b) / 2

return c
```

Suggested approach:

* Pratt parser or precedence-climbing parser.
* Separate expression parsing from code generation.
* Generate an expression AST before lowering to machine code.

---

### Expression AST

Introduce a dedicated AST layer for expressions.

Potential structure:

```go
type Expr interface{}

type IntExpr struct{}
type VarExpr struct{}
type BinaryExpr struct{}
type UnaryExpr struct{}
type CallExpr struct{}
```

This will significantly simplify future language features.

---

### Additional Arithmetic Operations

Planned operators:

* `*`
* `/`
* `%`
* `&`
* `|`
* `^`
* `<<`
* `>>`

Backend support is required in the x86-64 emitter.

---

### Function Calls Inside Expressions

Current function calls are primarily handled as statements or return expressions.

Target:

```gloria
let value = read_port(0x60) + 1

return foo(a) + bar(b)
```

---

## Medium Priority

### Statement AST

The current compiler directly emits code while parsing statements.

Long-term goal:

```go
type Stmt interface{}

type LetStmt struct{}
type AssignStmt struct{}
type ReturnStmt struct{}
type IfStmt struct{}
type WhileStmt struct{}
type CallStmt struct{}
```

This would make the frontend significantly easier to maintain.

---

### Improved Conditional Expressions

Current conditions are intentionally minimal.

Future support:

```gloria
if a == b { }

if a < b { }

if a > b { }

if (a + b) < c { }

if foo(x) != 0 { }
```

---

### Improved While Conditions

Future support:

```gloria
while counter > 0 {
    counter -= 1
}
```

instead of relying on simple truthy variable checks.

---

### Constant Folding

Compile-time evaluation of constant expressions.

Example:

```gloria
let x = 2 + 3 * 4
```

should become:

```gloria
let x = 14
```

during compilation.

---

### Peephole Optimization Pass

Current peephole optimization is intentionally minimal.

Potential improvements:

* Constant propagation
* Dead move elimination
* Redundant load/store elimination
* Simple register coalescing

---

## Long-Term Goals

### Intermediate Representation (IR)

Introduce a backend-independent IR.

Proposed architecture:

```text
Lexer
  ↓
Parser
  ↓
AST
  ↓
IR
  ↓
Backend
```

Benefits:

* Cleaner optimization pipeline
* Multiple backends
* Easier testing

---

### Multiple Backend Targets

Potential future targets:

* x86-64
* ARM64
* WebAssembly

---

### Better Register Allocation

Current code generation relies heavily on stack temporaries.

Future work:

* Virtual registers
* Liveness analysis
* Register allocation
* Reduced memory traffic

---

### Testing Infrastructure

Needed:

* Lexer tests
* Parser tests
* AST tests
* Code generation tests
* Bare-metal integration tests

Contributions in this area are highly appreciated.

---

## Notes

The immediate focus is the expression parser and expression AST. Most upcoming language features depend on this work, and it will significantly reduce complexity inside the current code generation pipeline.

