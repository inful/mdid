# mdid

mdid (Markdown ID) is a Go library and command-line tool for adding unique identifiers
to markdown files. It generates a UUID v7 and stores it in the YAML frontmatter under a
`uid` field — but only if no usable uid is already there. Once assigned, the uid
is never overwritten, making it a stable identifier for the document. A `uid:`
key with no usable value (empty, `null`, a number) counts as missing and is
filled in.

## Features

- **Pure Go library** — UUID v7 is generated in-package from `crypto/rand`; `github.com/inful/mdfm` handles frontmatter parsing/mutation
- **Stable identifiers** — Adds a `uid` only when missing; never overwrites an existing one
- **Fills placeholders** — A `uid:` key with no usable value (empty, `null`, a number) is treated as missing and gets a real identifier
- **UUID v7** — Time-sortable identifiers
- **Timestamp strategy** — `ProcessFile` uses file `mtime`; `ProcessContent` uses current time unless `ProcessContentAtTime` is used
- **Automatic frontmatter management** — Creates frontmatter if the file has none
- **Command-line tool** — Easy-to-use CLI for processing files and directories
- **Recursive processing** — Process entire directory trees
- **Pipeline/stdin-stdout support** — Works seamlessly in Unix pipelines
- **Well-tested** — Comprehensive test coverage

## Installation

### Using Go install

```sh
go install github.com/inful/mdid/cmd/mdid@latest
```

### From source

```sh
git clone https://github.com/inful/mdid.git
cd mdid
go build -o mdid ./cmd/mdid
```

### Download pre-built binaries

