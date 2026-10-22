package mdid

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/inful/mdfm"
)

// testFrontmatterInput is a minimal markdown document used across multiple tests.
const testFrontmatterInput = "---\ntitle: Test\n---\n# Content"

// uuidV7Re matches a canonical UUID v7 string.
var (
	uuidV7Re  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	uidLineRe = regexp.MustCompile(`(?m)^uid:\s*([^\s]+)\s*$`)
)

func isValidV7UUID(s string) bool {
	return uuidV7Re.MatchString(s)
}

// parseV7Timestamp decodes a canonical 36-char UUID string and returns the
// millisecond timestamp embedded in it.
func parseV7Timestamp(uid string) (time.Time, error) {
	clean := strings.ReplaceAll(uid, "-", "")
	if len(clean) != 32 {
		return time.Time{}, fmt.Errorf("invalid UUID format: %q", uid)
	}
	var b [16]byte
	if _, err := hex.Decode(b[:], []byte(clean)); err != nil {
		return time.Time{}, fmt.Errorf("invalid UUID format: %q", uid)
	}

	if b[6]>>4 != 7 {
		return time.Time{}, fmt.Errorf("expected UUID v7, got v%d", b[6]>>4)
	}
	ms := int64(b[0])<<40 | int64(b[1])<<32 | int64(b[2])<<24 |
		int64(b[3])<<16 | int64(b[4])<<8 | int64(b[5])
	return time.UnixMilli(ms).UTC(), nil
}

func getUIDFromContent(content string) string {
	matches := uidLineRe.FindStringSubmatch(content)
	if len(matches) != 2 {
		return ""
	}

	return matches[1]
}

// mustParse parses content, failing the test if it is not valid markdown.
func mustParse(t *testing.T, content string) *mdfm.Document {
	t.Helper()

	doc, err := mdfm.ParseString(content)
	if err != nil {
		t.Fatalf("mdfm.ParseString() error = %v", err)
	}

	return doc
}

// mustBytes serializes doc, failing the test if it cannot be serialized.
func mustBytes(t *testing.T, doc *mdfm.Document) string {
	t.Helper()

	out, err := doc.Bytes()
	if err != nil {
		t.Fatalf("Document.Bytes() error = %v", err)
	}

	return string(out)
}

// mustModTime returns a file's modification time.
func mustModTime(t *testing.T, path string) time.Time {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v", path, err)
	}

	return info.ModTime()
}

// mustReadFile reads a file the test just wrote.
func mustReadFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path) //nolint:gosec // path is built by the test
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}

	return string(content)
}

// The library keeps no shared state, so callers may process documents in
// parallel. This test is what keeps that true: run with -race, it fails the
// moment a package-level cache or counter appears.
func TestProcessContentInParallel(t *testing.T) {
	const workers = 8

	var wg sync.WaitGroup
	uids := make([]string, workers)
	for i := range workers {
		wg.Go(func() {
			out, err := ProcessContent(testFrontmatterInput)
			if err != nil {
				t.Errorf("ProcessContent() error = %v", err)
				return
			}
			uids[i] = getUIDFromContent(out)
		})
	}
	wg.Wait()

	seen := make(map[string]bool, workers)
	for i, uid := range uids {
		if !isValidV7UUID(uid) {
			t.Fatalf("worker %d produced uid %q, want a valid UUID v7", i, uid)
		}
		if seen[uid] {
			t.Fatalf("workers produced the same uid %q twice", uid)
		}
		seen[uid] = true
	}
}

func TestGenerateUID(t *testing.T) {
	uid := GenerateUID()
	if !isValidV7UUID(uid) {
		t.Errorf("GenerateUID() = %q, want valid UUID v7", uid)
	}
	uid2 := GenerateUID()
	if uid == uid2 {
		t.Error("GenerateUID() returned identical values on consecutive calls")
	}
}

