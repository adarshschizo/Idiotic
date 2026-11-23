package compiler

import (
	"strings"
	"testing"

	"idiotic/internal/evaluator"
	"idiotic/internal/lexer"
	"idiotic/internal/object"
	"idiotic/internal/parser"
	"idiotic/internal/vm"
)

func compileAndRun(t *testing.T, source string) object.Object {
	t.Helper()
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	compiler := New()
	if err := compiler.Compile(program); err != nil {
		t.Fatalf("compile error: %v", err)
	}
	machine := vm.New(compiler.Bytecode())
	if err := machine.Run(); err != nil {
		t.Fatalf("VM error: %v", err)
	}
	return machine.LastPoppedStackElem()
}

func TestCompilerArithmeticAndGlobals(t *testing.T) {
	got := compileAndRun(t, `let value = 1 + 2 * 3; value;`)
	if got.Inspect() != "7" {
		t.Fatalf("result = %q, want 7", got.Inspect())
	}
}

func TestCompilerRecordsRequiredGlobalCount(t *testing.T) {
	p := parser.New(lexer.New(`let first = 1; let second = first + 1;`))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	compiler := New()
	if err := compiler.Compile(program); err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if got := compiler.Bytecode().NumGlobals; got != 2 {
		t.Fatalf("bytecode global count = %d, want 2", got)
	}
}

func TestCompilerConditionals(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{name: "consequence", source: `if (true) { 10; } else { 20; }`, want: "10"},
		{name: "alternative", source: `if (false) { 10; } else { 20; }`, want: "20"},
		{name: "missing alternative", source: `if (false) { 10; }`, want: "null"},
		{name: "empty block", source: `if (true) {}`, want: "null"},
		{name: "assignment in branch", source: `let value = 1; if (true) { value = 8; } value;`, want: "8"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := compileAndRun(t, test.source)
			if got.Inspect() != test.want {
				t.Fatalf("result = %q, want %q", got.Inspect(), test.want)
			}
		})
	}
}

func TestCompilerWhileLoop(t *testing.T) {
	got := compileAndRun(t, `
let i = 0;
while (i < 3) {
	i = i + 1;
}
i;`)
	if got.Inspect() != "3" {
		t.Fatalf("result = %q, want 3", got.Inspect())
	}
}

func TestCompilerLogicalOperatorsShortCircuit(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{source: `let value = 0; false && (value = 1); value;`, want: "0"},
		{source: `let value = 0; true || (value = 1); value;`, want: "0"},
		{source: `false && missing;`, want: "false"},
		{source: `true || missing;`, want: "true"},
		{source: `1 && "yes";`, want: "true"},
		{source: `null || false;`, want: "false"},
	}

	for _, test := range tests {
		got := compileAndRun(t, test.source)
		if got.Inspect() != test.want {
			t.Errorf("%q = %q, want %q", test.source, got.Inspect(), test.want)
		}
	}
}

func TestCompilerRejectsAssignmentToUndefinedGlobalAtRuntime(t *testing.T) {
	p := parser.New(lexer.New(`true && (missing = 1);`))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	compiler := New()
	if err := compiler.Compile(program); err != nil {
		t.Fatalf("compile error: %v", err)
	}
	machine := vm.New(compiler.Bytecode())
	err := machine.Run()
	if err == nil || !strings.Contains(err.Error(), "uninitialized") {
		t.Fatalf("VM error = %v, want uninitialized global error", err)
	}
}

func TestCompilerComparisonAndNullSemantics(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{source: "5 >= 5;", want: "true"},
		{source: "4 <= 3;", want: "false"},
		{source: "null == null;", want: "true"},
		{source: `"a" + "b";`, want: "ab"},
		{source: "let x = 1;", want: "null"},
		{source: "", want: "null"},
	}
	for _, test := range tests {
		got := compileAndRun(t, test.source)
		if got.Inspect() != test.want {
			t.Errorf("%q = %q, want %q", test.source, got.Inspect(), test.want)
		}
	}
}

