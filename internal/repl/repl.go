package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"idiotic/internal/ast"
	"idiotic/internal/compiler"
	"idiotic/internal/evaluator"
	"idiotic/internal/lexer"
	"idiotic/internal/object"
	"idiotic/internal/parser"
	"idiotic/internal/vm"
)

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	env := object.NewEnvironment()
	buffer := ""
	evaluator.InstallBuiltins(env, out)
	fmt.Fprint(out, "Idiotic REPL (type :help or :quit)\n")
	for {
		if buffer == "" {
			fmt.Fprint(out, "idiotic> ")
		} else {
			fmt.Fprint(out, "......> ")
		}
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Fprintln(out, "Input error:", err)
			}
			if buffer != "" {
				run(buffer, env, out)
			}
			return
		}
		line := scanner.Text()
		if buffer == "" {
			switch strings.TrimSpace(line) {
			case ":quit":
				return
			case ":help":
				fmt.Fprintln(out, "Commands: :quit, :help, :reset")
				continue
			case ":reset":
				env = object.NewEnvironment()
				evaluator.InstallBuiltins(env, out)
				fmt.Fprintln(out, "Environment reset.")
				continue
			}
		}
		buffer += line + "\n"
		if hasUnclosedDelimiters(buffer) {
			continue
		}
		run(buffer, env, out)
		buffer = ""
	}
}

func RunFile(path string, env *object.Environment, out io.Writer) (object.Object, error) {
	program, err := readProgram(path)
	if err != nil {
		return nil, err
	}
	evaluator.InstallBuiltins(env, out)
	result := evaluator.Eval(program, env)
	if result != nil && result.Type() == "ERROR" {
		return result, fmt.Errorf("%s", result.Inspect())
	}
	return result, nil
}

func RunFileWithVM(path string, out io.Writer) (object.Object, error) {
	program, err := readProgram(path)
	if err != nil {
		return nil, err
	}
	bytecodeCompiler := compiler.New()
	if err := bytecodeCompiler.Compile(program); err != nil {
		return nil, fmt.Errorf("compile error: %w", err)
	}
	machine := vm.NewWithOutput(bytecodeCompiler.Bytecode(), out)
	if err := machine.Run(); err != nil {
		return nil, err
	}
	return machine.LastPoppedStackElem(), nil
}

func readProgram(path string) (*ast.Program, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	l := lexer.New(string(data))
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors:\n%s", strings.Join(p.Errors(), "\n"))
	}
	return program, nil
}

func run(input string, env *object.Environment, out io.Writer) {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			fmt.Fprintln(out, "Parse error:", err)
		}
		return
	}
	result := evaluator.Eval(program, env)
	if result != nil && result.Type() == "ERROR" {
		fmt.Fprintln(out, result.Inspect())
		return
	}
	if result != nil && len(program.Statements) > 0 {
		if _, isExpression := program.Statements[len(program.Statements)-1].(*ast.ExpressionStatement); isExpression {
			if _, isNull := result.(*object.Null); !isNull {
				fmt.Fprintln(out, result.Inspect())
			}
		}
	}
}
func hasUnclosedDelimiters(source string) bool {
	var braces, parentheses int
	inString, escaped, inComment := false, false, false
	runes := []rune(source)
	for i, r := range runes {
		if inComment {
			if r == '\n' {
				inComment = false
			}
			continue
		}
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch r {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}
		if r == '/' && i+1 < len(runes) && runes[i+1] == '/' {
			inComment = true
			continue
		}
		switch r {
		case '"':
			inString = true
		case '{':
			braces++
		case '}':
			braces--
		case '(':
			parentheses++
		case ')':
			parentheses--
		}
	}
	return inString || braces > 0 || parentheses > 0
}
