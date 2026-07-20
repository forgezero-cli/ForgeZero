# Examples

The project includes examples for simple C/assembly builds and for experimentation with Gloria and the FZP preprocessor. The examples are useful because they show how the same build system can support both traditional code and more unusual low-level workflows.

## Example 1: simple assembly build

A small assembly project can be built directly by pointing ForgeZero at a source directory or a single file.

## Example 2: toolchain customization

A TOML configuration can inject compiler flags, linker flags, and custom hooks without changing the build system itself.

## Example 3: custom preprocessing

The built-in preprocessor can drive conditional includes and macro definitions before the downstream toolchain sees the file.
