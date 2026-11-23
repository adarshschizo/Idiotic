package repl

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"idiotic/internal/object"
)

func TestStartKeepsStateAndRunsMultilineFunctions(t *testing.T) {
	input := `let count = 2
fn increment() {
	count = count + 1
	return count
}
increment()
:quit
`
	var out bytes.Buffer
	Start(strings.NewReader(input), &out)

	if got := out.String(); !strings.Contains(got, "3\n") {
		t.Fatalf("REPL output did not contain function result 3:\n%s", got)
	}
}

func TestStartIgnoresBracesInsideStringsAndComments(t *testing.T) {
	input := "let text = \"{\"\n// }\nprint(text)\n:quit\n"
	var out bytes.Buffer
	Start(strings.NewReader(input), &out)

	if got := out.String(); !strings.Contains(got, "{\n") {
		t.Fatalf("REPL output did not contain printed string:\n%s", got)
	}
}

func TestStartCommandsResetEnvironment(t *testing.T) {
	input := "let value = 7\n:help\n:reset\nvalue\n:quit\n"
	var out bytes.Buffer
	Start(strings.NewReader(input), &out)

	for _, want := range []string{"Commands: :quit, :help, :reset", "Environment reset.", "identifier not found: value"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("REPL output did not contain %q:\n%s", want, out.String())
		}
	}
}

func TestStartReportsReaderErrors(t *testing.T) {
	var out bytes.Buffer
	Start(errorReader{}, &out)
	if !strings.Contains(out.String(), "Input error: test input failure") {
		t.Fatalf("REPL did not report input error:\n%s", out.String())
	}
}

func TestRunFileRoutesOutputAndReportsErrors(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name        string
		source      string
		wantOutput  string
		wantErrText string
	}{
		{name: "success", source: `print("script output");`, wantOutput: "script output\n"},
		{name: "parse error", source: "let = 1;", wantErrText: "parse errors:"},
		{name: "runtime error", source: "missing;", wantErrText: "identifier not found: missing"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(dir, test.name+".idio")
			if err := os.WriteFile(path, []byte(test.source), 0600); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			_, err := RunFile(path, object.NewEnvironment(), &out)
			if test.wantErrText == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if test.wantErrText != "" && (err == nil || !strings.Contains(err.Error(), test.wantErrText)) {
				t.Fatalf("got error %v, want text %q", err, test.wantErrText)
			}
			if got := out.String(); got != test.wantOutput {
				t.Fatalf("got output %q, want %q", got, test.wantOutput)
			}
		})
	}
}

func TestRunFileReportsReadErrors(t *testing.T) {
	_, err := RunFile(filepath.Join(t.TempDir(), "missing.idio"), object.NewEnvironment(), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected a file read error")
	}
}

func TestRunFileWithVMRoutesOutputAndRunsClosures(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vm.idio")
	source := `
fn makeCounter(start) {
	let count = start;
	fn increment() {
		count = count + 1;
		return count;
	}
	return increment;
}
let counter = makeCounter(8);
print(counter());
print(counter());`
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if _, err := RunFileWithVM(path, &out); err != nil {
		t.Fatalf("unexpected VM error: %v", err)
	}
	if got, want := out.String(), "9\n10\n"; got != want {
		t.Fatalf("VM stdout = %q, want %q", got, want)
	}
}

func TestRunFileWithVMReportsCompileAndRuntimeErrors(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		wantErrText string
	}{
		{name: "parse error", source: "let = 1;", wantErrText: "parse errors:"},
		{name: "compile error", source: "return 1;", wantErrText: "compile error:"},
		{name: "runtime error", source: "missing;", wantErrText: "uninitialized"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "program.idio")
			if err := os.WriteFile(path, []byte(test.source), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := RunFileWithVM(path, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), test.wantErrText) {
				t.Fatalf("got error %v, want text %q", err, test.wantErrText)
			}
		})
	}
}

func TestRunFileWithVMReportsOutputErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output.idio")
	if err := os.WriteFile(path, []byte(`print("value");`), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := RunFileWithVM(path, failingWriter{})
	if err == nil || !strings.Contains(err.Error(), "print failed: test output failure") {
		t.Fatalf("got error %v, want output writer failure", err)
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("test input failure")
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("test output failure")
}
