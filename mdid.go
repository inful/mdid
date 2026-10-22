// SPDX-License-Identifier: GPL-3.0-or-later

// Package mdid provides functionality for adding unique identifiers
// to markdown files with YAML frontmatter.
package mdid

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/inful/mdfm"
)

// UIDField is the field name used in frontmatter for the unique identifier.
const UIDField = "uid"

// mergeKey is the YAML merge key: frontmatter may pull keys in from an anchor
// with "<<: *anchor".
const mergeKey = "<<"

// ErrMergeKey reports frontmatter that uses a YAML merge key with no literal
// uid of its own. The merged keys hold a uid that only a full YAML parse would
// reveal, so mdid refuses to guess instead of adding an identifier that would
// shadow the inherited one.
var ErrMergeKey = errors.New("frontmatter uses a YAML merge key; cannot tell whether it already has a uid")

// FrontmatterDocument is the minimal interface required to add a uid to a
// document's frontmatter. *mdfm.Document satisfies this interface.
type FrontmatterDocument interface {
	Has(key string) (bool, error)
	Get(key string) (any, bool, error)
	SetString(key, value string) error
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

// uuidHexDigits is the lookup table for byte → 2-char hex encoding.
const uuidHexDigits = "0123456789abcdef"

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
	var u [16]byte
	// crypto/rand.Read goes straight to the OS CSPRNG (getrandom /
	// BCryptGenRandom) and, since Go 1.24, never returns an error. We call it
	// directly rather than through a function variable so the local u stays on
	// the stack — a function-variable indirection would force the compiler to
	// assume the slice may be retained and promote u to the heap. Only the
	// random bits in bytes 6-15 are used; bytes 0-5 are overwritten by the
	// timestamp below.
	_, _ = rand.Read(u[:])

	// The timestamp field is 48 unsigned bits. A time before 1970 (a zero
	// time.Time, a pre-epoch file mtime) has no place there, so clamp it to the
	// epoch instead of wrapping into the far future.
	ms := uint64(max(t.UnixMilli(), 0))
	u[0] = byte(ms >> msShift40)
	u[1] = byte(ms >> msShift32)
	u[2] = byte(ms >> msShift24)
	u[3] = byte(ms >> msShift16)
	u[4] = byte(ms >> msShift8)
	u[5] = byte(ms)
	u[uuidVersionByte] = (u[uuidVersionByte] & uuidVersionMask) | uuidVersion7
	u[uuidVariantByte] = (u[uuidVariantByte] & uuidVariantMask) | uuidVariantRFC

	return formatUUIDv7(u)
}

// formatUUIDv7 formats a 16-byte UUID into its canonical 36-char hex form
// (8-4-4-4-12 with dashes). The implementation is hand-rolled to avoid
// uuid.UUID.String()'s fmt.Sprintf overhead, which formats each 2/4-byte
// group via reflection. On a microbenchmark of the formatting step in
// isolation the hand-rolled version is several times faster; on the full
// GenerateUID path the CSPRNG syscall dominates and the end-to-end speedup
// is in the low double-digit percent range. The byte slice is constructed
// in a stack-allocated [36]byte and returned as a string, so the only heap
// allocation is the resulting 36-byte string itself.
func formatUUIDv7(u [16]byte) string {
	var b [36]byte
	b[8] = '-'
	b[13] = '-'
	b[18] = '-'
	b[23] = '-'

	b[0] = uuidHexDigits[u[0]>>4]
	b[1] = uuidHexDigits[u[0]&0x0f]
	b[2] = uuidHexDigits[u[1]>>4]
	b[3] = uuidHexDigits[u[1]&0x0f]
	b[4] = uuidHexDigits[u[2]>>4]
	b[5] = uuidHexDigits[u[2]&0x0f]
	b[6] = uuidHexDigits[u[3]>>4]
	b[7] = uuidHexDigits[u[3]&0x0f]
	b[9] = uuidHexDigits[u[4]>>4]
	b[10] = uuidHexDigits[u[4]&0x0f]
	b[11] = uuidHexDigits[u[5]>>4]
	b[12] = uuidHexDigits[u[5]&0x0f]
	b[14] = uuidHexDigits[u[6]>>4]
	b[15] = uuidHexDigits[u[6]&0x0f]
	b[16] = uuidHexDigits[u[7]>>4]
	b[17] = uuidHexDigits[u[7]&0x0f]
	b[19] = uuidHexDigits[u[8]>>4]
	b[20] = uuidHexDigits[u[8]&0x0f]
	b[21] = uuidHexDigits[u[9]>>4]
	b[22] = uuidHexDigits[u[9]&0x0f]
	b[24] = uuidHexDigits[u[10]>>4]
	b[25] = uuidHexDigits[u[10]&0x0f]
	b[26] = uuidHexDigits[u[11]>>4]
	b[27] = uuidHexDigits[u[11]&0x0f]
	b[28] = uuidHexDigits[u[12]>>4]
	b[29] = uuidHexDigits[u[12]&0x0f]
	b[30] = uuidHexDigits[u[13]>>4]
	b[31] = uuidHexDigits[u[13]&0x0f]
	b[32] = uuidHexDigits[u[14]>>4]
	b[33] = uuidHexDigits[u[14]&0x0f]
	b[34] = uuidHexDigits[u[15]>>4]
	b[35] = uuidHexDigits[u[15]&0x0f]

	return string(b[:])
}