func TestCompilerFunctions(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "parameters and implicit return",
			source: `fn add(a, b) { a + b; } add(2, 3);`,
			want:   "5",
		},
		{
			name:   "explicit return from branch",
			source: `fn choose(value) { if (value) { return 7; } 9; } choose(true);`,
			want:   "7",
		},
		{
			name:   "named recursion",
			source: `fn factorial(n) { if (n <= 1) { 1; } else { n * factorial(n - 1); } } factorial(5);`,
			want:   "120",
		},
		{
			name:   "recursive function literal",
			source: `let factorial = fn(n) { if (n <= 1) { 1; } else { n * factorial(n - 1); } }; factorial(5);`,
			want:   "120",
		},
		{
			name:   "nested recursion with captured local",
			source: `let makeCounter = fn(start) { fn count(n) { if (n <= 0) { start; } else { count(n - 1); } } count; }; makeCounter(7)(3);`,
			want:   "7",
		},
		{
			name:   "empty function returns null",
			source: `fn empty() {} empty();`,
			want:   "null",
		},
		{
			name:   "builtin calls",
			source: `len("héllo");`,
			want:   "5",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := compileAndRun(t, test.source)
			if got.Inspect() != test.want {
				t.Fatalf("result = %q, want %q", got.Inspect(), test.want)
			}
		})
	}
}

func TestCompilerClosuresCaptureMutableLocals(t *testing.T) {
	got := compileAndRun(t, `
let makeCounter = fn(start) {
	let count = start;
	(fn(increment) {
		count = count + increment;
		count;
	});
};
let counter = makeCounter(10);
counter(1);
counter(2);`)
	if got.Inspect() != "13" {
		t.Fatalf("result = %q, want 13", got.Inspect())
	}
}

func TestCompilerClosuresKeepInvocationStateSeparate(t *testing.T) {
	got := compileAndRun(t, `
let makeCounter = fn(start) {
	let count = start;
	(fn() {
		count = count + 1;
		count;
	});
};
let first = makeCounter(0);
let second = makeCounter(10);
first();
second();
first();`)
	if got.Inspect() != "2" {
		t.Fatalf("result = %q, want 2", got.Inspect())
	}
}

func TestCompilerClosureResolvesLocalsDeclaredLater(t *testing.T) {
	got := compileAndRun(t, `
let outer = fn() {
	let getLater = fn() { later; };
	let later = 42;
	getLater();
};
outer();`)
	if got.Inspect() != "42" {
		t.Fatalf("result = %q, want 42", got.Inspect())
	}
}

func TestCompilerReportsFunctionCallErrors(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{source: `fn identity(value) { value; } identity();`, want: "wrong number of arguments"},
		{source: `fn identity(value) { value; } identity(1, 2);`, want: "wrong number of arguments"},
		{source: `1(2);`, want: "not a function"},
	}
	for _, test := range tests {
		p := parser.New(lexer.New(test.source))
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}
		compiler := New()
		if err := compiler.Compile(program); err != nil {
			t.Fatalf("compile error: %v", err)
		}
		machine := vm.New(compiler.Bytecode())
		err := machine.Run()
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("%q: VM error = %v, want text %q", test.source, err, test.want)
		}
	}
}

func TestVMStopsAtCallFrameLimit(t *testing.T) {
	p := parser.New(lexer.New(`fn recurse(n) { recurse(n + 1); } recurse(0);`))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	compiler := New()
	if err := compiler.Compile(program); err != nil {
		t.Fatalf("compile error: %v", err)
	}
	err := vm.New(compiler.Bytecode()).Run()
	if err == nil || !strings.Contains(err.Error(), "maximum call frame depth exceeded") {
		t.Fatalf("VM error = %v, want call-frame depth limit", err)
	}
}

func TestCompilerRejectsReturnOutsideFunction(t *testing.T) {
	p := parser.New(lexer.New(`return 1;`))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	err := New().Compile(program)
	if err == nil || !strings.Contains(err.Error(), "only valid inside functions") {
		t.Fatalf("compile error = %v, want return-outside-function error", err)
	}
}

