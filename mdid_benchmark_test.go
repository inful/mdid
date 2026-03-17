package mdid

import (
	"os"
	"path/filepath"
	"testing"
)

const benchmarkContentWithoutUID = `---
title: Benchmark Document
author: Jane Doe
tags:
  - go
  - markdown
---
# Hello World

This is benchmark content for mdid.
`

const benchmarkContentWithUID = `---
uid: 01974f2a-9c00-7abc-8def-0123456789ab
title: Benchmark Document
author: Jane Doe
tags:
  - go
  - markdown
---
# Hello World

This is benchmark content for mdid.
`

func BenchmarkGenerateUID(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = GenerateUID()
	}
}

func BenchmarkProcessContentAddUID(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_, err := ProcessContent(benchmarkContentWithoutUID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessContentExistingUID(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_, err := ProcessContent(benchmarkContentWithUID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessFileAddUID(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	path := filepath.Join(dir, "add_uid.md")

	for b.Loop() {
		b.StopTimer()
		if err := os.WriteFile(path, []byte("# Hello World\n"), 0o600); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		if err := ProcessFile(path); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessFileExistingUID(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	path := filepath.Join(dir, "existing_uid.md")
	content := "---\nuid: 01974f2a-9c00-7abc-8def-0123456789ab\ntitle: Test\n---\n# Hello\n"

	for b.Loop() {
		b.StopTimer()
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		if err := ProcessFile(path); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessFileSymlinkRejected(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	target := filepath.Join(dir, "target.md")
	link := filepath.Join(dir, "link.md")
	if err := os.WriteFile(target, []byte("# Target\n"), 0o600); err != nil {
		b.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		b.Skipf("os.Symlink not supported: %v", err)
	}

	for b.Loop() {
		if err := ProcessFile(link); err == nil {
			b.Fatal("ProcessFile() expected symlink rejection error")
		}
	}
}

func BenchmarkWriteFileDirect(b *testing.B) {
	b.ReportAllocs()

	dir := b.TempDir()
	path := filepath.Join(dir, "direct.md")
	data := []byte(benchmarkContentWithoutUID)

	for b.Loop() {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			b.Fatal(err)
		}
	}
}
