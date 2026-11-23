package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHelpAndArgumentErrors(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantCode  int
		wantOut   string
		wantError string
	}{
		{name: "help", args: []string{"--help"}, wantOut: "--vm"},
		{name: "unknown command", args: []string{"unknown"}, wantCode: 2, wantError: "Unknown command"},
		{name: "missing file", args: []string{"run"}, wantCode: 2, wantError: "Usage: idiotic run"},
		{name: "extra arguments", args: []string{"run", "file.idio", "extra"}, wantCode: 2, wantError: "Usage: idiotic run"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			gotCode := run(test.args, strings.NewReader(":quit\n"), &out, &errOut)
			if gotCode != test.wantCode {
				t.Fatalf("got exit code %d, want %d", gotCode, test.wantCode)
			}
			if !strings.Contains(out.String(), test.wantOut) {
				t.Errorf("stdout %q does not contain %q", out.String(), test.wantOut)
			}
			if !strings.Contains(errOut.String(), test.wantError) {
				t.Errorf("stderr %q does not contain %q", errOut.String(), test.wantError)
			}
		})
	}
}

func TestRunScriptSuccessAndFailureExitCodes(t *testing.T) {
	dir := t.TempDir()
	successPath := filepath.Join(dir, "success.idio")
	if err := os.WriteFile(successPath, []byte(`print("ok");`), 0600); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if code := run([]string{"run", successPath}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("got exit code %d, want 0 (stderr %q)", code, errOut.String())
	}
	if got, want := out.String(), "ok\n"; got != want {
		t.Fatalf("got stdout %q, want %q", got, want)
	}

	invalidPath := filepath.Join(dir, "invalid.idio")
	if err := os.WriteFile(invalidPath, []byte("missing;"), 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()
	if code := run([]string{"run", invalidPath}, strings.NewReader(""), &out, &errOut); code != 1 {
		t.Fatalf("got exit code %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "identifier not found: missing") {
		t.Fatalf("stderr did not contain runtime error: %q", errOut.String())
	}
}

func TestRunScriptWithVMOption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vm.idio")
	if err := os.WriteFile(path, []byte(`print(2 * 3);`), 0600); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if code := run([]string{"run", "--vm", path}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("got exit code %d, want 0 (stderr %q)", code, errOut.String())
	}
	if got, want := out.String(), "6\n"; got != want {
		t.Fatalf("VM stdout = %q, want %q", got, want)
	}

	out.Reset()
	errOut.Reset()
	if code := run([]string{"run", "--vm", path, "extra"}, strings.NewReader(""), &out, &errOut); code != 2 {
		t.Fatalf("got exit code %d for invalid VM arguments, want 2", code)
	}
	if !strings.Contains(errOut.String(), "idiotic run [--vm]") {
		t.Fatalf("usage does not mention VM form: %q", errOut.String())
	}

	invalidPath := filepath.Join(t.TempDir(), "invalid.idio")
	if err := os.WriteFile(invalidPath, []byte("missing;"), 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()
	if code := run([]string{"run", "--vm", invalidPath}, strings.NewReader(""), &out, &errOut); code != 1 {
		t.Fatalf("got exit code %d for VM runtime error, want 1", code)
	}
	if out.Len() != 0 || !strings.Contains(errOut.String(), "uninitialized") {
		t.Fatalf("VM error routing: stdout %q, stderr %q", out.String(), errOut.String())
	}
}
