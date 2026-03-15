package mdid

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// testFrontmatterInput is a minimal markdown document used across multiple tests.
const testFrontmatterInput = "---\ntitle: Test\n---\n# Content"

// uuidV7Re matches a canonical UUID v7 string.
var uuidV7Re = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func isValidV7UUID(s string) bool {
	return uuidV7Re.MatchString(s)
}

// parseV7Timestamp extracts the millisecond timestamp embedded in a UUID v7.
func parseV7Timestamp(uid string) (time.Time, error) {
	u, err := uuid.Parse(uid)
	if err != nil {
		return time.Time{}, err
	}
	if u.Version() != 7 {
		return time.Time{}, fmt.Errorf("expected UUID v7, got v%d", u.Version())
	}
	ms := int64(u[0])<<40 | int64(u[1])<<32 | int64(u[2])<<24 |
		int64(u[3])<<16 | int64(u[4])<<8 | int64(u[5])
	return time.UnixMilli(ms).UTC(), nil
}

func TestParseMarkdown(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantFrontmatter string
		wantBody        string
		wantErr         bool
	}{
		{
			name:            "with frontmatter",
			input:           "---\ntitle: Test\n---\n# Hello World",
			wantFrontmatter: "title: Test",
			wantBody:        "# Hello World",
		},
		{
			name:            "no frontmatter",
			input:           "# Hello World\nContent here",
			wantFrontmatter: "",
			wantBody:        "# Hello World\nContent here",
		},
		{
			name:    "unclosed frontmatter",
			input:   "---\ntitle: Test\n# Hello World",
			wantErr: true,
		},
		{
			name:            "empty frontmatter",
			input:           "---\n---\n# Content",
			wantFrontmatter: "",
			wantBody:        "# Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFrontmatter, gotBody, err := ParseMarkdown(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMarkdown() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if gotFrontmatter != tt.wantFrontmatter {
				t.Errorf("ParseMarkdown() frontmatter = %q, want %q", gotFrontmatter, tt.wantFrontmatter)
			}
			if gotBody != tt.wantBody {
				t.Errorf("ParseMarkdown() body = %q, want %q", gotBody, tt.wantBody)
			}
		})
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

func TestHasUID(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter string
		want        bool
	}{
		{name: "has uid at start", frontmatter: "uid: abc\ntitle: Test", want: true},
		{name: "has uid mid", frontmatter: "title: Test\nuid: abc\nauthor: Jane", want: true},
		{name: "no uid", frontmatter: "title: Test\nauthor: Jane", want: false},
		{name: "empty frontmatter", frontmatter: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasUID(tt.frontmatter); got != tt.want {
				t.Errorf("HasUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetUID(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter string
		want        string
	}{
		{name: "uid present", frontmatter: "uid: abc-123\ntitle: Test", want: "abc-123"},
		{name: "uid with extra spaces", frontmatter: "uid:   spaced\ntitle: Test", want: "spaced"},
		{name: "no uid", frontmatter: "title: Test", want: ""},
		{name: "empty frontmatter", frontmatter: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetUID(tt.frontmatter); got != tt.want {
				t.Errorf("GetUID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAddUIDToFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter string
		uid         string
		want        string
	}{
		{
			name:        "add to existing frontmatter",
			frontmatter: "title: Test\nauthor: John",
			uid:         "abc-123",
			want:        "uid: abc-123\ntitle: Test\nauthor: John\n",
		},
		{
			name:        "add to empty frontmatter",
			frontmatter: "",
			uid:         "abc-123",
			want:        "uid: abc-123\n",
		},
		{
			name:        "add to frontmatter with trailing newline",
			frontmatter: "title: Test\n",
			uid:         "xyz-789",
			want:        "uid: xyz-789\ntitle: Test\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AddUIDToFrontmatter(tt.frontmatter, tt.uid)
			if got != tt.want {
				t.Errorf("AddUIDToFrontmatter() = %q, want %q", got, tt.want)
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
		frontmatter, _, _ := ParseMarkdown(got)
		if !isValidV7UUID(GetUID(frontmatter)) {
			t.Errorf("ProcessContent() uid is not a valid UUID v7: %q", GetUID(frontmatter))
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
		frontmatter, _, _ := ParseMarkdown(got)
		if GetUID(frontmatter) != existingUID {
			t.Errorf("ProcessContent() uid = %q, want %q", GetUID(frontmatter), existingUID)
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
		if !strings.Contains(got, "---") {
			t.Error("ProcessContent() should add frontmatter delimiters")
		}
	})

	t.Run("returns error for unclosed frontmatter", func(t *testing.T) {
		_, err := ProcessContent("---\ntitle: Test\n# Missing closing delimiter")
		if err == nil {
			t.Error("ProcessContent() expected error for unclosed frontmatter")
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

func TestProcessContentAtTime(t *testing.T) {
	t.Run("embeds provided timestamp in uid", func(t *testing.T) {
		knownTime := time.Date(2024, 3, 10, 8, 0, 0, 0, time.UTC)
		got, err := ProcessContentAtTime(testFrontmatterInput, knownTime)
		if err != nil {
			t.Fatalf("ProcessContentAtTime() error = %v", err)
		}
		fm, _, _ := ParseMarkdown(got)
		uid := GetUID(fm)
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

	t.Run("returns error for unclosed frontmatter", func(t *testing.T) {
		_, err := ProcessContentAtTime("---\ntitle: Test\n# no close", time.Now())
		if err == nil {
			t.Error("ProcessContentAtTime() expected error for unclosed frontmatter")
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
		got, _ := os.ReadFile(f) //nolint:gosec
		if !strings.Contains(string(got), "uid:") {
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
		got, _ := os.ReadFile(f) //nolint:gosec
		gotStr := string(got)
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
		got, _ := os.ReadFile(f) //nolint:gosec
		fm, _, _ := ParseMarkdown(string(got))
		uid := GetUID(fm)
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
	testProcessFileUsesMtime(t, tmpDir)
	testProcessFilePreservesExisting(t, tmpDir)
	testProcessFileErrors(t, tmpDir)
}
