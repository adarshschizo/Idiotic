# Idiotic

Idiotic is a small, dynamically typed programming language implemented in Go. It includes a tree-walking AST interpreter and an optional bytecode compiler and virtual machine.

## Features

- Integer, string, boolean, and `null` values
- Variables declared with `let` and assignment to existing variables
- Arithmetic, comparison, logical, and unary operators
- `if`/`else` and `while` control flow
- First-class functions, recursion, lexical scoping, and mutable closures
- A persistent REPL with multiline input and `:help`, `:reset`, and `:quit`
- `print` and `len` built-ins
- An opt-in bytecode VM for running scripts

The AST interpreter is the default engine. The REPL currently uses the AST interpreter; the VM is available for script execution with `run --vm`.

## Requirements

- Go 1.22 or later

## Quick start

From the project directory in Linux, macOS, or WSL:

```sh
go run ./cmd/idiotic --help
go run ./cmd/idiotic
go run ./cmd/idiotic run examples/hello.idio
go run ./cmd/idiotic run --vm examples/hello.idio
```

The interpreter and VM both print:

```text
Hello, Idiotic
```

To build a local executable:

```sh
go build -o bin/idiotic ./cmd/idiotic
./bin/idiotic run examples/hello.idio
```

### Windows PowerShell

```powershell
go build -o .\idiotic.exe .\cmd\idiotic
.\idiotic.exe run .\examples\hello.idio
.\idiotic.exe run --vm .\examples\hello.idio
.\idiotic.exe
```

## Language example

```idio
fn makeCounter(start) {
    let count = start;

    fn increment() {
        count = count + 1;
        return count;
    }

    return increment;
}

let counter = makeCounter(10);
print(counter()); // 11
print(counter()); // 12
```

See [`examples/`](examples/) for runnable programs and [`docs/language-spec.md`](docs/language-spec.md) for the language rules.

## Current limitations

`for` loops, arrays, `break`, and `continue` are not implemented. User-function recursion is limited to 1,024 active calls.

## Development

Run the test suite, race detector, static analysis, and build:

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./...
```

Run a bounded parser fuzz session and evaluator benchmarks:

```sh
go test ./internal/parser -fuzz FuzzParserDoesNotPanic -fuzztime 10s
go test ./internal/evaluator -bench "^BenchmarkEval" -benchmem
```

The GitHub Actions workflow runs formatting, tests, race detection, vet, and build checks on pushes and pull requests.

## License

This project is distributed under the MIT License. See [`LICENSE`](LICENSE).
