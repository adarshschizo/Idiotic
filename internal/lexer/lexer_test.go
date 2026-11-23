package lexer

import (
	"testing"

	"idiotic/internal/token"
)

func TestLexerTokenizesKeywordsAndOperators(t *testing.T) {
	input := `fn let true false null if else return while for _x 123 "text" = + - ! * / < > == != <= >= && || , ; : ( ) { }`
	expected := []token.Type{
		token.FUNCTION, token.LET, token.TRUE, token.FALSE, token.NULL, token.IF,
		token.ELSE, token.RETURN, token.WHILE, token.FOR, token.IDENT, token.INT,
		token.STRING, token.ASSIGN, token.PLUS, token.MINUS, token.BANG,
		token.ASTERISK, token.SLASH, token.LT, token.GT, token.EQ, token.NOT_EQ,
		token.LTE, token.GTE, token.AND, token.OR, token.COMMA, token.SEMICOLON,
		token.COLON, token.LPAREN, token.RPAREN, token.LBRACE, token.RBRACE,
		token.EOF,
	}

	l := New(input)
	for i, want := range expected {
		if got := l.NextToken(); got.Type != want {
			t.Fatalf("token %d: got %s (%q), want %s", i, got.Type, got.Literal, want)
		}
	}
}

func TestLexerDecodesEscapedString(t *testing.T) {
	l := New(`"line\nquote: \" slash: \\"`)
	got := l.NextToken()
	if got.Type != token.STRING {
		t.Fatalf("got token type %s, want STRING", got.Type)
	}
	if want := "line\nquote: \" slash: \\"; got.Literal != want {
		t.Fatalf("got string %q, want %q", got.Literal, want)
	}
}

func TestLexerDecodesEachSupportedEscape(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: `"a\nb"`, want: "a\nb"},
		{input: `"a\rb"`, want: "a\rb"},
		{input: `"a\tb"`, want: "a\tb"},
		{input: `"a\"b"`, want: `a"b`},
		{input: `"a\\b"`, want: `a\b`},
	}

	for _, test := range tests {
		got := New(test.input).NextToken()
		if got.Type != token.STRING || got.Literal != test.want {
			t.Errorf("%q produced %s (%q), want STRING (%q)", test.input, got.Type, got.Literal, test.want)
		}
	}
}

func TestLexerSkipsCommentsAndTracksLocations(t *testing.T) {
	l := New("let x = 1; // comment\n@")
	for range 5 {
		l.NextToken()
	}
	got := l.NextToken()
	if got.Type != token.ILLEGAL || got.Literal != "@" {
		t.Fatalf("got token %s (%q), want ILLEGAL (@)", got.Type, got.Literal)
	}
	if got.Line != 2 || got.Column != 1 {
		t.Fatalf("got location %d:%d, want 2:1", got.Line, got.Column)
	}
}

func TestLexerTracksLocationsAcrossCRLFAndUnicode(t *testing.T) {
	l := New("\"é\";\r\nx")
	want := []struct {
		literal string
		line    int
		column  int
	}{
		{literal: "é", line: 1, column: 1},
		{literal: ";", line: 1, column: 4},
		{literal: "x", line: 2, column: 1},
		{literal: "", line: 2, column: 2},
	}

	for i, expected := range want {
		got := l.NextToken()
		if got.Literal != expected.literal || got.Line != expected.line || got.Column != expected.column {
			t.Errorf("token %d = %q at %d:%d, want %q at %d:%d",
				i, got.Literal, got.Line, got.Column,
				expected.literal, expected.line, expected.column)
		}
	}
}

func TestLexerEmitsSingleCharacterLogicalOperatorsAsIllegal(t *testing.T) {
	for _, input := range []string{"@", "&", "|"} {
		l := New(input + "x")
		illegal := l.NextToken()
		if illegal.Type != token.ILLEGAL || illegal.Literal != input {
			t.Errorf("%q produced %s (%q), want ILLEGAL (%q)", input, illegal.Type, illegal.Literal, input)
		}
		if next := l.NextToken(); next.Type != token.IDENT || next.Literal != "x" {
			t.Errorf("token after %q = %s (%q), want IDENT (x)", input, next.Type, next.Literal)
		}
	}
}

func TestLexerRepeatedEOFHasStableLocation(t *testing.T) {
	for _, input := range []string{"", "x"} {
		l := New(input)
		if input != "" {
			l.NextToken()
		}
		first := l.NextToken()
		second := l.NextToken()
		if first.Type != token.EOF || second.Type != token.EOF {
			t.Fatalf("%q: got %s then %s, want EOF twice", input, first.Type, second.Type)
		}
		if first.Line != second.Line || first.Column != second.Column {
			t.Errorf("%q: EOF moved from %d:%d to %d:%d",
				input, first.Line, first.Column, second.Line, second.Column)
		}
	}
}

func TestLexerReportsUnterminatedString(t *testing.T) {
	got := New(`"unfinished`).NextToken()
	if got.Type != token.ILLEGAL || got.Literal != "unterminated string" {
		t.Fatalf("got token %s (%q), want ILLEGAL (unterminated string)", got.Type, got.Literal)
	}
	if got.Line != 1 || got.Column != 1 {
		t.Fatalf("got location %d:%d, want 1:1", got.Line, got.Column)
	}
}
