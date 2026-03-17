package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/inful/mdid"
)

const version = "0.1.0"

const filePermissions = 0o600

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

func usage() {
	fmt.Fprintf(os.Stderr, "mdid - Markdown ID Tool v%s\n\n", version)
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
	flag.Parse()

	if showVersion {
		fmt.Fprintf(os.Stderr, "mdid version %s\n", version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
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

	return processFile(path)
}

func processStdin() error {
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

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}
		if err := processFile(path); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

func processFile(path string) error {
	var hadUID bool
	if verboseMode {
		if content, err := os.ReadFile(path); err == nil { //nolint:gosec
			hadUID = strings.Contains(string(content), "uid:")
		}
	}

	if err := mdid.ProcessFile(path); err != nil {
		return err
	}

	if verboseMode {
		if hadUID {
			fmt.Fprintf(os.Stderr, "[ok] %s: uid already present\n", path)
		} else {
			fmt.Fprintf(os.Stderr, "[ok] %s: uid added\n", path)
		}
	}

	return nil
}
