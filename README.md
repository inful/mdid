# mdid

mdid (Markdown ID) is a Go library and command-line tool for adding unique identifiers
to markdown files. It generates a UUID v7 and stores it in the YAML frontmatter under a
`uid` field — but only if one is not already present. Once assigned, the uid is never
overwritten, making it a stable identifier for the document.

## Features

- **Pure Go library** — Uses `github.com/google/uuid` for UUID generation and `github.com/inful/mdfm` for frontmatter parsing/mutation
- **Stable identifiers** — Adds a `uid` only when missing; never overwrites an existing one
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

If the file already contains a `uid`, it is left completely untouched.

#### Process multiple files

```sh
mdid file1.md file2.md file3.md
```

#### Process directory recursively

```sh
mdid -r docs/
```

#### Verbose output

```sh
mdid -v -r .
```

#### Pipeline/stdin-stdout usage

```sh
# Add uid if missing and output to stdout
cat document.md | mdid > output.md

# Process in a pipeline
curl https://example.com/document.md | mdid | gzip > output.md.gz
```

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
already contains a `uid`, it is left untouched.

#### `ProcessContent(content string) (string, error)`

Processes markdown content and returns it with a `uid` added to the frontmatter.
Returns the content unchanged if a `uid` is already present.

#### `ProcessContentAtTime(content string, t time.Time) (string, error)`

Processes markdown content using an explicit timestamp for the UUID v7. Useful when you
have a known reference time for the content (e.g. the mtime of the originating file).
Returns the content unchanged if a `uid` is already present.

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
    SetString(key, value string) error
}
```

Any type implementing these two methods can be passed to `ProcessDocument` and
`ProcessDocumentAtTime`, so callers that already hold a parsed document avoid a
second parse.

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
(GPL-3.0-or-later) — see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests and linter (`go test ./... && golangci-lint run ./...`)
4. Commit your changes (`git commit -m 'Add some amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request
