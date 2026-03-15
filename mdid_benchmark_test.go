package mdid

import "testing"

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

func BenchmarkParseMarkdown(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_, _, err := ParseMarkdown(benchmarkContentWithoutUID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

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
