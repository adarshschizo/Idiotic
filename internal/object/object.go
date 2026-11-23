package object

import (
	"fmt"
	"strings"

	"idiotic/internal/ast"
)

type Object interface {
	Type() string
	Inspect() string
}
type Integer struct{ Value int64 }

func (o *Integer) Type() string    { return "INTEGER" }
func (o *Integer) Inspect() string { return fmt.Sprintf("%d", o.Value) }

type String struct{ Value string }

func (o *String) Type() string    { return "STRING" }
func (o *String) Inspect() string { return o.Value }

type Boolean struct{ Value bool }

func (o *Boolean) Type() string { return "BOOLEAN" }
func (o *Boolean) Inspect() string {
	if o.Value {
		return "true"
	}
	return "false"
}

type Null struct{}

func (o *Null) Type() string    { return "NULL" }
func (o *Null) Inspect() string { return "null" }

type Error struct{ Message string }

func (o *Error) Type() string    { return "ERROR" }
func (o *Error) Inspect() string { return "ERROR: " + o.Message }

type ReturnValue struct{ Value Object }

func (o *ReturnValue) Type() string    { return "RETURN_VALUE" }
func (o *ReturnValue) Inspect() string { return o.Value.Inspect() }

type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
}

func (o *Function) Type() string { return "FUNCTION" }
func (o *Function) Inspect() string {
	params := []string{}
	for _, p := range o.Parameters {
		params = append(params, p.String())
	}
	return "fn(" + strings.Join(params, ", ") + ") " + o.Body.String()
}

type Builtin struct{ Fn func(args ...Object) Object }

func (o *Builtin) Type() string    { return "BUILTIN" }
func (o *Builtin) Inspect() string { return "builtin function" }

type CompiledFunction struct {
	Instructions  []byte
	NumLocals     int
	NumParameters int
}

func (o *CompiledFunction) Type() string    { return "COMPILED_FUNCTION" }
func (o *CompiledFunction) Inspect() string { return "compiled function" }

type Cell struct{ Value Object }

type CellReference struct{ Cell *Cell }

func (o *CellReference) Type() string    { return "CELL_REFERENCE" }
func (o *CellReference) Inspect() string { return "cell reference" }

type Closure struct {
	Fn   *CompiledFunction
	Free []*Cell
}

func (o *Closure) Type() string    { return "CLOSURE" }
func (o *Closure) Inspect() string { return "closure" }

type Environment struct {
	store map[string]Object
	outer *Environment
}

func NewEnvironment() *Environment { return &Environment{store: map[string]Object{}} }
func NewEnclosedEnvironment(outer *Environment) *Environment {
	return &Environment{store: map[string]Object{}, outer: outer}
}
func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		return e.outer.Get(name)
	}
	return obj, ok
}
func (e *Environment) Set(name string, Object Object) Object { e.store[name] = Object; return Object }
func (e *Environment) Assign(name string, value Object) bool {
	if _, ok := e.store[name]; ok {
		e.store[name] = value
		return true
	}
	if e.outer != nil {
		return e.outer.Assign(name, value)
	}
	return false
}