func TestGenerateUIDAtTime(t *testing.T) {
	knownTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	uid := GenerateUIDAtTime(knownTime)

	if !isValidV7UUID(uid) {
		t.Errorf("GenerateUIDAtTime() = %q, want valid UUID v7", uid)
	}

	ts, err := parseV7Timestamp(uid)
	if err != nil {
		t.Fatalf("parseV7Timestamp() error = %v", err)
	}

	want := knownTime.Truncate(time.Millisecond).UTC()
	if !ts.Equal(want) {
		t.Errorf("GenerateUIDAtTime() timestamp = %v, want %v", ts, want)
	}

	// Two calls with the same time produce different UIDs (random component).
	uid2 := GenerateUIDAtTime(knownTime)
	if uid == uid2 {
		t.Error("GenerateUIDAtTime() returned identical UIDs for the same time")
	}
}

func TestGenerateUIDAtTimeClampsNegativeTime(t *testing.T) {
	// A zero time or a pre-1970 file mtime has no place in the 48 unsigned
	// bits of the timestamp field; it must clamp to the epoch instead of
	// wrapping into the far future.
	for _, tt := range []struct {
		name string
		when time.Time
	}{
		{name: "zero time", when: time.Time{}},
		{name: "before the epoch", when: time.Date(1969, 6, 15, 12, 0, 0, 0, time.UTC)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uid := GenerateUIDAtTime(tt.when)
			if !isValidV7UUID(uid) {
				t.Fatalf("GenerateUIDAtTime(%v) = %q, want valid UUID v7", tt.when, uid)
			}

			ts, err := parseV7Timestamp(uid)
			if err != nil {
				t.Fatalf("parseV7Timestamp() error = %v", err)
			}
			if ts.Before(time.UnixMilli(0)) || ts.After(time.Now().Add(time.Minute)) {
				t.Errorf("GenerateUIDAtTime(%v) timestamp = %v, want at or after the Unix epoch", tt.when, ts)
			}
		})
	}
}

func TestProcessContent(t *testing.T) { //nolint:cyclop
	t.Run("adds uid when missing with frontmatter", func(t *testing.T) {
		got, err := ProcessContent(testFrontmatterInput)
		if err != nil {
			t.Fatalf("ProcessContent() error = %v", err)
		}
		if !strings.Contains(got, "uid:") {
			t.Error("ProcessContent() output missing uid field")
		}
		uid := getUIDFromContent(got)
		if !isValidV7UUID(uid) {
			t.Errorf("ProcessContent() uid is not a valid UUID v7: %q", uid)
		}
		if !strings.Contains(got, "title: Test") {
			t.Error("ProcessContent() lost original frontmatter fields")
		}
	})

	t.Run("does not overwrite existing uid", func(t *testing.T) {
		const existingUID = "11111111-1111-4111-8111-111111111111"
		input := "---\nuid: " + existingUID + "\ntitle: Test\n---\n# Content"
		got, err := ProcessContent(input)
		if err != nil {
			t.Fatalf("ProcessContent() error = %v", err)
		}
		if got != input {
			t.Error("ProcessContent() modified content that already had a uid")
		}
		uid := getUIDFromContent(got)
		if uid != existingUID {
			t.Errorf("ProcessContent() uid = %q, want %q", uid, existingUID)
		}
	})

	t.Run("adds uid when no frontmatter", func(t *testing.T) {
		got, err := ProcessContent("# Just content")
		if err != nil {
			t.Fatalf("ProcessContent() error = %v", err)
		}
		if !strings.Contains(got, "uid:") {
			t.Error("ProcessContent() output missing uid field")
		}
	})

	t.Run("calling twice preserves uid", func(t *testing.T) {
		first, err := ProcessContent(testFrontmatterInput)
		if err != nil {
			t.Fatalf("first ProcessContent() error = %v", err)
		}
		second, err := ProcessContent(first)
		if err != nil {
			t.Fatalf("second ProcessContent() error = %v", err)
		}
		if first != second {
			t.Error("ProcessContent() changed content on second call (uid should be stable)")
		}
	})
}

