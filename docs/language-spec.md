# Idiotic Language Specification

## Values

Idiotic supports 64-bit integers, strings, booleans, and `null`.

The language is dynamically typed. `false` and `null` are falsey; all other values are truthy.

## Variables

Variables are declared with `let`:

```idio
let name = "Idiotic";
name = "New name";
```

Assignments require an existing variable. Lexical scopes allow inner environments to read outer variables and assign to the nearest captured binding.

## Functions

```idio
fn add(a, b) {
    return a + b;
}
```

Functions are first-class values and support recursion and closures.
At most 1024 user-function calls may be active at once; exceeding this limit returns a runtime error.

## Expressions

Arithmetic operators are `+`, `-`, `*`, `/`. Comparisons are `==`, `!=`, `<`, `<=`, `>`, `>=`. Logical operators are `&&` and `||`; they short-circuit, so the right operand is evaluated only when needed.

Unary operators are `-` and `!`.

## Control Flow

```idio
if (condition) {
    // ...
} else {
    // ...
}

while (condition) {
    // ...
}
```

Line comments begin with `//`.

The REPL preserves variables between inputs, accepts multiline expressions and blocks, and prints non-null expression results. Use `:help`, `:reset`, and `:quit` for REPL commands.

## Implementations

The AST interpreter is the default for the CLI and REPL. Use `idiotic run --vm`
to execute a script with the bytecode compiler and VM. The REPL continues to use
the AST interpreter.

## Not yet supported

Idiotic does not implement `for` loops, arrays, `break`, or `continue`.

## Built-ins

- `print(value, ...)` prints values and a trailing newline.
- `len(string)` returns the number of Unicode characters.

Errors are reported as parser or runtime diagnostics; malformed programs should not cause uncontrolled panics.