Download the latest release from the [releases page](https://github.com/inful/mdid/releases).

## Usage

### Command Line

#### Process a single file

```sh
mdid document.md
```

If the file has no frontmatter or is missing a `uid`, this will add one:

```markdown
---
uid: 018f1f10-6d3b-7c7f-8a3d-3f9f2e5b8c41
title: My Document
---

# Content
```

If the file already contains a usable `uid`, it is left completely untouched.

#### Process multiple files

```sh
mdid file1.md file2.md file3.md
```

Files named on the command line must have a markdown extension (`.md` or
`.markdown`) too, so `mdid Makefile` fails instead of writing frontmatter into
a build file.

#### Process directory recursively

```sh
mdid -r docs/
```

A directory given on the command line must be walked with `-r`, and it must not
be a symlink. Naming a file inside a symlinked directory works and writes
through to the real directory.

Flags may come before or after the paths: `mdid -r docs/ -v` behaves like
`mdid -v -r docs/`.

#### Verbose output

```sh
mdid -v -r .
```

Each file is read and parsed twice in verbose mode, so `-v` roughly triples the
time of a run over a tree whose files already have a uid.

#### Pipeline/stdin-stdout usage

```sh
# Add uid if missing and output to stdout
cat document.md | mdid > output.md

# Process in a pipeline
curl https://example.com/document.md | mdid | gzip > output.md.gz
```

With no file arguments **and a terminal on both stdin and stdout**, `mdid`
prints its usage and exits 2 rather than waiting for you to press Ctrl-D. A pipe
or a redirection such as `mdid < /dev/null > output.md` is processed normally.

#### Exit codes

| Code | Meaning |
| ---- | ------- |
| 0 | Everything asked for was processed, or deliberately left alone |
| 1 | At least one file, or the input, could not be processed; the rest still were |
| 2 | Called with no arguments on a terminal; the usage was printed |

#### Get help

```sh
mdid -h
```

### As a Library

```go
package main

import (
    "log"

    "github.com/inful/mdid"
)

func main() {
    // Process a single file (adds uid if missing)
    err := mdid.ProcessFile("document.md")
    if err != nil {
        log.Fatal(err)
    }

    // Process content directly
    content := `---
title: Test
---
# Hello World`

    processed, err := mdid.ProcessContent(content)
    if err != nil {
        log.Fatal(err)
    }
    _ = processed
}
```

## API Documentation

### Core Functions

#### `ProcessFile(filepath string) error`

Reads a markdown file, adds a `uid` if missing, and writes it back. If the file
already contains a usable `uid`, it is left untouched.

#### `ProcessContent(content string) (string, error)`

Processes markdown content and returns it with a `uid` added to the frontmatter.
Returns the content unchanged if a usable `uid` is already present.

#### `HasUID(content string) (bool, error)`

Reports whether the markdown content's frontmatter contains a `uid` field with a
value. Returns `false` for content with no frontmatter, no `uid`, or a
placeholder `uid` (empty, `null`, or a number). Useful when callers want to
inspect a document without mutating it (and as the canonical "is this a known
document?" check before any batch processing).

It returns an error when the frontmatter cannot be parsed, and
[`ErrMergeKey`](#errmergekey) when the frontmatter uses a YAML merge key with no
`uid` of its own — the answer would be a guess, so callers must check for it
with `errors.Is` rather than treating every error as malformed input.

#### `ProcessContentAtTime(content string, t time.Time) (string, error)`

Processes markdown content using an explicit timestamp for the UUID v7. Useful when you
have a known reference time for the content (e.g. the mtime of the originating file).
Returns the content unchanged if a usable `uid` is already present.

#### `ProcessDocument(doc FrontmatterDocument) error`

Adds a `uid` to an already-parsed `*mdfm.Document` (or any type satisfying
`FrontmatterDocument`) if one is not present. Mutates the document in place;
the caller serializes when ready. Uses the current time as the UUID v7 timestamp.

#### `ProcessDocumentAtTime(doc FrontmatterDocument, t time.Time) error`

Same as `ProcessDocument`, but uses an explicit timestamp for the UUID v7.

#### `FrontmatterDocument` interface

The minimal interface satisfied by `*mdfm.Document`:

```go
type FrontmatterDocument interface {
    Has(key string) (bool, error)
    Get(key string) (any, bool, error)
    SetString(key, value string) error
}
```

Any type implementing these three methods can be passed to `ProcessDocument`
and `ProcessDocumentAtTime`, so callers that already hold a parsed document
avoid a second parse. `Get` is needed to tell a real identifier from a
placeholder `uid:` key.

#### `ErrAnchorInUse`

Returned by `ProcessContent`, `ProcessFile` and `ProcessDocument` when the `uid`
key defines a YAML anchor that another key still aliases. Filling in a
placeholder would produce frontmatter that no longer parses, so the document is
left untouched. Match it with `errors.Is`; the CLI reports it as a `[skip]`.

#### `ErrMergeKey`

Returned by `HasUID`, `ProcessContent`, `ProcessFile` and `ProcessDocument` when
the frontmatter uses a YAML merge key (`<<: *anchor`) and has no literal `uid`
of its own. The merged keys may already supply a uid, and mdid cannot see it
without expanding YAML merges, so it refuses instead of adding an identifier
that would shadow the inherited one. Test for it with `errors.Is(err,
mdid.ErrMergeKey)`; the file or document is left untouched. The CLI prints
`[skip]` for such a file and continues, so a run over a tree containing one is
not an error.

#### `UIDField`

The frontmatter key mdid reads and writes: the constant string `"uid"`. Use it
instead of repeating the literal when you need to look the identifier up in a
parsed document.

#### `GenerateUID() string`

Returns a new UUID v7 string timestamped at the current time.

#### `GenerateUIDAtTime(t time.Time) string`

Returns a UUID v7 string with the given time embedded as the millisecond-precision
timestamp (RFC 9562, §5.7). The remaining bits are cryptographically random.

## How It Works

1. **Parse** — Extracts YAML frontmatter (between `---` delimiters) from the markdown file
2. **Check** — Looks for an existing `uid` field in the frontmatter
3. **Generate** — If no `uid` is found, generates a UUID v7
4. **Update** — Adds a `uid` field to the frontmatter if missing
5. **Write** — Reconstructs the file with updated frontmatter

The UUID v7 timestamp is sourced from the file's modification time (`mtime`) when
processing files via `ProcessFile`, so documents are time-sortable by when they were
last edited regardless of when `mdid` was run. `ProcessContent` uses the current time,
and `ProcessContentAtTime` accepts an explicit timestamp for full control.

The `uid` is inserted into frontmatter through `mdfm`'s YAML-aware mutation APIs,
and once set it is never changed, making it a reliable long-term identifier for the document.

## What changes in your files

Adding a `uid` rewrites the whole frontmatter block, so the first run can produce
a larger diff than the one new line:

- **Nested YAML is re-indented** from 2 spaces to 4. `- one` becomes `    - one`.
  The YAML means exactly the same thing; the text does not.
- Comments, quoting style, long lines, dates and literal blocks are preserved.
- A YAML merge key comes back with its explicit tag: `<<: *defaults` becomes
  `!!merge <<: *defaults`. It still merges the same way.
- Line endings (LF or CRLF), the file's permissions and a UTF-8 BOM are
  preserved, and a block closed with `...` keeps its `...`.
- **Extended attributes, hard links and the file's identity are preserved.**
  The file is rewritten in place, so macOS Finder tags, quarantine flags and
  similar metadata stay, and every hard link to the file sees the new content.
- The write is **not atomic**: the file is written and only then truncated, so a
  write that fails partway leaves the file at its original length rather than a
  short one, but mdid killed mid-write can leave a partly written file.
- A read-only markdown file (mode `0444`, or owned by another user) **cannot be
  updated**: the write happens in place, so mdid needs write permission on the
  file itself and not only on its directory.
- Files that already have a usable `uid` are not touched at all, so later runs
  are no-ops.

## Frontmatter mdid will not change

If the frontmatter uses a YAML merge key (`<<: *anchor`) and has no `uid` key
of its own, mdid cannot see whether the merged keys already supply one. It
prints `[skip] <file>: frontmatter uses a YAML merge key, left unchanged` and
leaves the file alone rather than adding an identifier that would shadow the
inherited one. A skipped file is not a failure: the run still exits 0, and no
other file is affected. Add a literal `uid:` to the frontmatter, or drop the
merge key, to let mdid work on such a file.

The same happens when the placeholder `uid` key defines a YAML anchor that
another key aliases (`uid: &u ""` plus `alias: *u`): filling it in would leave
frontmatter that no longer parses, so mdid prints `[skip] <file>: the uid key
defines an anchor another key aliases` and leaves the file alone. Library
callers get `mdid.ErrMergeKey` and `mdid.ErrAnchorInUse` from the same calls,
so both refusals can be matched with `errors.Is`.

Because a skipped file keeps whatever uid it has and does not get one of its
own, exit code 0 means "every markdown file was processed or deliberately left
alone", not "every file now has a uid". Each skipped file gets a `[skip]` line,
and a run that skipped any ends with `[skip] N file(s) left unchanged`, which
is the number to look for in a long log or in CI output.

## Development

### Running Tests

```sh
go test -v
```

### Building

```sh
go build -o mdid ./cmd/mdid
```

### Running Linter

```sh
golangci-lint run ./...
```

## License

This project is licensed under the GNU General Public License v3.0 or later
(GPL-3.0-or-later). Every Go file carries
`// SPDX-License-Identifier: GPL-3.0-or-later`, so license scanners read the same
grant as this section; see the [LICENSE](LICENSE) file for the full text.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests and linter (`go test ./... && golangci-lint run ./...`)
4. Commit your changes (`git commit -m 'Add some amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request