// TestProcessContentFillsPlaceholderUID covers a uid key that exists but holds
// nothing usable: it is a placeholder, not an identifier, so it must be filled.
func TestHasUIDRejectsPlaceholderUID(t *testing.T) {
	for _, input := range []string{
		"---\nuid:\n---\n# Content",
		"---\nuid: \"\"\n---\n# Content",
		"---\nuid: null\n---\n# Content",
		"---\nuid: 12345\n---\n# Content",
	} {
		has, err := HasUID(input)
		if err != nil {
			t.Fatalf("HasUID(%q) error = %v", input, err)
		}
		if has {
			t.Errorf("HasUID(%q) = true, want false for a placeholder uid", input)
		}
	}
}

// A uid inherited through a YAML merge key must never be replaced: mdid cannot
// see it, and a new uid would shadow it.
func TestMergeKeyUIDIsRefused(t *testing.T) {
	const inherited = "---\ndefaults: &d\n  uid: 11111111-1111-4111-8111-111111111111\n<<: *d\n---\n# Content"

	t.Run("HasUID reports the merge", func(t *testing.T) {
		if _, err := HasUID(inherited); !errors.Is(err, ErrMergeKey) {
			t.Errorf("HasUID() error = %v, want ErrMergeKey", err)
		}
	})

	t.Run("ProcessContent refuses instead of adding one", func(t *testing.T) {
		got, err := ProcessContent(inherited)
		if !errors.Is(err, ErrMergeKey) {
			t.Fatalf("ProcessContent() error = %v, want ErrMergeKey", err)
		}
		if strings.Contains(got, "uid:") && !strings.Contains(got, "11111111-1111-4111-8111-111111111111") {
			t.Errorf("ProcessContent() returned an unexpected uid: %q", got)
		}
	})
}

// A literal uid overrides a merged one, so the merge does not apply.
func TestLiteralUIDWinsOverMergeKey(t *testing.T) {
	const withLiteral = "---\ndefaults: &d\n  uid: 11111111-1111-4111-8111-111111111111\nuid: 22222222-2222-4222-8222-222222222222\n<<: *d\n---\n# Content"

	has, err := HasUID(withLiteral)
	if err != nil {
		t.Fatalf("HasUID() error = %v", err)
	}
	if !has {
		t.Error("HasUID() = false, want true: a literal uid overrides a merged one")
	}
}

func TestMergeKeyFileIsLeftAlone(t *testing.T) {
	const inherited = "---\ndefaults: &d\n  uid: 11111111-1111-4111-8111-111111111111\n<<: *d\n---\n# Content"

	t.Run("ProcessFile leaves the file untouched", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "inherited.md")
		if err := os.WriteFile(path, []byte(inherited), 0o600); err != nil {
			t.Fatal(err)
		}
		before := mustReadFile(t, path)
		mtimeBefore := mustModTime(t, path)

		if err := ProcessFile(path); !errors.Is(err, ErrMergeKey) {
			t.Fatalf("ProcessFile() error = %v, want ErrMergeKey", err)
		}
		if after := mustReadFile(t, path); after != before {
			t.Error("ProcessFile() rewrote a file whose uid comes from a merge key")
		}
		if mustModTime(t, path) != mtimeBefore {
			t.Error("ProcessFile() changed the mtime of a file it refused to process")
		}
	})
}

func TestProcessContentFillsPlaceholderUID(t *testing.T) {
	for _, tt := range []struct {
		name    string
		content string
	}{
		{name: "empty value", content: "---\nuid:\ntitle: Test\n---\n# Content"},
		{name: "quoted spaces", content: "---\nuid: \"   \"\n---\n# Content"},
		{name: "null", content: "---\nuid: null\n---\n# Content"},
		{name: "not a string", content: "---\nuid: 12345\n---\n# Content"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			has, err := HasUID(tt.content)
			if err != nil {
				t.Fatalf("HasUID() error = %v", err)
			}
			if has {
				t.Fatal("HasUID() = true for a placeholder uid, want false")
			}

			got, err := ProcessContent(tt.content)
			if err != nil {
				t.Fatalf("ProcessContent() error = %v", err)
			}
			uid := getUIDFromContent(got)
			if !isValidV7UUID(uid) {
				t.Errorf("ProcessContent() uid = %q, want a real UUID v7", uid)
			}
		})
	}
}

