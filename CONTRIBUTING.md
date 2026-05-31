# Contributing

Thanks for helping improve `ripemd160mb`. This guide covers the local workflow
and the checks that must pass before a change is merged. The same checks run in
CI (see [.github/workflows/ci.yml](.github/workflows/ci.yml)).

## Prerequisites

- Go 1.22 or newer.
- `staticcheck` for linting:
  `go install honnef.co/go/tools/cmd/staticcheck@2024.1.1`.

## Everyday checks

Run these before opening a pull request:

```sh
gofmt -l .                                # must print nothing
go vet ./...
staticcheck ./...

go test ./...                             # native backend
GORIPEMD160MB_FORCE=scalar go test ./...  # scalar oracle
go test -race ./...                       # race detector
```

## Coverage

The importable library packages (root and `hash160`) are held to a 95%
statement-coverage floor; they currently sit at 100%.

```sh
go test -covermode=atomic -coverprofile=coverage.out ./ ./hash160
go tool cover -func=coverage.out | tail -1
go tool cover -html=coverage.out          # find uncovered lines
```

New code in these packages should keep coverage at or above the floor. The NEON
generator (`internal/neongen`) and the runnable example are excluded from the
coverage gate; they are validated by `go vet`, the cross-build job, and the
`go generate` clean-tree check instead.

## Fuzzing

Every fast path is differentially fuzzed against `golang.org/x/crypto`:

```sh
go test -run '^$' -fuzz '^FuzzSum$'        -fuzztime=30s .
go test -run '^$' -fuzz '^FuzzHash32$'     -fuzztime=30s .
go test -run '^$' -fuzz '^FuzzHash160_32$' -fuzztime=30s ./hash160
```

If fuzzing finds a failure it writes a reproducer under `testdata/fuzz/`. Commit
that file with your fix so the case becomes a permanent regression test.

## Benchmarks

See [PERFORMANCE.md](PERFORMANCE.md) for the full methodology. In short:

```sh
go test -run '^$' -bench '^BenchmarkHash32$' -benchmem ./
```

Validate performance claims with `benchstat`, not single runs.

## Generated assembly — do not hand-edit

The arm64 NEON kernel is generated. The generator source is
[`internal/neongen/`](internal/neongen), and the `go:generate` directive lives
in [`generate.go`](generate.go). Regenerate with:

```sh
go generate ./...
```

Do not edit `block_arm64.s` by hand — change `internal/neongen` and rerun
`go generate`. CI runs `go generate ./...` followed by `git diff --exit-code`,
so any drift between the generator and the committed output fails the build.

Any new vector kernel (for example an amd64 SSE2/AVX2/AVX-512 backend) must be
verified bit-for-bit against the scalar oracle by the differential and fuzz
tests before it is wired into dispatch. A backend must never report a SIMD name
via `Backend()` while actually running the scalar fallback.

## Pull request checklist

- [ ] `gofmt`, `go vet`, and `staticcheck` are clean.
- [ ] `go test ./...`, the forced-scalar run, and `go test -race ./...` pass.
- [ ] Library coverage is at or above 95%.
- [ ] `go generate ./...` produces no diff.
- [ ] New behavior has tests; new fast-path behavior is also fuzzed.
- [ ] Performance claims are backed by `benchstat` output.
- [ ] User-facing changes update the relevant docs and `CHANGELOG.md`.
