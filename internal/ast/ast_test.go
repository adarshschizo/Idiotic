package ast

import (
	"math"
	"testing"
)

func TestNodeStringAndTokenLiteral(t *testing.T) {
	left := &Identifier{Value: "left"}
	right := &Identifier{Value: "right"}
	body := &BlockStatement{Statements: []Statement{
		&ReturnStatement{ReturnValue: &Identifier{Value: "x"}},
	}}
	fn := &FunctionLiteral{
		Parameters: []*Identifier{{Value: "x"}},
		Body:       body,
	}

	tests := []struct {
		name        string
		node        Node
		wantString  string
		wantLiteral string
	}{
		{
			name:        "program",
			node:        &Program{Statements: []Statement{&LetStatement{Name: &Identifier{Value: "x"}, Value: &IntegerLiteral{Value: 5}}}},
			wantString:  "let x = 5;",
			wantLiteral: "let",
		},
		{
			name:        "let statement",
			node:        &LetStatement{Name: &Identifier{Value: "x"}, Value: &IntegerLiteral{Value: 5}},
			wantString:  "let x = 5;",
			wantLiteral: "let",
		},
		{
			name:        "return statement",
			node:        &ReturnStatement{ReturnValue: &Identifier{Value: "x"}},
			wantString:  "return x;",
			wantLiteral: "return",
		},
		{
			name:        "expression statement",
			node:        &ExpressionStatement{Expression: &IntegerLiteral{Value: 5}},
			wantString:  "5",
			wantLiteral: "5",
		},
		{
			name:        "block statement",
			node:        body,
			wantString:  "{return x;}",
			wantLiteral: "{",
		},
		{
			name:        "identifier",
			node:        left,
			wantString:  "left",
			wantLiteral: "left",
		},
		{
			name:        "integer literal",
			node:        &IntegerLiteral{Value: math.MinInt64},
			wantString:  "-9223372036854775808",
			wantLiteral: "-9223372036854775808",
		},
		{
			name:        "escaped string literal",
			node:        &StringLiteral{Value: "a\"b\n"},
			wantString:  `"a\"b\n"`,
			wantLiteral: `"a\"b\n"`,
		},
		{
			name:        "boolean",
			node:        &Boolean{Value: false},
			wantString:  "false",
			wantLiteral: "false",
		},
		{
			name:        "null",
			node:        &Null{},
			wantString:  "null",
			wantLiteral: "null",
		},
		{
			name:        "prefix expression",
			node:        &PrefixExpression{Operator: "!", Right: &Boolean{Value: false}},
			wantString:  "(!false)",
			wantLiteral: "!",
		},
		{
			name:        "infix expression",
			node:        &InfixExpression{Left: left, Operator: "+", Right: right},
			wantString:  "(left + right)",
			wantLiteral: "left",
		},
		{
			name:        "assignment expression",
			node:        &AssignExpression{Name: &Identifier{Value: "target"}, Value: &IntegerLiteral{Value: 3}},
			wantString:  "target = 3",
			wantLiteral: "target",
		},
		{
			name: "if expression",
			node: &IfExpression{
				Condition:   &Boolean{Value: true},
				Consequence: &BlockStatement{Statements: []Statement{&ExpressionStatement{Expression: &IntegerLiteral{Value: 1}}}},
				Alternative: &BlockStatement{Statements: []Statement{&ExpressionStatement{Expression: &IntegerLiteral{Value: 2}}}},
			},
			wantString:  "if (true) {1} else {2}",
			wantLiteral: "if",
		},
		{
			name:        "while expression",
			node:        &WhileExpression{Condition: &Boolean{Value: true}, Body: &BlockStatement{}},
			wantString:  "while (true) {}",
			wantLiteral: "while",
		},
		{
			name:        "function literal",
			node:        fn,
			wantString:  "fn(x) {return x;}",
			wantLiteral: "fn",
		},
		{
			name:        "function statement",
			node:        &FunctionStatement{Name: &Identifier{Value: "identity"}, Function: fn},
			wantString:  "fn identity(x) {return x;}",
			wantLiteral: "fn",
		},
		{
			name:        "call expression",
			node:        &CallExpression{Function: &Identifier{Value: "add"}, Arguments: []Expression{&IntegerLiteral{Value: 1}, &IntegerLiteral{Value: 2}}},
			wantString:  "add(1, 2)",
			wantLiteral: "add",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.node.String(); got != test.wantString {
				t.Errorf("String() = %q, want %q", got, test.wantString)
			}
			if got := test.node.TokenLiteral(); got != test.wantLiteral {
				t.Errorf("TokenLiteral() = %q, want %q", got, test.wantLiteral)
			}
		})
	}
}
