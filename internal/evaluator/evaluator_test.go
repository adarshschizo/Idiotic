package evaluator

import (
	"strings"
	"testing"

	"idiotic/internal/ast"
	"idiotic/internal/lexer"
	"idiotic/internal/object"
	"idiotic/internal/parser"
)

func testEval(t *testing.T, input string) object.Object {
	t.Helper()
	p := parser.New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	return Eval(program, object.NewEnvironment())
}

func TestArithmetic(t *testing.T) {
	obj := testEval(t, `let x = 5 + 3 * 2; x;`)
	result, ok := obj.(*object.Integer)
	if !ok || result.Value != 11 {
		t.Fatalf("got %v, want 11", obj)
	}
}

func TestEvaluatorExpressionSemantics(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "5 + 3 * 2;", want: "11"},
		{input: "5 <= 5;", want: "true"},
		{input: `"hello" + " world";`, want: "hello world"},
		{input: "!true;", want: "false"},
		{input: "!!5;", want: "true"},
		{input: "null == null;", want: "true"},
		{input: `if (false) { 1; } else { 2; }`, want: "2"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got := testEval(t, test.input)
			if got.Inspect() != test.want {
				t.Fatalf("got %q, want %q", got.Inspect(), test.want)
			}
		})
	}
}

func TestClosure(t *testing.T) {
	obj := testEval(t, `
fn makeCounter(start) {
	let count = start;
	fn increment() {
		count = count + 1;
		return count;
	}
	return increment;
}
let counter = makeCounter(10);
counter();`)
	result, ok := obj.(*object.Integer)
	if !ok || result.Value != 11 {
		t.Fatalf("got %v, want 11", obj)
	}
}

func TestClosureRetainsMutableLexicalBinding(t *testing.T) {
	obj := testEval(t, `
fn makeCounter(start) {
	let count = start;
	fn increment() {
		count = count + 1;
		return count;
	}
	return increment;
}
let counter = makeCounter(10);
counter();
counter();
counter();`)
	result, ok := obj.(*object.Integer)
	if !ok || result.Value != 13 {
		t.Fatalf("got %v, want 13", obj)
	}
}

func TestFunctionsUseLexicalScope(t *testing.T) {
	obj := testEval(t, `
let value = 10;
fn readValue() { return value; }
fn callWithShadow() {
	let value = 20;
	return readValue();
}
callWithShadow();`)
	result, ok := obj.(*object.Integer)
	if !ok || result.Value != 10 {
		t.Fatalf("got %v, want 10", obj)
	}
}

func TestRecursion(t *testing.T) {
	obj := testEval(t, `
fn factorial(n) {
	if (n <= 1) { return 1; }
	return n * factorial(n - 1);
}
factorial(6);`)
	result, ok := obj.(*object.Integer)
	if !ok || result.Value != 720 {
		t.Fatalf("got %v, want 720", obj)
	}
}

func TestDeepFiniteRecursion(t *testing.T) {
	obj := testEval(t, `
fn countdown(n) {
	if (n <= 0) { return 0; }
	return countdown(n - 1);
}
countdown(200);`)
	result, ok := obj.(*object.Integer)
	if !ok || result.Value != 0 {
		t.Fatalf("got %v, want 0", obj)
	}
}

func TestExcessiveRecursionReturnsRuntimeError(t *testing.T) {
	obj := testEval(t, `
fn recurse() { return recurse(); }
recurse();`)
	err, ok := obj.(*object.Error)
	if !ok {
		t.Fatalf("got %v, want runtime error", obj)
	}
	if want := "maximum function call depth exceeded"; !strings.HasPrefix(err.Message, want) {
		t.Fatalf("error = %q, want prefix %q", err.Message, want)
	}
}

func TestDivisionByZero(t *testing.T) {
	obj := testEval(t, `10 / 0;`)
	err, ok := obj.(*object.Error)
	if !ok || err.Message != "division by zero" {
		t.Fatalf("got %v, want division by zero", obj)
	}
}

func TestEvalRejectsNilInputs(t *testing.T) {
	if got := Eval(nil, object.NewEnvironment()); !isError(got) {
		t.Fatalf("Eval(nil, env) = %v, want runtime error", got)
	}
	program := &ast.Program{}
	if got := Eval(program, nil); !isError(got) {
		t.Fatalf("Eval(program, nil) = %v, want runtime error", got)
	}
}

func TestLogicalOperatorsShortCircuit(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{input: "false && missing;", want: false},
		{input: "true || missing;", want: true},
	}
	for _, test := range tests {
		obj := testEval(t, test.input)
		if err, ok := obj.(*object.Error); ok {
			t.Fatalf("%q returned error: %s", test.input, err.Message)
		}
		result, ok := obj.(*object.Boolean)
		if !ok || result.Value != test.want {
			t.Errorf("%q got %v, want %t", test.input, obj, test.want)
		}
	}
}

func TestWhileLoop(t *testing.T) {
	obj := testEval(t, `
let count = 0;
while (count < 3) {
	count = count + 1;
}
count;`)
	result, ok := obj.(*object.Integer)
	if !ok || result.Value != 3 {
		t.Fatalf("got %v, want 3", obj)
	}
}

func BenchmarkEvalArithmetic(b *testing.B) {
	benchmarkProgram(b, `
let total = 0;
let i = 0;
while (i < 1000) {
	total = total + i * 2;
	i = i + 1;
}
total;`)
}

func BenchmarkEvalRecursive(b *testing.B) {
	benchmarkProgram(b, `
fn fibonacci(n) {
	if (n < 2) { return n; }
	return fibonacci(n - 1) + fibonacci(n - 2);
}
fibonacci(12);`)
}

func BenchmarkEvalFunctionCalls(b *testing.B) {
	benchmarkProgram(b, `
fn identity(value) { return value; }
let i = 0;
while (i < 1000) {
	identity(i);
	i = i + 1;
}
i;`)
}

func benchmarkProgram(b *testing.B, source string) {
	b.Helper()
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		b.Fatalf("parser errors: %v", p.Errors())
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := Eval(program, object.NewEnvironment())
		if isError(result) {
			b.Fatalf("evaluation failed: %s", result.Inspect())
		}
	}
}
