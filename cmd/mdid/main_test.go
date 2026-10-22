package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// filePermissions keeps the files these tests create private to the test user.
const filePermissions = 0o600

// TestIsInteractiveTerminal is the regression test for `mdid < /dev/null`,
// which must be treated as a redirection even though /dev/null is a character
// device.
func TestIsInteractiveTerminal(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = devNull.Close() })

	regular := filepath.Join(t.TempDir(), "file.md")
	if writeErr := os.WriteFile(regular, []byte("# hello"), filePermissions); writeErr != nil {
		t.Fatal(writeErr)
	}
	regularFile, err := os.Open(regular) //nolint:gosec // path is built by the test
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = regularFile.Close() })

	for _, tt := range []struct {
		name          string
		stdin, stdout *os.File
	}{
		{name: "null stdin is a redirect, not a terminal", stdin: devNull, stdout: regularFile},
		{name: "a plain file on both ends is not a terminal", stdin: regularFile, stdout: regularFile},
	} {
		t.Run(tt.name, func(t *testing.T) {
			oldIn, oldOut := os.Stdin, os.Stdout
			os.Stdin, os.Stdout = tt.stdin, tt.stdout
			t.Cleanup(func() { os.Stdin, os.Stdout = oldIn, oldOut })

			if isInteractiveTerminal() {
				t.Error("isInteractiveTerminal() = true, want false")
			}
		})
	}
}

func TestUsageListsEveryFlag(t *testing.T) {
	resetFlagSet(t)
	initFlags()

	_, stderrPath := captureOutputFiles(t)
	usage()

	got := readFile(t, stderrPath)
	for _, name := range []string{"-r", "-v", "-version"} {
		if !strings.Contains(got, name) {
			t.Errorf("usage() output does not mention %s:\n%s", name, got)
		}
	}
}

// docsDir is the path used by the permutation cases below.
const docsDir = "docs"

func TestPermuteArgs(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want []string
	}{
		{name: "already in order", args: []string{"-v", "-r", docsDir}, want: []string{"-v", "-r", docsDir}},
		{name: "flag after a path", args: []string{"-r", docsDir, "-v"}, want: []string{"-r", "-v", docsDir}},
		{name: "no flags", args: []string{"a.md", "b.md"}, want: []string{"a.md", "b.md"}},
		{name: "a bare dash is a path", args: []string{"-"}, want: []string{"-"}},
		{name: "everything after -- stays", args: []string{"-v", "--", "-weird.md"}, want: []string{"-v", "--", "-weird.md"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := permuteArgs(tt.args)
			if strings.Join(got, " ") != strings.Join(tt.want, " ") {
				t.Errorf("permuteArgs(%q) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

// Finding 2: naming a non-markdown file must not touch it.
func TestProcessPathRefusesNonMarkdown(t *testing.T) {
	setModes(t, false, false)

	for _, name := range []string{"Makefile", "notes.txt", "script.sh", "no-extension"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name)
			if err := os.WriteFile(path, []byte("all:\n\techo hi\n"), filePermissions); err != nil {
				t.Fatal(err)
			}

			err := processPath(path)
			if err == nil {
				t.Fatalf("processPath(%q) = nil, want a not-a-markdown error", name)
			}
			if !strings.Contains(err.Error(), "not a markdown file") {
				t.Errorf("processPath(%q) error = %v, want 'not a markdown file'", name, err)
			}
			if got := readFile(t, path); got != "all:\n\techo hi\n" {
				t.Errorf("processPath(%q) modified the file: %q", name, got)
			}
		})
	}
}

func TestProcessPathAcceptsMarkdownExtensions(t *testing.T) {
	setModes(t, false, false)

	for _, name := range []string{"doc.md", "DOC.MD", "doc.markdown"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name)
			if err := os.WriteFile(path, []byte("# hello"), filePermissions); err != nil {
				t.Fatal(err)
			}
			if err := processPath(path); err != nil {
				t.Fatalf("processPath(%q) error = %v", name, err)
			}
			if got := readFile(t, path); !strings.Contains(got, "uid:") {
				t.Errorf("processPath(%q) did not add a uid", name)
			}
		})
	}
}

// The two messages the CLI prints in verbose mode.
const (
	msgUIDAdded   = "uid added"
	msgUIDPresent = "uid already present"
)

