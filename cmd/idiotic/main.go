package main

import (
	"fmt"
	"io"
	"os"

	"idiotic/internal/object"
	"idiotic/internal/repl"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, in io.Reader, out, errOut io.Writer) int {
	if len(args) == 0 {
		repl.Start(in, out)
		return 0
	}

	switch args[0] {
	case "run":
		var runErr error
		switch {
		case len(args) == 2:
			_, runErr = repl.RunFile(args[1], object.NewEnvironment(), out)
		case len(args) == 3 && args[1] == "--vm":
			_, runErr = repl.RunFileWithVM(args[2], out)
		default:
			fmt.Fprintln(errOut, "Usage: idiotic run [--vm] <file.idio>")
			return 2
		}
		if runErr != nil {
			fmt.Fprintln(errOut, runErr)
			return 1
		}
		return 0
	case "--help", "-h":
		fmt.Fprintln(out, "Usage:\n  idiotic                         Start the REPL\n  idiotic run [--vm] <file.idio>  Execute a source file")
		return 0
	default:
		fmt.Fprintf(errOut, "Unknown command: %s\n", args[0])
		return 2
	}
}
