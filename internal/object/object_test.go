package object

import (
	"testing"

	"idiotic/internal/ast"
)

func TestObjectTypeAndInspect(t *testing.T) {
	function := &Function{
		Parameters: []*ast.Identifier{{Value: "x"}},
		Body: &ast.BlockStatement{Statements: []ast.Statement{
			&ast.ReturnStatement{ReturnValue: &ast.Identifier{Value: "x"}},
		}},
		Env: NewEnvironment(),
	}
	builtin := &Builtin{Fn: func(...Object) Object { return &Null{} }}

	tests := []struct {
		name        string
		object      Object
		wantType    string
		wantInspect string
	}{
		{name: "integer", object: &Integer{Value: -42}, wantType: "INTEGER", wantInspect: "-42"},
		{name: "string", object: &String{Value: "hello"}, wantType: "STRING", wantInspect: "hello"},
		{name: "true", object: &Boolean{Value: true}, wantType: "BOOLEAN", wantInspect: "true"},
		{name: "false", object: &Boolean{Value: false}, wantType: "BOOLEAN", wantInspect: "false"},
		{name: "null", object: &Null{}, wantType: "NULL", wantInspect: "null"},
		{name: "error", object: &Error{Message: "bad input"}, wantType: "ERROR", wantInspect: "ERROR: bad input"},
		{name: "return", object: &ReturnValue{Value: &Integer{Value: 3}}, wantType: "RETURN_VALUE", wantInspect: "3"},
		{name: "function", object: function, wantType: "FUNCTION", wantInspect: "fn(x) {return x;}"},
		{name: "builtin", object: builtin, wantType: "BUILTIN", wantInspect: "builtin function"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.object.Type(); got != test.wantType {
				t.Errorf("Type() = %q, want %q", got, test.wantType)
			}
			if got := test.object.Inspect(); got != test.wantInspect {
				t.Errorf("Inspect() = %q, want %q", got, test.wantInspect)
			}
		})
	}
}

func TestEnvironmentGetSetAndAssignAcrossScopes(t *testing.T) {
	outer := NewEnvironment()
	outer.Set("value", &Integer{Value: 1})
	outer.Set("outerOnly", &Integer{Value: 0})
	inner := NewEnclosedEnvironment(outer)

	if got, ok := inner.Get("value"); !ok || got.Inspect() != "1" {
		t.Fatalf("Get inherited value = (%v, %t), want (1, true)", got, ok)
	}

	inner.Set("value", &Integer{Value: 2})
	if !inner.Assign("value", &Integer{Value: 3}) {
		t.Fatal("Assign should update the nearest local binding")
	}
	if got, _ := inner.Get("value"); got.Inspect() != "3" {
		t.Errorf("inner value = %s, want 3", got.Inspect())
	}
	if got, _ := outer.Get("value"); got.Inspect() != "1" {
		t.Errorf("outer value = %s, want shadowed value 1", got.Inspect())
	}

	if !inner.Assign("outerOnly", &Integer{Value: 4}) {
		t.Fatal("Assign should update an inherited binding")
	}
	if got, _ := outer.Get("outerOnly"); got.Inspect() != "4" {
		t.Errorf("outerOnly = %s, want 4", got.Inspect())
	}

	if _, ok := inner.Get("missing"); ok {
		t.Error("Get reported a missing binding as present")
	}
	if inner.Assign("missing", &Integer{Value: 5}) {
		t.Error("Assign created a missing binding")
	}
}
