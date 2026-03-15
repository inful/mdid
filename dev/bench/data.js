window.BENCHMARK_DATA = {
  "lastUpdate": 1773536534812,
  "repoUrl": "https://github.com/inful/mdid",
  "entries": {
    "mdid Go Benchmarks": [
      {
        "commit": {
          "author": {
            "email": "inful@users.noreply.github.com",
            "name": "Jone Marius Vignes",
            "username": "inful"
          },
          "committer": {
            "email": "inful@users.noreply.github.com",
            "name": "Jone Marius Vignes",
            "username": "inful"
          },
          "distinct": true,
          "id": "162dd684e06564a25709dbcd43e52a9cdf51ef05",
          "message": "chore: enable dependabot updates",
          "timestamp": "2026-03-15T00:54:18Z",
          "tree_id": "94fb333a692ee7f76093a1f500f8fb8706069d24",
          "url": "https://github.com/inful/mdid/commit/162dd684e06564a25709dbcd43e52a9cdf51ef05"
        },
        "date": 1773536079661,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkParseMarkdown",
            "value": 60.01,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "19818541 times\n4 procs"
          },
          {
            "name": "BenchmarkParseMarkdown - ns/op",
            "value": 60.01,
            "unit": "ns/op",
            "extra": "19818541 times\n4 procs"
          },
          {
            "name": "BenchmarkParseMarkdown - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "19818541 times\n4 procs"
          },
          {
            "name": "BenchmarkParseMarkdown - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "19818541 times\n4 procs"
          },
          {
            "name": "BenchmarkGenerateUID",
            "value": 215.9,
            "unit": "ns/op\t      64 B/op\t       2 allocs/op",
            "extra": "5526646 times\n4 procs"
          },
          {
            "name": "BenchmarkGenerateUID - ns/op",
            "value": 215.9,
            "unit": "ns/op",
            "extra": "5526646 times\n4 procs"
          },
          {
            "name": "BenchmarkGenerateUID - B/op",
            "value": 64,
            "unit": "B/op",
            "extra": "5526646 times\n4 procs"
          },
          {
            "name": "BenchmarkGenerateUID - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "5526646 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentAddUID",
            "value": 478,
            "unit": "ns/op\t     352 B/op\t       4 allocs/op",
            "extra": "2506915 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentAddUID - ns/op",
            "value": 478,
            "unit": "ns/op",
            "extra": "2506915 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentAddUID - B/op",
            "value": 352,
            "unit": "B/op",
            "extra": "2506915 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentAddUID - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "2506915 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentExistingUID",
            "value": 120,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "10010976 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentExistingUID - ns/op",
            "value": 120,
            "unit": "ns/op",
            "extra": "10010976 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentExistingUID - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "10010976 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentExistingUID - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "10010976 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "inful@users.noreply.github.com",
            "name": "Jone Marius Vignes",
            "username": "inful"
          },
          "committer": {
            "email": "inful@users.noreply.github.com",
            "name": "Jone Marius Vignes",
            "username": "inful"
          },
          "distinct": true,
          "id": "a3df24040fefcccd6303eaac03199417a253f910",
          "message": "perf: reduce allocations in ProcessContent add-uid path",
          "timestamp": "2026-03-15T01:00:34Z",
          "tree_id": "8afa351cd49fc5efb09b7df6f7fcab202d66d83c",
          "url": "https://github.com/inful/mdid/commit/a3df24040fefcccd6303eaac03199417a253f910"
        },
        "date": 1773536457877,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkParseMarkdown",
            "value": 54.26,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "22182487 times\n4 procs"
          },
          {
            "name": "BenchmarkParseMarkdown - ns/op",
            "value": 54.26,
            "unit": "ns/op",
            "extra": "22182487 times\n4 procs"
          },
          {
            "name": "BenchmarkParseMarkdown - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "22182487 times\n4 procs"
          },
          {
            "name": "BenchmarkParseMarkdown - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "22182487 times\n4 procs"
          },
          {
            "name": "BenchmarkGenerateUID",
            "value": 178.6,
            "unit": "ns/op\t      64 B/op\t       2 allocs/op",
            "extra": "6709544 times\n4 procs"
          },
          {
            "name": "BenchmarkGenerateUID - ns/op",
            "value": 178.6,
            "unit": "ns/op",
            "extra": "6709544 times\n4 procs"
          },
          {
            "name": "BenchmarkGenerateUID - B/op",
            "value": 64,
            "unit": "B/op",
            "extra": "6709544 times\n4 procs"
          },
          {
            "name": "BenchmarkGenerateUID - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "6709544 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentAddUID",
            "value": 371.9,
            "unit": "ns/op\t     240 B/op\t       3 allocs/op",
            "extra": "3230835 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentAddUID - ns/op",
            "value": 371.9,
            "unit": "ns/op",
            "extra": "3230835 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentAddUID - B/op",
            "value": 240,
            "unit": "B/op",
            "extra": "3230835 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentAddUID - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3230835 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentExistingUID",
            "value": 102.3,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "11742808 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentExistingUID - ns/op",
            "value": 102.3,
            "unit": "ns/op",
            "extra": "11742808 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentExistingUID - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "11742808 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentExistingUID - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "11742808 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "73816+inful@users.noreply.github.com",
            "name": "Jone Marius Vignes",
            "username": "inful"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "2a2853c53ecd952f62b4b354592b4636d767cf5e",
          "message": "Merge pull request #1 from inful/dependabot/github_actions/goreleaser/goreleaser-action-7\n\nchore(deps): bump goreleaser/goreleaser-action from 6 to 7",
          "timestamp": "2026-03-15T02:01:35+01:00",
          "tree_id": "329f87bc808986ad6caabe1d6c28062583a230cd",
          "url": "https://github.com/inful/mdid/commit/2a2853c53ecd952f62b4b354592b4636d767cf5e"
        },
        "date": 1773536516849,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkParseMarkdown",
            "value": 59.54,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "18237038 times\n4 procs"
          },
          {
            "name": "BenchmarkParseMarkdown - ns/op",
            "value": 59.54,
            "unit": "ns/op",
            "extra": "18237038 times\n4 procs"
          },
          {
            "name": "BenchmarkParseMarkdown - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "18237038 times\n4 procs"
          },
          {
            "name": "BenchmarkParseMarkdown - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "18237038 times\n4 procs"
          },
          {
            "name": "BenchmarkGenerateUID",
            "value": 181.7,
            "unit": "ns/op\t      64 B/op\t       2 allocs/op",
            "extra": "6594124 times\n4 procs"
          },
          {
            "name": "BenchmarkGenerateUID - ns/op",
            "value": 181.7,
            "unit": "ns/op",
            "extra": "6594124 times\n4 procs"
          },
          {
            "name": "BenchmarkGenerateUID - B/op",
            "value": 64,
            "unit": "B/op",
            "extra": "6594124 times\n4 procs"
          },
          {
            "name": "BenchmarkGenerateUID - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "6594124 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentAddUID",
            "value": 379.1,
            "unit": "ns/op\t     240 B/op\t       3 allocs/op",
            "extra": "3186364 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentAddUID - ns/op",
            "value": 379.1,
            "unit": "ns/op",
            "extra": "3186364 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentAddUID - B/op",
            "value": 240,
            "unit": "B/op",
            "extra": "3186364 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentAddUID - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3186364 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentExistingUID",
            "value": 102.8,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "11618901 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentExistingUID - ns/op",
            "value": 102.8,
            "unit": "ns/op",
            "extra": "11618901 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentExistingUID - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "11618901 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentExistingUID - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "11618901 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "73816+inful@users.noreply.github.com",
            "name": "Jone Marius Vignes",
            "username": "inful"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "2d8cf62bfffc3cb0629a0ef9946f19463db45d5c",
          "message": "Merge pull request #2 from inful/dependabot/github_actions/actions/cache-5\n\nchore(deps): bump actions/cache from 4 to 5",
          "timestamp": "2026-03-15T02:01:53+01:00",
          "tree_id": "5c0601212e55be1ee2dcb222d9c8593f7f60b531",
          "url": "https://github.com/inful/mdid/commit/2d8cf62bfffc3cb0629a0ef9946f19463db45d5c"
        },
        "date": 1773536534568,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkParseMarkdown",
            "value": 48.92,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "22664589 times\n4 procs"
          },
          {
            "name": "BenchmarkParseMarkdown - ns/op",
            "value": 48.92,
            "unit": "ns/op",
            "extra": "22664589 times\n4 procs"
          },
          {
            "name": "BenchmarkParseMarkdown - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "22664589 times\n4 procs"
          },
          {
            "name": "BenchmarkParseMarkdown - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "22664589 times\n4 procs"
          },
          {
            "name": "BenchmarkGenerateUID",
            "value": 215,
            "unit": "ns/op\t      64 B/op\t       2 allocs/op",
            "extra": "5561776 times\n4 procs"
          },
          {
            "name": "BenchmarkGenerateUID - ns/op",
            "value": 215,
            "unit": "ns/op",
            "extra": "5561776 times\n4 procs"
          },
          {
            "name": "BenchmarkGenerateUID - B/op",
            "value": 64,
            "unit": "B/op",
            "extra": "5561776 times\n4 procs"
          },
          {
            "name": "BenchmarkGenerateUID - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "5561776 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentAddUID",
            "value": 410.2,
            "unit": "ns/op\t     240 B/op\t       3 allocs/op",
            "extra": "2928088 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentAddUID - ns/op",
            "value": 410.2,
            "unit": "ns/op",
            "extra": "2928088 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentAddUID - B/op",
            "value": 240,
            "unit": "B/op",
            "extra": "2928088 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentAddUID - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "2928088 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentExistingUID",
            "value": 120.2,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "9861258 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentExistingUID - ns/op",
            "value": 120.2,
            "unit": "ns/op",
            "extra": "9861258 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentExistingUID - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "9861258 times\n4 procs"
          },
          {
            "name": "BenchmarkProcessContentExistingUID - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "9861258 times\n4 procs"
          }
        ]
      }
    ]
  }
}