// HasUID reports whether the markdown content's frontmatter contains a uid
// field with a value. It returns false when the content has no frontmatter, no
// uid, or an empty uid — a placeholder key is not an identifier. It returns an
// error only when the content has malformed YAML frontmatter.
func HasUID(content string) (bool, error) {
	doc, err := mdfm.ParseString(content)
	if err != nil {
		return false, err
	}
	return hasUID(doc)
}

// hasUID reports whether doc already carries a usable uid. A key that exists
// but holds nothing usable (empty, null, a number) counts as missing, so a
// placeholder gets a real identifier instead of being reported as done.
func hasUID(doc FrontmatterDocument) (bool, error) {
	present, err := doc.Has(UIDField)
	if err != nil {
		return false, err
	}
	if !present {
		// A merge key can supply the uid we cannot see. Adding one here would
		// shadow the inherited identifier, so refuse instead of guessing.
		merged, mergeErr := doc.Has(mergeKey)
		if mergeErr != nil {
			return false, mergeErr
		}
		if merged {
			return false, ErrMergeKey
		}

		return false, nil
	}

	// A usable uid is a non-empty string. `uid: null` decodes to a nil value
	// and `uid: 12345` to a number; neither is an identifier, so both fall
	// through and get a real one.
	value, _, err := doc.Get(UIDField)
	if err != nil {
		return false, err
	}

	text, ok := value.(string)

	return ok && strings.TrimSpace(text) != "", nil
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
	has, err := hasUID(doc)
	if err != nil {
		return err
	}
	if has {
		return nil
	}

	return doc.SetString(UIDField, GenerateUIDAtTime(t))
}

// ProcessContent adds a uid to the frontmatter if one is not already present,
// using the current time as the UUID v7 timestamp. If a usable uid already
// exists, the content is returned unchanged.
func ProcessContent(content string) (string, error) {
	return ProcessContentAtTime(content, time.Now())
}

// ProcessContentAtTime adds a uid to the frontmatter if one is not already
// present, embedding t as the UUID v7 timestamp. If a usable uid already
// exists, the content is returned unchanged.
func ProcessContentAtTime(content string, t time.Time) (string, error) {
	doc, err := mdfm.ParseString(content)
	if err != nil {
		return "", err
	}

	has, hasErr := hasUID(doc)
	if hasErr != nil {
		return "", hasErr
	}
	if has {
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
// already contains a usable uid it is left untouched.
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
		has, hasErr := hasUID(doc)
		if hasErr != nil {
			return hasErr
		}
		if has {
			return nil
		}

		return doc.SetString(UIDField, GenerateUIDAtTime(uidTime))
	})
	if err != nil {
		return fmt.Errorf("failed to process content: %w", err)
	}

	return nil
}