func TestProcessContentAtTime(t *testing.T) {
	t.Run("embeds provided timestamp in uid", func(t *testing.T) {
		knownTime := time.Date(2024, 3, 10, 8, 0, 0, 0, time.UTC)
		got, err := ProcessContentAtTime(testFrontmatterInput, knownTime)
		if err != nil {
			t.Fatalf("ProcessContentAtTime() error = %v", err)
		}
		uid := getUIDFromContent(got)
		ts, err := parseV7Timestamp(uid)
		if err != nil {
			t.Fatalf("parseV7Timestamp() error = %v", err)
		}
		want := knownTime.Truncate(time.Millisecond).UTC()
		if !ts.Equal(want) {
			t.Errorf("uid timestamp = %v, want %v", ts, want)
		}
	})

	t.Run("does not overwrite existing uid", func(t *testing.T) {
		const existingUID = "11111111-1111-4111-8111-111111111111"
		input := "---\nuid: " + existingUID + "\ntitle: Test\n---\n# Content"
		got, err := ProcessContentAtTime(input, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("ProcessContentAtTime() error = %v", err)
		}
		if got != input {
			t.Error("ProcessContentAtTime() modified content that already had a uid")
		}
	})
}

// TestProcessDocument exercises the public FrontmatterDocument entry points
// against the real *mdfm.Document, which is the implementation callers get.
func TestProcessDocument(t *testing.T) {
	t.Run("adds uid when missing", func(t *testing.T) {
		doc := mustParse(t, testFrontmatterInput)
		if err := ProcessDocument(doc); err != nil {
			t.Fatalf("ProcessDocument() error = %v", err)
		}
		uid := getUIDFromContent(mustBytes(t, doc))
		if !isValidV7UUID(uid) {
			t.Errorf("ProcessDocument() uid = %q, want valid UUID v7", uid)
		}
	})

	t.Run("does not overwrite existing uid", func(t *testing.T) {
		const existingUID = "11111111-1111-4111-8111-111111111111"
		doc := mustParse(t, "---\nuid: "+existingUID+"\n---\n# Content")
		before := mustBytes(t, doc)
		if err := ProcessDocument(doc); err != nil {
			t.Fatalf("ProcessDocument() error = %v", err)
		}
		if got := mustBytes(t, doc); got != before {
			t.Errorf("ProcessDocument() overwrote existing uid: %q", got)
		}
	})

	t.Run("fills in a placeholder uid", func(t *testing.T) {
		doc := mustParse(t, "---\nuid:\n---\n# Content")
		if err := ProcessDocument(doc); err != nil {
			t.Fatalf("ProcessDocument() error = %v", err)
		}
		if uid := getUIDFromContent(mustBytes(t, doc)); !isValidV7UUID(uid) {
			t.Errorf("ProcessDocument() uid = %q, want a real UUID v7", uid)
		}
	})
}

