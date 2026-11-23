package vm

import (
	"fmt"
	"io"
	"unicode/utf8"

	"idiotic/internal/object"
)

func builtinFor(name string, output io.Writer) *object.Builtin {
	switch name {
	case "print":
		return &object.Builtin{Fn: func(args ...object.Object) object.Object {
			for i, arg := range args {
				if i > 0 {
					if _, err := fmt.Fprint(output, " "); err != nil {
						return &object.Error{Message: fmt.Sprintf("print failed: %s", err)}
					}
				}
				if _, err := fmt.Fprint(output, arg.Inspect()); err != nil {
					return &object.Error{Message: fmt.Sprintf("print failed: %s", err)}
				}
			}
			if _, err := fmt.Fprintln(output); err != nil {
				return &object.Error{Message: fmt.Sprintf("print failed: %s", err)}
			}
			return &object.Null{}
		}}
	case "len":
		return &object.Builtin{Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments: got=%d, want=1", len(args))}
			}
			value, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to len must be STRING, got %s", args[0].Type())}
			}
			return &object.Integer{Value: int64(utf8.RuneCountInString(value.Value))}
		}}
	}
	return nil
}
