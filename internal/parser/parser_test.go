package parser

import (
	"idiotic/internal/lexer"
	"testing"
)

func TestPrecedenceAndClosureProgram(t *testing.T) {
	input := `let x = -(5 + 3) * 2;
fn makeCounter(start) {
	let count = start;
	fn increment() {
		count = count + 1;
		return count;
	}
	return increment;
}`
	p := New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	if len(program.Statements) != 2 {
		t.Fatalf("got %d statements, want 2", len(program.Statements))
	}
}

func TestPrattPrecedenceString(t *testing.T) {
	p := New(lexer.New("5 + 3 * 2"))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	if got, want := program.String(), "(5 + (3 * 2))"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestMalformedBlocksAndParametersReportErrors(t *testing.T) {
	inputs := []string{
		"if (true) {",
		"fn incomplete(value) {",
		"fn invalid(1) {}",
		"if (true) {} else",
	}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			p := New(lexer.New(input))
			p.ParseProgram()
			if len(p.Errors()) == 0 {
				t.Fatal("expected a parser error")
			}
		})
	}
}

func FuzzParserDoesNotPanic(f *testing.F) {
	for _, source := range []string{
		"",
		"5 + 3 * 2",
		"fn add(a, b) { return a + b; }",
		"if (true) {",
		"let = ;",
		"fn invalid(1) {}",
	} {
		f.Add(source)
	}

	f.Fuzz(func(t *testing.T, source string) {
		p := New(lexer.New(source))
		p.ParseProgram()
	})
}