func TestProcessDocumentAtTime(t *testing.T) {
	t.Run("embeds provided timestamp in uid", func(t *testing.T) {
		knownTime := time.Date(2024, 3, 10, 8, 0, 0, 0, time.UTC)
		doc := mustParse(t, testFrontmatterInput)
		if err := ProcessDocumentAtTime(doc, knownTime); err != nil {
			t.Fatalf("ProcessDocumentAtTime() error = %v", err)
		}
		ts, err := parseV7Timestamp(getUIDFromContent(mustBytes(t, doc)))
		if err != nil {
			t.Fatalf("parseV7Timestamp() error = %v", err)
		}
		want := knownTime.Truncate(time.Millisecond).UTC()
		if !ts.Equal(want) {
			t.Errorf("uid timestamp = %v, want %v", ts, want)
		}
	})

	t.Run("does not overwrite existing uid", func(t *testing.T) {
		const existingUID = "11111111-1111-4111-8111-111111111111"
		doc := mustParse(t, "---\nuid: "+existingUID+"\n---\n# Content")
		before := mustBytes(t, doc)
		if err := ProcessDocumentAtTime(doc, time.Now()); err != nil {
			t.Fatalf("ProcessDocumentAtTime() error = %v", err)
		}
		if got := mustBytes(t, doc); got != before {
			t.Errorf("ProcessDocumentAtTime() overwrote existing uid: %q", got)
		}
	})
}

