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

// FrontmatterDocument is the minimal interface required to add a uid to a
// document's frontmatter. *mdfm.Document satisfies this interface.
type FrontmatterDocument interface {
	Has(key string) (bool, error)
	SetString(key, value string) error
}

// frontmatterCodec is the full set of operations mdid performs on a parsed
// markdown document. *mdfm.Document satisfies this interface; tests can
// substitute other implementations to exercise error paths without depending
// on the mdfm library's internal failure modes.
type frontmatterCodec interface {
	FrontmatterDocument
	Bytes() ([]byte, error)
}

// parseFrontmatter parses markdown content into a document codec. It is a
// package variable so tests can inject failing implementations; production
// callers should treat it as opaque.
var parseFrontmatter = func(content string) (frontmatterCodec, error) {
	return mdfm.ParseString(content)
}

// updateFrontmatterFile reads path, applies update, and writes the result
// back. It is a package variable so tests can inject failing implementations;
// production callers should treat it as opaque.
var updateFrontmatterFile = func(path string, update func(frontmatterCodec) error) error {
	return mdfm.UpdateFile(path, func(doc *mdfm.Document) error {
		return update(doc)
	})
}

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
//
//nolint:gosec // UUID v7 byte layout per RFC 9562 §5.7; byte casts below are intentional truncation.
func GenerateUIDAtTime(t time.Time) string {
	u := uuid.New()             // 16 cryptographically random bytes; panics on CSPRNG failure
	ms := uint64(t.UnixMilli()) // timestamp is always non-negative for modern files
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

// HasUID reports whether the markdown content's frontmatter contains a uid
// field. It returns false when the content has no frontmatter or the
// frontmatter has no uid. It returns an error only when the content has
// malformed YAML frontmatter.
func HasUID(content string) (bool, error) {
	doc, err := parseFrontmatter(content)
	if err != nil {
		return false, err
	}
	return doc.Has(UIDField)
}

// ProcessDocument adds a uid to doc's frontmatter if one is not already
// present, using the current time as the UUID v7 timestamp. The document is
// mutated in place.
func ProcessDocument(doc FrontmatterDocument) error {
	return ProcessDocumentAtTime(doc, time.Now())
}

// ProcessDocumentAtTime adds a uid to doc's frontmatter if one is not already
// present, embedding t as the UUID v7 timestamp. The document is mutated
// in place.
func ProcessDocumentAtTime(doc FrontmatterDocument, t time.Time) error {
	hasUID, err := doc.Has(UIDField)
	if err != nil {
		return err
	}
	if hasUID {
		return nil
	}

	return doc.SetString(UIDField, GenerateUIDAtTime(t))
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
	doc, err := parseFrontmatter(content)
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

	if err = doc.SetString(UIDField, GenerateUIDAtTime(t)); err != nil {
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
	err = updateFrontmatterFile(path, func(doc frontmatterCodec) error {
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
