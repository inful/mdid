// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/inful/mdid"
)

// version is a var, not a const, so the release build can stamp the real
// version into it with -ldflags "-X main.version=...". Builds without those
// flags (go install, local builds) report "dev" rather than a number that
// goes stale.
var version = "dev"

// exitCodeUsage is the conventional exit code for "you called me wrong".
const exitCodeUsage = 2

var (
	recursiveMode bool
	verboseMode   bool
	showVersion   bool
)

func initFlags() {
	flag.BoolVar(&recursiveMode, "r", false, "Process directories recursively")
	flag.BoolVar(&verboseMode, "v", false, "Verbose output")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.Usage = usage
}

// permuteArgs moves flags in front of paths, keeping the order within each
// group. A bare "-" is a path, not a flag, and everything from "--" onwards is
// left where it is; the separator itself moves with the flags.
func permuteArgs(args []string) []string {
	flags := make([]string, 0, len(args))
	paths := make([]string, 0, len(args))
	afterSeparator := false
	hadSeparator := false

	for _, arg := range args {
		switch {
		case afterSeparator:
			paths = append(paths, arg)
		case arg == "--":
			afterSeparator, hadSeparator = true, true
		case len(arg) > 1 && strings.HasPrefix(arg, "-"):
			flags = append(flags, arg)
		default:
			paths = append(paths, arg)
		}
	}

	// The flag package stops at the first non-flag argument, so a separator
	// written after a path has to be re-emitted straight after the flags.
	if hadSeparator && len(paths) > 0 {
		return append(append(flags, "--"), paths...)
	}

	return append(flags, paths...)
}

func usage() {
	fmt.Fprintf(os.Stderr, "mdid - Markdown ID Tool %s\n\n", version)
	fmt.Fprintf(os.Stderr, "Usage: mdid [options] <file|directory>...\n")
	fmt.Fprintf(os.Stderr, "       mdid [options]            # Read from stdin, write to stdout\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  mdid file.md                  # Add uid to a single file if missing\n")
	fmt.Fprintf(os.Stderr, "  mdid -r docs/                 # Process all .md files in docs/ recursively\n")
	fmt.Fprintf(os.Stderr, "  mdid -v -r .                  # Process all .md files verbosely\n")
	fmt.Fprintf(os.Stderr, "  cat file.md | mdid            # Read from stdin, write to stdout\n")
}

func main() {
	initFlags()
	// Go's flag package stops at the first non-flag, so `mdid -r docs/ -v`
	// would read -v as a path. All three flags are boolean, so moving the
	// flags in front of the paths is safe.
	if err := flag.CommandLine.Parse(permuteArgs(os.Args[1:])); err != nil {
		usage()
		os.Exit(exitCodeUsage)
	}

	if showVersion {
		_, _ = fmt.Fprintf(os.Stdout, "mdid version %s\n", version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		// With no arguments mdid reads stdin. On a terminal that looks like a
		// hang until the user finds Ctrl-D, so print the usage instead.
		if isInteractiveTerminal() {
			usage()
			os.Exit(exitCodeUsage)
		}
		if err := processStdin(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	exitCode := 0
	for _, path := range args {
		if err := processPath(path); err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", path, err)
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}

func processPath(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("'%s' is a symlink (skipped)", path)
	}

	if info.IsDir() {
		if !recursiveMode {
			return fmt.Errorf("'%s' is a directory (use -r for recursive processing)", path)
		}
		return processDirectory(path)
	}

	// The walk only touches markdown, and so must an explicitly named file:
	// writing frontmatter into a Makefile breaks it.
	if !isMarkdown(path) {
		return fmt.Errorf("'%s' is not a markdown file", path)
	}

	_, err = processFile(path)

	return err
}

// isMarkdown reports whether path has a markdown extension.
func isMarkdown(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown":
		return true
	default:
		return false
	}
}

// isInteractiveTerminal reports whether stdin and stdout are the same terminal.
// Both ends must be character devices and the same device: a lone
// `mdid < /dev/null > out.md` must be treated as a redirection, not as a
// terminal, even though /dev/null is itself a character device.
func isInteractiveTerminal() bool {
	in, err := os.Stdin.Stat()
	if err != nil || in.Mode()&os.ModeCharDevice == 0 {
		return false
	}

	out, err := os.Stdout.Stat()
	if err != nil || out.Mode()&os.ModeCharDevice == 0 {
		return false
	}

	return os.SameFile(in, out)
}

func processStdin() error {
	// shortcut: io.ReadAll grows from 512 bytes and recopies as it goes, and
	// mdfm then copies the body again, so a very large piped document costs a
	// few times its size in memory. Fine for markdown; add a size limit if this
	// is ever pointed at unbounded input.
	content, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read from stdin: %w", err)
	}
	original := string(content)

	processed, err := mdid.ProcessContent(original)
	if err != nil {
		return fmt.Errorf("failed to process content: %w", err)
	}

	if verboseMode {
		if processed == original {
			fmt.Fprintf(os.Stderr, "[ok] stdin: uid already present\n")
		} else {
			fmt.Fprintf(os.Stderr, "[ok] stdin: uid added\n")
		}
	}

	_, err = os.Stdout.WriteString(processed)
	return err
}

func processDirectory(dir string) error {
	var errs []string
	var skipped int

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// One unreadable directory must not end the walk: keep going and
			// report it with the other failures, or the rest of the tree is
			// silently left without a uid.
			errs = append(errs, err.Error())
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 || d.IsDir() || !isMarkdown(path) {
			return nil
		}
		leftAlone, err := processFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))

			return nil
		}
		if leftAlone {
			skipped++
		}

		return nil
	})
	if err != nil {
		errs = append(errs, err.Error())
	}

	// The per-file [skip] lines scroll away in a long run, so say how many
	// there were: that is the number a caller needs.
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "[skip] %d file(s) left unchanged\n", skipped)
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

// processFile adds a uid to one file. It reports whether the file was left
// alone on purpose, which is not a failure.
func processFile(path string) (bool, error) {
	var hadUID bool
	// shortcut: -v reads and parses each file a second time to decide which
	// message to print, which roughly triples a no-op run over a tree of
	// 2,000 files. The alternatives are worse: a ProcessFile that reports
	// whether it wrote is public API growth, and an mtime comparison is wrong
	// on filesystems with one-second timestamps. Drop this when either is
	// worth it.
	if verboseMode {
		if content, err := os.ReadFile(path); err == nil { //nolint:gosec
			// Use the library's parser rather than a substring grep so a "uid:"
			// in the body of the document (e.g. inside a code fence) cannot
			// fool the verbose output.
			if has, hasErr := mdid.HasUID(string(content)); hasErr == nil {
				hadUID = has
			}
		}
	}

	if err := mdid.ProcessFile(path); err != nil {
		// A deliberate refusal is not a failure: mdid will not write a uid it
		// cannot safely write. Report it, but do not fail the run over it.
		if errors.Is(err, mdid.ErrMergeKey) {
			fmt.Fprintf(os.Stderr, "[skip] %s: frontmatter uses a YAML merge key, left unchanged\n", path)
			return true, nil
		}
		if errors.Is(err, mdid.ErrAnchorInUse) {
			fmt.Fprintf(os.Stderr, "[skip] %s: the uid key defines an anchor another key aliases, left unchanged\n", path)
			return true, nil
		}

		return false, err
	}

	if verboseMode {
		if hadUID {
			fmt.Fprintf(os.Stderr, "[ok] %s: uid already present\n", path)
		} else {
			fmt.Fprintf(os.Stderr, "[ok] %s: uid added\n", path)
		}
	}

	return false, nil
}