// cliPath is the compiled CLI. TestMain builds it once so the tests can check
// the real exit code and stdout/stderr split that main() produces, which no
// in-process test can see because main() calls os.Exit.
var cliPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "mdid-cli-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if _, err := exec.LookPath("go"); err != nil {
		// No toolchain, e.g. a test binary compiled elsewhere and run in a
		// scratch container: skip what needs the CLI instead of failing all.
		fmt.Fprintln(os.Stderr, "go not found in PATH: skipping the tests that build the CLI")
		os.Exit(m.Run())
	}

	built, buildErr := buildCLI(dir)
	if buildErr != nil {
		fmt.Fprintln(os.Stderr, buildErr)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	cliPath = built

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// buildCLI compiles the CLI into dir, with a timeout so a stalled build fails
// the test run instead of hanging it.
func buildCLI(dir string) (string, error) {
	path := filepath.Join(dir, "mdid")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	//nolint:gosec // fixed command; only the output path comes from this test
	out, err := exec.CommandContext(ctx, "go", "build", "-o", path, ".").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go build: %w\n%s", err, out)
	}

	return path, nil
}

// TestMainContract covers the only parts of the tool a script can see: the exit
// code, what lands on stdout, and whether a named file was really updated.
// cliCase is one end-to-end run of the CLI. args and wantUids are names inside
// the case's own directory, so no case can affect another; a file in wantUids
// is created before the run.
type cliCase struct {
	name     string
	args     []string
	stdin    string
	wantCode int
	wantOut  string
	wantErr  string
	wantUids []string
}

func TestMainContract(t *testing.T) {
	for _, tt := range []cliCase{
		{name: "a good file succeeds", args: []string{"good.md"}, wantCode: 0, wantUids: []string{"good.md"}},
		{name: "a missing file fails", args: []string{"nope.md"}, wantCode: 1},
		{name: "a directory without -r fails", args: []string{"."}, wantCode: 1},
		{
			name:     "one bad path does not stop the rest",
			args:     []string{"nope.md", "second.md"},
			wantCode: 1,
			wantUids: []string{"second.md"},
		},
		{name: "-version goes to stdout", args: []string{"-version"}, wantCode: 0, wantOut: "mdid version"},
		{
			name:     "a flag after the path still works",
			args:     []string{"-v", "late.md", "-v"},
			wantCode: 0,
			wantErr:  msgUIDAdded,
			wantUids: []string{"late.md"},
		},
		{name: "stdin is echoed to stdout", stdin: "# hello", wantCode: 0, wantOut: "uid:"},
		{name: "malformed stdin exits 1", stdin: "---\ntitle: bad\n# no close", wantCode: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range tt.wantUids {
				if writeErr := os.WriteFile(filepath.Join(dir, name), []byte("# hello"), filePermissions); writeErr != nil {
					t.Fatal(writeErr)
				}
			}

			code, stdout, stderr := runCLI(t, resolvePaths(dir, tt.args), tt.stdin)
			checkCLIOutput(t, tt, code, stdout, stderr)
			for _, name := range tt.wantUids {
				if got := readFile(t, filepath.Join(dir, name)); !strings.Contains(got, "uid:") {
					t.Errorf("mdid %v left %s without a uid", tt.args, name)
				}
			}
		})
	}
}

// checkCLIOutput asserts the exit code and the output a case expects.
func checkCLIOutput(t *testing.T, tt cliCase, code int, stdout, stderr string) {
	t.Helper()

	if code != tt.wantCode {
		t.Errorf("mdid %v exited %d, want %d (stderr: %s)", tt.args, code, tt.wantCode, stderr)
	}
	if tt.wantOut != "" && !strings.Contains(stdout, tt.wantOut) {
		t.Errorf("mdid %v stdout = %q, want it to contain %q", tt.args, stdout, tt.wantOut)
	}
	if tt.wantErr != "" && !strings.Contains(stderr, tt.wantErr) {
		t.Errorf("mdid %v stderr = %q, want it to contain %q", tt.args, stderr, tt.wantErr)
	}
}

// resolvePaths turns the table's file names into paths under dir, leaving flags
// such as -version alone.
func resolvePaths(dir string, args []string) []string {
	resolved := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			resolved = append(resolved, arg)
			continue
		}
		resolved = append(resolved, filepath.Join(dir, arg))
	}

	return resolved
}

