// Package mdid provides functionality for adding unique identifiers
// to markdown files with YAML frontmatter.
package mdid

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FrontmatterDelimiter is the delimiter used for YAML frontmatter.
const FrontmatterDelimiter = "---"

// UIDField is the field name used in frontmatter for the unique identifier.
const UIDField = "uid"

const (
	filePermissions = 0o600
	growthOverhead  = 4
)

// UUID v7 byte layout constants (RFC 9562, Section 5.7).
const (
	uuidVersionByte = 6
	uuidVariantByte = 8
	uuidVersionMask = 0x0f
	uuidVersion7    = 0x70
	uuidVariantMask = 0x3f
	uuidVariantRFC  = 0x80
	msShift40       = 40
	msShift32       = 32
	msShift24       = 24
	msShift16       = 16
	msShift8        = 8
)

// ParseMarkdown extracts the frontmatter and body from a markdown file.
func ParseMarkdown(content string) (frontmatter string, body string, err error) {
	if !strings.HasPrefix(content, FrontmatterDelimiter+"\n") {
		return "", content, nil
	}

	rest := content[len(FrontmatterDelimiter)+1:]

	if strings.HasPrefix(rest, FrontmatterDelimiter+"\n") {
		body = rest[len(FrontmatterDelimiter)+1:]
		return "", body, nil
	}

	idx := strings.Index(rest, "\n"+FrontmatterDelimiter+"\n")
	if idx == -1 {
		return "", "", errors.New("unclosed frontmatter block")
	}

	frontmatter = rest[:idx]
	body = rest[idx+len(FrontmatterDelimiter)+2:]
	return frontmatter, body, nil
}

// GenerateUID returns a new UUID v7 using the current time as the timestamp.
func GenerateUID() string {
	return GenerateUIDAtTime(time.Now())
}

// GenerateUIDAtTime returns a new UUID v7 using t as the millisecond-precision
// timestamp. The embedded timestamp makes UIDs time-sortable while retaining
// global uniqueness through random bits.
func GenerateUIDAtTime(t time.Time) string {
	u := uuid.New()             // 16 cryptographically random bytes; panics on CSPRNG failure
	ms := uint64(t.UnixMilli()) //nolint:gosec // timestamp is always non-negative for modern files
	u[0] = byte(ms >> msShift40)
	u[1] = byte(ms >> msShift32)
	u[2] = byte(ms >> msShift24)
	u[3] = byte(ms >> msShift16)
	u[4] = byte(ms >> msShift8)
	u[5] = byte(ms)
	u[uuidVersionByte] = (u[uuidVersionByte] & uuidVersionMask) | uuidVersion7
	u[uuidVariantByte] = (u[uuidVariantByte] & uuidVariantMask) | uuidVariantRFC
	return u.String()
}

// HasUID reports whether the frontmatter already contains a uid field.
func HasUID(frontmatter string) bool {
	if frontmatter == "" {
		return false
	}
	return strings.HasPrefix(frontmatter, UIDField+":") ||
		strings.Contains(frontmatter, "\n"+UIDField+":")
}

// GetUID extracts the uid value from the frontmatter.
// It returns an empty string if no uid field is present.
func GetUID(frontmatter string) string {
	const splitParts = 2
	for line := range strings.SplitSeq(frontmatter, "\n") {
		if strings.HasPrefix(line, UIDField+":") {
			parts := strings.SplitN(line, ":", splitParts)
			if len(parts) == splitParts {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// AddUIDToFrontmatter prepends a uid field to the frontmatter.
// The uid is placed at the top so it is easy to find.
func AddUIDToFrontmatter(frontmatter, uid string) string {
	frontmatter = strings.TrimRight(frontmatter, "\n")

	var b strings.Builder
	b.Grow(len(UIDField) + len(uid) + len(frontmatter) + growthOverhead)

	b.WriteString(UIDField)
	b.WriteString(": ")
	b.WriteString(uid)
	b.WriteString("\n")

	if frontmatter != "" {
		b.WriteString(frontmatter)
		b.WriteString("\n")
	}

	return b.String()
}

// ProcessContent adds a uid to the frontmatter if one is not already present,
// using the current time as the UUID v7 timestamp. If a uid already exists, the
// content is returned unchanged.
func ProcessContent(content string) (string, error) {
	return ProcessContentAtTime(content, time.Now())
}

// ProcessContentAtTime adds a uid to the frontmatter if one is not already
// present, embedding t as the UUID v7 timestamp. If a uid already exists, the
// content is returned unchanged.
func ProcessContentAtTime(content string, t time.Time) (string, error) {
	frontmatter, body, err := ParseMarkdown(content)
	if err != nil {
		return "", err
	}

	if HasUID(frontmatter) {
		return content, nil
	}

	uid := GenerateUIDAtTime(t)
	frontmatter = AddUIDToFrontmatter(frontmatter, uid)

	var b strings.Builder
	b.Grow(len(FrontmatterDelimiter)*2 + len(frontmatter) + len(body) + growthOverhead)

	b.WriteString(FrontmatterDelimiter)
	b.WriteString("\n")
	b.WriteString(frontmatter)
	b.WriteString(FrontmatterDelimiter)
	b.WriteString("\n")
	b.WriteString(body)

	return b.String(), nil
}

// ProcessFile reads a markdown file, adds a uid if missing, and writes it back.
// The uid timestamp is derived from the file's modification time so that
// documents are time-sortable by when they were last edited. If the file
// already contains a uid it is left untouched.
func ProcessFile(filepath string) error {
	info, err := os.Stat(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	content, err := os.ReadFile(filepath) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	processed, err := ProcessContentAtTime(string(content), info.ModTime())
	if err != nil {
		return fmt.Errorf("failed to process content: %w", err)
	}

	if processed == string(content) {
		return nil
	}

	if err = os.WriteFile(filepath, []byte(processed), filePermissions); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
