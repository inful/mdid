package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetFlagSet(t *testing.T) {
	t.Helper()

	old := flag.CommandLine
	t.Cleanup(func() {
		flag.CommandLine = old
	})

	flag.CommandLine = flag.NewFlagSet("mdid-test", flag.ContinueOnError)
}

func setStdinToString(t *testing.T, input string) {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "stdin-*.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.WriteString(input); err != nil {
		t.Fatal(err)
	}
	if _, err = f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	old := os.Stdin
	os.Stdin = f
	t.Cleanup(func() {
		_ = f.Close()
		os.Stdin = old
	})
}

func captureOutputFiles(t *testing.T) (stdoutPath, stderrPath string) {
	t.Helper()

	stdoutFile, err := os.CreateTemp(t.TempDir(), "stdout-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	stderrFile, err := os.CreateTemp(t.TempDir(), "stderr-*.txt")
	if err != nil {
		t.Fatal(err)
	}

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout = stdoutFile
	os.Stderr = stderrFile

	t.Cleanup(func() {
		_ = stdoutFile.Close()
		_ = stderrFile.Close()
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	})

	return stdoutFile.Name(), stderrFile.Name()
}

func setModes(t *testing.T, recursive, verbose bool) {
	t.Helper()

	oldRecursive := recursiveMode
	oldVerbose := verboseMode
	recursiveMode = recursive
	verboseMode = verbose

	t.Cleanup(func() {
		recursiveMode = oldRecursive
		verboseMode = oldVerbose
	})
}

func TestInitFlags(t *testing.T) {
	resetFlagSet(t)

	initFlags()

	if flag.Lookup("r") == nil {
		t.Fatal("initFlags() did not register -r")
	}
	if flag.Lookup("v") == nil {
		t.Fatal("initFlags() did not register -v")
	}
	if flag.Lookup("version") == nil {
		t.Fatal("initFlags() did not register -version")
	}
	if flag.Usage == nil {
		t.Fatal("initFlags() did not set usage function")
	}
}

func TestProcessStdin(t *testing.T) {
	t.Run("adds uid and writes to stdout", func(t *testing.T) {
		setModes(t, false, false)
		setStdinToString(t, "# hello")
		stdoutPath, _ := captureOutputFiles(t)

		if err := processStdin(); err != nil {
			t.Fatalf("processStdin() error = %v", err)
		}

		got, err := os.ReadFile(stdoutPath) //nolint:gosec
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(got), "uid:") {
			t.Fatal("processStdin() output missing uid")
		}
	})

	t.Run("verbose reports when uid already present", func(t *testing.T) {
		setModes(t, false, true)
		setStdinToString(t, "---\nuid: 11111111-1111-4111-8111-111111111111\n---\n# hello")
		_, stderrPath := captureOutputFiles(t)

		if err := processStdin(); err != nil {
			t.Fatalf("processStdin() error = %v", err)
		}

		got, err := os.ReadFile(stderrPath) //nolint:gosec
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(got), "uid already present") {
			t.Fatalf("processStdin() stderr = %q, want 'uid already present'", string(got))
		}
	})
}

func testProcessPathDirectoryRequiresRecursive(t *testing.T) {
	t.Helper()
	setModes(t, false, false)

	err := processPath(t.TempDir())
	if err == nil {
		t.Fatal("processPath() expected directory error without recursive mode")
	}
	if !strings.Contains(err.Error(), "use -r") {
		t.Fatalf("processPath() error = %v, want hint for -r", err)
	}
}

func testProcessPathProcessesFile(t *testing.T) {
	t.Helper()
	setModes(t, false, false)

	f := filepath.Join(t.TempDir(), "doc.md")
	if err := os.WriteFile(f, []byte("# hello"), filePermissions); err != nil {
		t.Fatal(err)
	}

	if err := processPath(f); err != nil {
		t.Fatalf("processPath() error = %v", err)
	}

	got, err := os.ReadFile(f) //nolint:gosec
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "uid:") {
		t.Fatal("processPath() did not update markdown file")
	}
}

