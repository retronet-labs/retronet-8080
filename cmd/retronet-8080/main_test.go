package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/retronet-labs/retronet-8080/cpu"
	"github.com/retronet-labs/retronet-8080/machine"
)

func TestRunLoadsProgramAndPrintsDump(t *testing.T) {
	bin := writeTempProgram(t, []byte{cpu.MVI(cpu.RegA), 0x2A, cpu.HLT()})
	var stdout, stderr bytes.Buffer

	code := run([]string{"-bin", bin, "-steps", "8"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run exit=%d stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, part := range []string{
		"profile=generic loaded=3",
		"A=0x2A",
		"PC=0x0003",
		"Halted=true",
		"AC=false",
	} {
		if !strings.Contains(out, part) {
			t.Fatalf("output missing %q:\n%s", part, out)
		}
	}
}

func TestRunDisassembles(t *testing.T) {
	bin := writeTempProgram(t, []byte{cpu.LXI(cpu.PairHL), 0x34, 0x12})
	var stdout, stderr bytes.Buffer

	code := run([]string{"-bin", bin, "-disasm", "1"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run exit=%d stderr=%s", code, stderr.String())
	}
	if got, want := stdout.String(), "0000: 21 34 12 LXI H,#0x1234\n"; got != want {
		t.Fatalf("stdout=%q want=%q", got, want)
	}
}

func TestRunTerminalEcho(t *testing.T) {
	bin := writeTempProgram(t, []byte{cpu.IN(), 0x00, cpu.OUT(), machine.TerminalOutputPort, cpu.HLT()})
	var stdout, stderr bytes.Buffer

	code := run([]string{"-bin", bin, "-terminal-input", "Z", "-steps", "8"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run exit=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Zprofile=") || !strings.Contains(stdout.String(), "A=0x5A") {
		t.Fatalf("stdout=%s", stdout.String())
	}
}

func TestRunListsProfiles(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"-profiles"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run exit=%d stderr=%s", code, stderr.String())
	}
	for _, part := range []string{"generic:", "altair-8800:", "imsai-8080:", "cpm-dev:", "io output 1"} {
		if !strings.Contains(stdout.String(), part) {
			t.Fatalf("profiles missing %q:\n%s", part, stdout.String())
		}
	}
}

func TestRunConformanceSuite(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"-conformance"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run exit=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "failed=0") {
		t.Fatalf("stdout=%s", stdout.String())
	}
}

func writeTempProgram(t *testing.T, program []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "program.bin")
	if err := os.WriteFile(path, program, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
