// Package mdid provides functionality for adding unique identifiers
// to markdown files with YAML frontmatter.
package mdid

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/inful/mdfm"
)

// UIDField is the field name used in frontmatter for the unique identifier.
const UIDField = "uid"

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
	doc, err := mdfm.ParseString(content)
	if err != nil {
		return "", err
	}

	hasUID, err := doc.Has(UIDField)
	if err != nil {
		return "", err
	}
	if hasUID {
		return content, nil
	}

	uid := GenerateUIDAtTime(t)
	if err = doc.SetString(UIDField, uid); err != nil {
		return "", err
	}

	out, err := doc.Bytes()
	if err != nil {
		return "", err
	}

	return string(out), nil
}

// ProcessFile reads a markdown file, adds a uid if missing, and writes it back.
// The uid timestamp is derived from the file's modification time so that
// documents are time-sortable by when they were last edited. If the file
// already contains a uid it is left untouched.
func ProcessFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to process symlink: %s", path)
	}

	uidTime := info.ModTime()
	err = mdfm.UpdateFile(path, func(doc *mdfm.Document) error {
		hasUID, hasErr := doc.Has(UIDField)
		if hasErr != nil {
			return hasErr
		}
		if hasUID {
			return nil
		}

		return doc.SetString(UIDField, GenerateUIDAtTime(uidTime))
	})
	if err != nil {
		return fmt.Errorf("failed to process content: %w", err)
	}

	return nil
}
