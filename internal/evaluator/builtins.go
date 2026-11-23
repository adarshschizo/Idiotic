package evaluator

import (
	"fmt"
	"io"

	"idiotic/internal/object"
)

func InstallBuiltins(env *object.Environment, out io.Writer) {
	env.Set("print", &object.Builtin{Fn: func(args ...object.Object) object.Object {
		for i, arg := range args {
			if i > 0 {
				if _, err := fmt.Fprint(out, " "); err != nil {
					return newError("print failed: %s", err)
				}
			}
			if _, err := fmt.Fprint(out, arg.Inspect()); err != nil {
				return newError("print failed: %s", err)
			}
		}
		if _, err := fmt.Fprintln(out); err != nil {
			return newError("print failed: %s", err)
		}
		return NULL
	}})
	env.Set("len", Builtins["len"])
}

var Builtins = map[string]*object.Builtin{
	"print": {Fn: func(args ...object.Object) object.Object {
		for i, arg := range args {
			if i > 0 {
				fmt.Print(" ")
			}
			fmt.Print(arg.Inspect())
		}
		fmt.Println()
		return NULL
	}},
	"len": {Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return newError("wrong number of arguments: got=%d, want=1", len(args))
		}
		switch arg := args[0].(type) {
		case *object.String:
			return &object.Integer{Value: int64(len([]rune(arg.Value)))}
		default:
			return newError("argument to len must be STRING, got %s", arg.Type())
		}
	}},
}