func testProcessFileAddsUID(t *testing.T, tmpDir string) {
	t.Helper()

	t.Run("adds uid to file without frontmatter", func(t *testing.T) {
		f := filepath.Join(tmpDir, "nofm.md")
		if err := os.WriteFile(f, []byte("# Hello"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := ProcessFile(f); err != nil {
			t.Fatalf("ProcessFile() error = %v", err)
		}
		if got := mustReadFile(t, f); !strings.Contains(got, "uid:") {
			t.Error("ProcessFile() did not add uid")
		}
	})

	t.Run("adds uid to file with frontmatter", func(t *testing.T) {
		f := filepath.Join(tmpDir, "withfm.md")
		content := "---\ntitle: Test Document\n---\n# Hello World\n\nSome content."
		if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := ProcessFile(f); err != nil {
			t.Fatalf("ProcessFile() error = %v", err)
		}
		gotStr := mustReadFile(t, f)
		if !strings.Contains(gotStr, "uid:") {
			t.Error("ProcessFile() did not add uid")
		}
		if !strings.Contains(gotStr, "title: Test Document") {
			t.Error("ProcessFile() lost original frontmatter")
		}
		if !strings.Contains(gotStr, "# Hello World") {
			t.Error("ProcessFile() lost body content")
		}
	})
}

func testProcessFileFillsPlaceholderUID(t *testing.T, tmpDir string) {
	t.Helper()

	t.Run("replaces a placeholder uid", func(t *testing.T) {
		f := filepath.Join(tmpDir, "placeholder.md")
		if err := os.WriteFile(f, []byte("---\nuid:\n---\n# Content"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := ProcessFile(f); err != nil {
			t.Fatalf("ProcessFile() error = %v", err)
		}
		if uid := getUIDFromContent(mustReadFile(t, f)); !isValidV7UUID(uid) {
			t.Errorf("ProcessFile() left a placeholder uid in place: %q", uid)
		}
	})
}

func testProcessFileUsesMtime(t *testing.T, tmpDir string) {
	t.Helper()

	t.Run("uses file mtime as uid timestamp", func(t *testing.T) {
		f := filepath.Join(tmpDir, "mtime_test.md")
		if err := os.WriteFile(f, []byte("# Hello mtime"), 0o600); err != nil {
			t.Fatal(err)
		}
		// Set a known past mtime so we can verify it was used.
		knownTime := time.Date(2024, 11, 5, 14, 30, 0, 0, time.UTC)
		if err := os.Chtimes(f, knownTime, knownTime); err != nil {
			t.Fatal(err)
		}
		if err := ProcessFile(f); err != nil {
			t.Fatalf("ProcessFile() error = %v", err)
		}
		uid := getUIDFromContent(mustReadFile(t, f))
		ts, err := parseV7Timestamp(uid)
		if err != nil {
			t.Fatalf("parseV7Timestamp() error = %v", err)
		}
		// UUID v7 has millisecond precision.
		want := knownTime.Truncate(time.Millisecond).UTC()
		if !ts.Equal(want) {
			t.Errorf("uid timestamp = %v, want file mtime %v", ts, want)
		}
	})
}

func testProcessFilePreservesExisting(t *testing.T, tmpDir string) {
	t.Helper()

	t.Run("does not modify file with existing uid", func(t *testing.T) {
		f := filepath.Join(tmpDir, "hasuid.md")
		content := "---\nuid: 22222222-2222-4222-8222-222222222222\ntitle: Test\n---\n# Content"
		if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		info1, _ := os.Stat(f)
		if err := ProcessFile(f); err != nil {
			t.Fatalf("ProcessFile() error = %v", err)
		}
		info2, _ := os.Stat(f)
		if info1.ModTime() != info2.ModTime() {
			t.Error("ProcessFile() wrote file even though uid already present")
		}
	})
}

func testProcessFileErrors(t *testing.T, tmpDir string) {
	t.Helper()

	t.Run("nonexistent file returns error", func(t *testing.T) {
		err := ProcessFile(filepath.Join(tmpDir, "nonexistent.md"))
		if err == nil {
			t.Error("ProcessFile() expected error for nonexistent file")
		}
		if !strings.Contains(err.Error(), "failed to stat file") {
			t.Errorf("ProcessFile() error = %v, want 'failed to stat file' error", err)
		}
	})

	t.Run("invalid markdown returns error", func(t *testing.T) {
		f := filepath.Join(tmpDir, "invalid.md")
		if err := os.WriteFile(f, []byte("---\ntitle: Test\n# Missing closing delimiter"), 0o600); err != nil {
			t.Fatal(err)
		}
		err := ProcessFile(f)
		if err == nil {
			t.Error("ProcessFile() expected error for invalid markdown")
		}
		if !strings.Contains(err.Error(), "failed to process content") {
			t.Errorf("ProcessFile() error = %v, want 'failed to process content' error", err)
		}
	})

	t.Run("symlink file returns error", func(t *testing.T) {
		target := filepath.Join(tmpDir, "target.md")
		link := filepath.Join(tmpDir, "link.md")
		if err := os.WriteFile(target, []byte("# Symlink target"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("os.Symlink not supported: %v", err)
		}

		err := ProcessFile(link)
		if err == nil {
			t.Error("ProcessFile() expected error for symlink")
		}
		if !strings.Contains(err.Error(), "symlink") {
			t.Errorf("ProcessFile() error = %v, want symlink error", err)
		}
	})
}

func TestProcessFile(t *testing.T) {
	tmpDir := t.TempDir()
	testProcessFileAddsUID(t, tmpDir)
	testProcessFileFillsPlaceholderUID(t, tmpDir)
	testProcessFileUsesMtime(t, tmpDir)
	testProcessFilePreservesExisting(t, tmpDir)
	testProcessFileErrors(t, tmpDir)
}

func TestHasUID(t *testing.T) {
	t.Run("returns true when uid is present", func(t *testing.T) {
		const existingUID = "11111111-1111-4111-8111-111111111111"
		input := "---\nuid: " + existingUID + "\ntitle: Test\n---\n# Content"
		has, err := HasUID(input)
		if err != nil {
			t.Fatalf("HasUID() error = %v", err)
		}
		if !has {
			t.Error("HasUID() = false, want true for content with uid")
		}
	})

	t.Run("returns false when no uid", func(t *testing.T) {
		has, err := HasUID(testFrontmatterInput)
		if err != nil {
			t.Fatalf("HasUID() error = %v", err)
		}
		if has {
			t.Error("HasUID() = true, want false for content without uid")
		}
	})

	t.Run("returns false when no frontmatter", func(t *testing.T) {
		has, err := HasUID("# Just content, no frontmatter")
		if err != nil {
			t.Fatalf("HasUID() error = %v", err)
		}
		if has {
			t.Error("HasUID() = true, want false for content without frontmatter")
		}
	})

	t.Run("returns error for malformed frontmatter", func(t *testing.T) {
		_, err := HasUID("---\ntitle: Test\n# Missing closing delimiter")
		if err == nil {
			t.Error("HasUID() expected error for malformed frontmatter")
		}
	})
}