func TestCompilerResultsMatchEvaluatorForCorePrograms(t *testing.T) {
	programs := []string{
		`let value = 3 * (4 + 2); value;`,
		`let i = 0; while (i < 4) { i = i + 1; } i;`,
		`if (2 < 3) { "yes"; } else { "no"; }`,
		`let n = 0; false && (n = 1); n;`,
		`fn factorial(n) { if (n <= 1) { 1; } else { n * factorial(n - 1); } } factorial(6);`,
		`let make = fn(x) { (fn() { x = x + 1; x; }); }; let f = make(4); f(); f();`,
		`let makeCounter = fn(start) { fn count(n) { if (n <= 0) { start; } else { count(n - 1); } } count; }; makeCounter(7)(3);`,
		`let outer = fn() { let getLater = fn() { later; }; let later = 42; getLater(); }; outer();`,
	}

	for _, source := range programs {
		p := parser.New(lexer.New(source))
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}
		expected := evaluator.Eval(program, object.NewEnvironment())

		compiler := New()
		if err := compiler.Compile(program); err != nil {
			t.Fatalf("compile %q: %v", source, err)
		}
		machine := vm.New(compiler.Bytecode())
		if err := machine.Run(); err != nil {
			t.Fatalf("run %q: %v", source, err)
		}
		if got := machine.LastPoppedStackElem().Inspect(); got != expected.Inspect() {
			t.Errorf("%q: VM = %q, evaluator = %q", source, got, expected.Inspect())
		}
	}
}

func TestCompilerAndEvaluatorRuntimeErrorParity(t *testing.T) {
	programs := []string{
		`10 / 0;`,
		`fn identity(value) { value; } identity();`,
		`len(1);`,
		`1();`,
	}

	for _, source := range programs {
		p := parser.New(lexer.New(source))
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}
		expected, ok := evaluator.Eval(program, object.NewEnvironment()).(*object.Error)
		if !ok {
			t.Fatalf("%q: evaluator did not return an error", source)
		}

		compiler := New()
		if err := compiler.Compile(program); err != nil {
			t.Fatalf("compile %q: %v", source, err)
		}
		err := vm.New(compiler.Bytecode()).Run()
		if err == nil || !strings.Contains(err.Error(), expected.Message) {
			t.Errorf("%q: VM error = %v, evaluator error = %q", source, err, expected.Message)
		}
	}
}

func BenchmarkEngineArithmetic(b *testing.B) {
	benchmarkBothEngines(b, `
	let total = 0;
	let i = 0;
	while (i < 1000) {
		total = total + i * 2;
		i = i + 1;
	}
	total;`)
}

func BenchmarkEngineRecursive(b *testing.B) {
	benchmarkBothEngines(b, `
	fn fibonacci(n) {
		if (n < 2) { return n; }
		return fibonacci(n - 1) + fibonacci(n - 2);
	}
	fibonacci(12);`)
}

func BenchmarkEngineFunctionCalls(b *testing.B) {
	benchmarkBothEngines(b, `
	fn identity(value) { return value; }
	let i = 0;
	while (i < 1000) {
		identity(i);
		i = i + 1;
	}
	i;`)
}

func benchmarkBothEngines(b *testing.B, source string) {
	b.Helper()
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		b.Fatalf("parser errors: %v", p.Errors())
	}
	compiler := New()
	if err := compiler.Compile(program); err != nil {
		b.Fatalf("compile error: %v", err)
	}
	bytecode := compiler.Bytecode()

	b.Run("Interpreter", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := evaluator.Eval(program, object.NewEnvironment())
			if runtimeError, ok := result.(*object.Error); ok {
				b.Fatalf("evaluation failed: %s", runtimeError.Inspect())
			}
		}
	})
	b.Run("VM", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			machine := vm.New(bytecode)
			if err := machine.Run(); err != nil {
				b.Fatalf("VM execution failed: %v", err)
			}
		}
	})
}