func testProcessPathRejectsSymlink(t *testing.T) {
	t.Helper()
	setModes(t, false, false)

	dir := t.TempDir()
	target := filepath.Join(dir, "target.md")
	link := filepath.Join(dir, "link.md")
	if err := os.WriteFile(target, []byte("# hello"), filePermissions); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("os.Symlink not supported: %v", err)
	}

	err := processPath(link)
	if err == nil {
		t.Fatal("processPath() expected symlink error")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("processPath() error = %v, want symlink error", err)
	}
}

func TestProcessPath(t *testing.T) {
	t.Run("directory requires recursive mode", func(t *testing.T) {
		testProcessPathDirectoryRequiresRecursive(t)
	})

	t.Run("file path is processed", func(t *testing.T) {
		testProcessPathProcessesFile(t)
	})

	t.Run("symlink path is rejected", func(t *testing.T) {
		testProcessPathRejectsSymlink(t)
	})
}

func testProcessDirectoryProcessesMarkdown(t *testing.T) {
	t.Helper()

	setModes(t, true, false)

	dir := t.TempDir()
	mdPath := filepath.Join(dir, "doc.md")
	txtPath := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(mdPath, []byte("# hello"), filePermissions); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(txtPath, []byte("plain text"), filePermissions); err != nil {
		t.Fatal(err)
	}

	if err := processDirectory(dir); err != nil {
		t.Fatalf("processDirectory() error = %v", err)
	}

	mdGot, err := os.ReadFile(mdPath) //nolint:gosec
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mdGot), "uid:") {
		t.Fatal("processDirectory() did not process markdown file")
	}

	txtGot, err := os.ReadFile(txtPath) //nolint:gosec
	if err != nil {
		t.Fatal(err)
	}
	if string(txtGot) != "plain text" {
		t.Fatal("processDirectory() should not modify non-markdown file")
	}
}

func testProcessDirectorySkipsSymlinkEntries(t *testing.T) {
	t.Helper()

	setModes(t, true, false)

	dir := t.TempDir()

	target := filepath.Join(dir, "target.txt")
	link := filepath.Join(dir, "symlink.md")
	if err := os.WriteFile(target, []byte("# target"), filePermissions); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("os.Symlink not supported: %v", err)
	}

	if err := processDirectory(dir); err != nil {
		t.Fatalf("processDirectory() error = %v", err)
	}

	targetGot, err := os.ReadFile(target) //nolint:gosec
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(targetGot), "uid:") {
		t.Fatal("processDirectory() should skip symlink entries and not modify symlink target")
	}
}

func testProcessDirectoryAggregatesErrors(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	badPath := filepath.Join(dir, "bad.md")
	if err := os.WriteFile(badPath, []byte("---\ntitle: bad\n# no close"), filePermissions); err != nil {
		t.Fatal(err)
	}

	err := processDirectory(dir)
	if err == nil {
		t.Fatal("processDirectory() expected error for invalid markdown")
	}
	if !strings.Contains(err.Error(), "bad.md") {
		t.Fatalf("processDirectory() error = %v, want bad filename", err)
	}
}

func TestProcessDirectory(t *testing.T) {
	t.Run("processes markdown and skips non-markdown", func(t *testing.T) {
		testProcessDirectoryProcessesMarkdown(t)
	})

	t.Run("skips symlink entries", func(t *testing.T) {
		testProcessDirectorySkipsSymlinkEntries(t)
	})

	t.Run("aggregates processing errors", func(t *testing.T) {
		testProcessDirectoryAggregatesErrors(t)
	})
}

func TestProcessFile(t *testing.T) {
	t.Run("verbose reports uid added", func(t *testing.T) {
		setModes(t, false, true)
		path := filepath.Join(t.TempDir(), "doc.md")
		if err := os.WriteFile(path, []byte("# hello"), filePermissions); err != nil {
			t.Fatal(err)
		}

		_, stderrPath := captureOutputFiles(t)
		if err := processFile(path); err != nil {
			t.Fatalf("processFile() error = %v", err)
		}

		got, err := os.ReadFile(stderrPath) //nolint:gosec
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(got), "uid added") {
			t.Fatalf("processFile() stderr = %q, want 'uid added'", string(got))
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		err := processFile(filepath.Join(t.TempDir(), "missing.md"))
		if err == nil {
			t.Fatal("processFile() expected read error")
		}
		if !strings.Contains(err.Error(), "failed to stat file") {
			t.Fatalf("processFile() error = %v, want stat error", err)
		}
	})
}
