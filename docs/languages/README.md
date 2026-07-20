# Languages and source handling

ForgeZero is not a new compiler for every supported language. Instead, it acts as a build orchestrator that routes each input to the right backend.

## Supported language families

- Assembly: NASM, GAS, FASM, and related dialects.
- C / C++ / Objective-C: routed through the relevant compiler frontends.
- Gloria: a small custom language implemented in the project itself.

## How dispatch works

A source file is detected by its extension and mapped to an appropriate compiler or assembler. The build engine then compiles the translation unit and feeds the resulting objects into the link stage.

## Why this approach?

This keeps the user-facing model simple while preserving interoperability with mature toolchains. It also makes the system easier to evolve: new languages can be added as new backends rather than forcing everything into one giant compiler implementation.