// runCLI runs the built CLI with args and stdin, and returns its exit code
// along with what it wrote to stdout and stderr.
func runCLI(t *testing.T, args []string, stdin string) (int, string, string) {
	t.Helper()

	if cliPath == "" {
		t.Skip("no Go toolchain available to build the CLI")
	}

	//nolint:gosec // args come from this file's own table, paths are t.TempDir()
	cmd := exec.CommandContext(t.Context(), cliPath, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	code := 0
	if err := cmd.Run(); err != nil {
		exitErr := &exec.ExitError{}
		if !errors.As(err, &exitErr) {
			t.Fatalf("running mdid: %v", err)
		}
		code = exitErr.ExitCode()
	}

	return code, stdout.String(), stderr.String()
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path) //nolint:gosec // path is built by the test
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}

	return string(content)
}

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
		if !strings.Contains(string(got), msgUIDPresent) {
			t.Fatalf("processStdin() stderr = %q, want %q", got, msgUIDPresent)
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

// One unreadable directory used to end the walk, leaving the rest of the tree
// without a uid and hiding the errors already collected.
func testProcessDirectoryWalksPastUnreadableDir(t *testing.T) {
	t.Helper()

	setModes(t, true, false)

	dir := t.TempDir()
	blocked := filepath.Join(dir, "m_blocked")
	if err := os.Mkdir(blocked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blocked, "hidden.md"), []byte("# hidden"), filePermissions); err != nil {
		t.Fatal(err)
	}
	after := filepath.Join(dir, "z_after.md")
	if err := os.WriteFile(after, []byte("# after"), filePermissions); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(blocked, 0o000); err != nil {
		t.Fatal(err)
	}
	// 0700, not 0600: a directory needs the execute bit to be traversable at cleanup.
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o700) }) //nolint:gosec // directory, not a file
	if _, err := os.ReadDir(blocked); err == nil {
		t.Skip("directory permissions are not enforced for this user")
	}

	err := processDirectory(dir)
	if err == nil {
		t.Fatal("processDirectory() expected an error for the unreadable directory")
	}
	if !strings.Contains(err.Error(), blocked) {
		t.Errorf("processDirectory() error = %v, want it to name %s", err, blocked)
	}

	afterGot, readErr := os.ReadFile(after) //nolint:gosec
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(afterGot), "uid:") {
		t.Error("processDirectory() stopped walking at the unreadable directory")
	}
}

func TestProcessDirectory(t *testing.T) {
	t.Run("keeps walking past an unreadable directory", func(t *testing.T) {
		testProcessDirectoryWalksPastUnreadableDir(t)
	})

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

func TestProcessFileVerbose(t *testing.T) {
	for _, tt := range []struct {
		name    string
		content string
		want    string
	}{
		{name: "reports an added uid", content: "# hello", want: msgUIDAdded},
		{
			name:    "reports a uid already present",
			content: "---\nuid: 11111111-1111-4111-8111-111111111111\n---\n# hello",
			want:    msgUIDPresent,
		},
		{name: "reports a filled placeholder as added", content: "---\nuid:\n---\n# hello", want: msgUIDAdded},
		{
			// A "uid:" inside a code fence in the body must not fool the
			// verbose output into thinking the file has a frontmatter uid.
			name:    "ignores a uid in the body",
			content: "# hello\n\n```\nuid: not-a-real-uid\n```\n",
			want:    msgUIDAdded,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			setModes(t, false, true)
			path := filepath.Join(t.TempDir(), "doc.md")
			if err := os.WriteFile(path, []byte(tt.content), filePermissions); err != nil {
				t.Fatal(err)
			}

			_, stderrPath := captureOutputFiles(t)
			if err := processFile(path); err != nil {
				t.Fatalf("processFile() error = %v", err)
			}

			got := readFile(t, stderrPath)
			if !strings.Contains(got, tt.want) {
				t.Fatalf("processFile() stderr = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProcessFileReturnsErrorForMissingFile(t *testing.T) {
	err := processFile(filepath.Join(t.TempDir(), "missing.md"))
	if err == nil {
		t.Fatal("processFile() expected read error")
	}
	if !strings.Contains(err.Error(), "failed to stat file") {
		t.Fatalf("processFile() error = %v, want stat error", err)
	}
}
